package core

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/amanzom/re-redis/config"
	"github.com/amanzom/re-redis/pkg/logger"
)

// Note for persisance of keys with expiration:
// 1. expiry related commands are logged on aof - in set with ex, expire commands, taking entire snapshot via background, active deletion of expired keys.
// 2. during background rewrite of aof from snapshot - expiried keys are not set in aof. - in future this will run frequently(every 1 sec), so will make a lot of expired keys to be skipped from aof.
// 3. during passive deletion of expired keys - we log a del command in aof.
// 4. active deletion of expired keys also handles deletion commmad logging in aof - will run 20 times in a sec in future so will handle a lot of left cases.
// 5. still cases would be left to handle some of the expired keys which were logged on aof, and are were not deleted - when we reconstruct store after downtime we may see such keys to reappear.

var commandsBuffer *bytes.Buffer // will be used for storing commands in buffer, and sync them in aof file periodically

// Process is similar to generating rdb file i.e take entire store snapshot and dump, but we only support aof file currently for simplicity
// This is not how aof files are generated w.r.t redis.
// TODO: instead of taking snapshot - design an algo to read previous aof file to generate new concised AOF file.
func dumpStoreSnapshotToAof() error {
	// to prevent race conditions once we start running this dump logic in background, not writing directly to main aof file
	logger.Info("Aof file rewrite initiated")

	// Open tempFile with read-write and create flags
	tempFile, err := os.OpenFile(config.TempAofFilePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModeAppend)
	if err != nil {
		return errors.New(fmt.Sprintf("error opening temp aof file, err: %v", err))
	}
	defer tempFile.Close()

	// setting permissions
	if err := os.Chmod(config.TempAofFilePath, os.ModeAppend|os.ModePerm|os.FileMode(0444)); err != nil {
		return errors.New(fmt.Sprintf("error setting permissions for temp aof file, err: %v", err))
	}

	// write the data to temp aof file
	for key, value := range store {
		if value.ExpiresAt < time.Now().UnixMilli() { // skipping if key expired
			continue
		}

		// set related
		_, err := tempFile.Write(getKeyValueSetCommandRespEncodedBytes(key, value.Value))
		if err != nil {
			return errors.New(fmt.Sprintf("error writing set command temp aof file, err: %v", err))
		}

		// expiry related
		if value.ExpiresAt != -1 { // expiry set
			expiryInSecs := (value.ExpiresAt - time.Now().UnixMilli()) / 1000
			_, err := tempFile.Write(getKeyValueExpireCommandRespEncodedBytes(key, value.Value, int(expiryInSecs)))
			if err != nil {
				return errors.New(fmt.Sprintf("error writing expire command to temp aof file, err: %v", err))
			}
		}
	}

	// Get the fileInfo for evaluating size of the file
	fileInfo, err := tempFile.Stat()
	if err != nil {
		return errors.New(fmt.Sprintf("error getting file info for temp aof file, err: %v", err))
	}

	// Read the content into a buffer
	buffer := make([]byte, fileInfo.Size())
	_, err = tempFile.ReadAt(buffer, 0)
	if err != nil {
		return errors.New(fmt.Sprintf("error reading temp aof file, err: %v", err))
	}

	// Open aofFile with read-write and create flags
	aofFile, err := os.OpenFile(config.AofFilePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModeAppend)
	if err != nil {
		return errors.New(fmt.Sprintf("error opening aof file, err: %v", err))
	}
	defer aofFile.Close()

	// set permissions
	if err := os.Chmod(config.AofFilePath, os.ModeAppend|os.ModePerm|os.FileMode(0444)); err != nil {
		return errors.New(fmt.Sprintf("error setting permissions for aof file, err: %v", err))
	}

	// Write the buffer to the main AOF file
	err = os.WriteFile(config.AofFilePath, buffer, os.ModeAppend)
	if err != nil {
		return errors.New(fmt.Sprintf("error rewriting main aof file from temp aof file, err: %v", err))
	}

	// Remove the temporary AOF file
	if err := os.Remove(config.TempAofFilePath); err != nil {
		return errors.New(fmt.Sprintf("error deleting temp aof file, err: %v", err))
	}

	logger.Info("Aof file rewrite complete")
	return nil
}

func TriggerAofWriteFromBuffer() error {
	logger.Info("Aof file sync from buffer initiated")

	file, err := os.OpenFile(config.AofFilePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, os.ModeAppend)
	if err != nil {
		return err
	}
	defer file.Close()

	if err := os.Chmod(config.AofFilePath, os.ModeAppend|os.ModePerm|os.FileMode(0444)); err != nil {
		return errors.New(fmt.Sprintf("error setting permissions for temp aof file, err: %v", err))
	}

	_, err = file.Write(commandsBuffer.Bytes())
	if err != nil {
		return nil
	}

	// re-initialise with empty buffer
	var b []byte
	commandsBuffer = bytes.NewBuffer(b)
	logger.Info("Aof file sync from buffer completed")
	return nil
}

func reconstructStoreFromAof() error {
	logger.Info("Starting store reconstruct on boot up")
	// Open aofFile with read-write and create flags
	aofFile, err := os.OpenFile(config.AofFilePath, os.O_RDONLY, os.ModeAppend)
	if err != nil {
		return errors.New(fmt.Sprintf("error opening aof file for reconstructing store, err: %v", err))
	}
	defer aofFile.Close()

	// set permissions
	if err := os.Chmod(config.AofFilePath, os.ModeAppend|os.ModePerm|os.FileMode(0444)); err != nil {
		return errors.New(fmt.Sprintf("error setting permissions for aof file, err: %v", err))
	}

	// Get the fileInfo for evaluating size of the file
	fileInfo, err := aofFile.Stat()
	if err != nil {
		return errors.New(fmt.Sprintf("error getting file info for temp aof file, err: %v", err))
	}

	// Read the content into a buffer
	n := fileInfo.Size()
	buffer := make([]byte, n)
	_, err = aofFile.ReadAt(buffer, 0)
	if err != nil {
		return errors.New(fmt.Sprintf("error reading temp aof file, err: %v", err))
	}

	// form redisCmd from buffer
	redisCmds, err := GetRedisCmdObjects(buffer, int(n))
	if err != nil {
		return errors.New(fmt.Sprintf("error creating redis commands during store reconstruct, err: %v", err))
	}

	// this internally will replay commands to reconstruct store and will put them in the commandsBuffer as well so reinitialise commandsBuffer
	EvalCmds(redisCmds)

	// re-initialise with empty buffer
	var b []byte
	commandsBuffer = bytes.NewBuffer(b)
	logger.Info("Store reconstruct on boot up successfull")
	return nil
}

// for a key, val in store - this function gives its corresponding resp command for set
func getKeyValueSetCommandRespEncodedBytes(key string, value interface{}) []byte {
	cmd := fmt.Sprintf("SET %v %v", key, value)
	cmdArr := strings.Split(cmd, " ")
	return encode(cmdArr, false)
}

// for a key, val in store - this function gives its corresponding resp command for delete
func getKeyValueDeleteCommandRespEncodedBytes(key string) []byte {
	cmd := fmt.Sprintf("DEL %v", key)
	cmdArr := strings.Split(cmd, " ")
	return encode(cmdArr, false)
}

// for a key, val, expiry in store - this function gives its corresponding resp command for expire
func getKeyValueExpireCommandRespEncodedBytes(key string, value interface{}, expiryInSec int) []byte {
	cmd := fmt.Sprintf("EXPIRE %v %v", key, expiryInSec)
	cmdArr := strings.Split(cmd, " ")
	return encode(cmdArr, false)
}
