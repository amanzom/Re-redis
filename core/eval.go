package core

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/amanzom/re-redis/pkg/logger"
)

func EvalCmds(cmds []*RedisCmd) []byte {
	var response []byte
	buf := bytes.NewBuffer(response)
	for _, cmd := range cmds {
		switch strings.ToLower(cmd.Cmd) {
		case ping:
			buf.Write(evalPing(cmd.Args))
		case set:
			buf.Write(evalSet(cmd.Args))
		case get:
			buf.Write(evalGet(cmd.Args))
		case ttl:
			buf.Write(evalTtl(cmd.Args))
		case expire:
			buf.Write(evalExpire(cmd.Args))
		case del:
			buf.Write(evalDel(cmd.Args))
		case bgWriteAof:
			buf.Write(evalBgWriteAof(cmd.Args))
		default:
			buf.Write(evalNotSupportedCmd(cmd.Cmd, cmd.Args))
		}
	}
	return buf.Bytes()
}

func evalNotSupportedCmd(cmd string, args []string) []byte {
	return encode(fmt.Errorf("ERR unknown command: '%v', with args beginning with: '%v'", cmd, strings.Join(args, "', '")), false)
}

func evalPing(args []string) []byte {
	// - if more than one args exists return error
	if len(args) > 1 {
		return encode(errors.New("ERR wrong number of arguments for 'ping' command"), false)
	}
	// if args empty - respond with PONG
	if len(args) == 0 {
		return encode(pong, true)
	}
	// else - respond with the PONG 'first arg value' as a bulk string
	return encode(args[0], false)
}

func evalSet(args []string) []byte {
	if len(args) <= 1 {
		return encode(errors.New("ERR wrong number of arguments for 'set' command"), false)
	}

	// first 2 agrs as key and value
	key, val := args[0], args[1]
	var expiryInMs int64 = -1  // -1 will be treated as no expiry
	var expiryInSec int64 = -1 // -1 will be treated as no expiry
	var err error
	// last 2 elements treated as ex and expiresAtInMs
	for i := 2; i < len(args); i++ {
		if strings.ToLower(args[i]) == ex {
			i++
			if i == len(args) {
				return encode(errors.New("ERR syntax error"), false)
			}
			expiryInSec, err = strconv.ParseInt(args[i], 10, 64)
			if err != nil {
				return encode(errors.New("ERR value is not an integer or out of range"), false)
			}

			expiryInMs = expiryInSec * 1000
		} else {
			return encode(errors.New("ERR syntax error"), false)
		}
	}

	PutInStore(key, NewObj(val, expiryInMs))
	if expiryInSec != -1 {
		// storing in commands buffer for aof writes periodically
		commandsBuffer.Write(getKeyValueExpireCommandRespEncodedBytes(key, val, int(expiryInSec)))
	}
	return []byte(resp_ok)
}

func evalGet(args []string) []byte {
	if len(args) != 1 {
		return encode(errors.New("ERR wrong number of arguments for 'get' command"), false)
	}

	obj := GetFromStore(args[0]) // args[0] represents key
	if obj == nil {
		return []byte(resp_nil)
	}
	return encode(obj.Value, false)
}

func evalTtl(args []string) []byte {
	if len(args) != 1 {
		return encode(errors.New("ERR wrong number of arguments for 'ttl' command"), false)
	}

	key := args[0]
	obj := GetFromStore(args[0]) // args[0] represents key
	if obj == nil {              // obj not found or has expired
		return []byte(":-2\r\n")
	}
	if obj.ExpiresAt == -1 { // ttl not set
		return []byte(":-1\r\n")
	}

	timeRemainingInSec := (obj.ExpiresAt - time.Now().UnixMilli()) / 1000
	// storing in commands buffer for aof writes periodically
	commandsBuffer.Write(getKeyValueExpireCommandRespEncodedBytes(key, obj.Value, int(timeRemainingInSec)))
	return encode(timeRemainingInSec, false)
}

func evalExpire(args []string) []byte {
	if len(args) <= 1 {
		return encode(errors.New("ERR wrong number of arguments for 'expire' command"), false)
	}

	expiryInSecs, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		return encode(errors.New("ERR value is not an integer or out of range"), false)
	}

	key := args[0]
	val := GetFromStore(key)
	if val == nil { // key not present or has expired
		return []byte(":0\r\n")
	}

	val.ExpiresAt = time.Now().UnixMilli() + expiryInSecs*1000
	// storing in commands buffer for aof writes periodically
	commandsBuffer.Write(getKeyValueExpireCommandRespEncodedBytes(key, val, int(expiryInSecs)))
	return []byte(":1\r\n")
}

func evalDel(args []string) []byte {
	if len(args) == 0 {
		return encode(errors.New("ERR wrong number of arguments for 'del' command"), false)
	}

	countDeleted := 0
	for _, key := range args {
		if isDeleted := DelFromStore(key); isDeleted {
			countDeleted++
		}
	}

	return encode(countDeleted, false)
}

func evalBgWriteAof(args []string) []byte {
	// TODO: Fork a separate process for rewritting the aof file.
	err := dumpStoreSnapshotToAof()
	if err != nil {
		logger.Error(err.Error())
		return encode(errors.New("ERR performing background rewrite of AOF"), false)
	}
	return []byte(resp_ok)
}