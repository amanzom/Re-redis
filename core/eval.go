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

// Note for Txns:
// Tnx commands are queued regardless of whether they will fail
// when 'exec' is called. Final response after exec will return array of
// results - with errors for failed commands in that txn.
// TODO: add support for checking the error status before queueing the commands.

// Aof file is written using commands buffer which currently doesn't log the txn state.
// For the commands executed in txn, it may happen during reconstruction of store
// from aof some commands failed due to some reason and will result in an inconsistent state
// of store which should not happen for cmds executed in txn. Ideally, we would need to add
// support for logging txn state for commands executed in txn while writing them to aof so that
// inconsistent states can be avoided during reconstruction i.e. either use all the cmds
// of a txn or discard all of them during reconstruction.

var txnEndCommands = map[string]bool{exec: true, discard: true}

func EvalCmds(cmds []*RedisCmd, c *Client) []byte {
	var response []byte
	buf := bytes.NewBuffer(response)
	for _, cmd := range cmds {
		// client not in txn mode - return the result
		if !c.IsTxnMode() {
			buf.Write(executeCommand(cmd, c))
			continue
		}

		// in txn mode cases:
		// 1. commands to return txn results i.e. exec or discard execute -
		// return the res of client's txn queued commands and discard txn
		// 2. else queue the commands in client's txn queue
		if txnEndCommands[cmd.Cmd] {
			buf.Write(executeCommand(cmd, c))
		} else {
			// nested txn's not allowed, i.e. if already received multi - return error
			if cmd.Cmd == multi {
				buf.Write(Encode(errors.New("ERR MULTI calls can not be nested"), false))
				continue
			}
			c.EnqueueTxnCommand(cmd)
			buf.Write([]byte(resp_queued))
		}
	}
	return buf.Bytes()
}

func executeCommand(cmd *RedisCmd, c *Client) []byte {
	switch strings.ToLower(cmd.Cmd) {
	case ping:
		return evalPing(cmd.Args)
	case set:
		return evalSet(cmd.Args)
	case get:
		return evalGet(cmd.Args)
	case ttl:
		return evalTtl(cmd.Args)
	case expire:
		return evalExpire(cmd.Args)
	case del:
		return evalDel(cmd.Args)
	case bgWriteAof:
		return evalBgWriteAof(cmd.Args)
	case incr:
		return evalIncr(cmd.Args)
	case info:
		return evalInfo(cmd.Args)
	case client:
		return evalClient(cmd.Args)
	case latency:
		return evalLatency(cmd.Args)
	case multi:
		return evalMulti(cmd.Args, c)
	case exec:
		return evalExec(cmd.Args, c)
	case discard:
		return evalDiscard(cmd.Args, c)
	default:
		return evalNotSupportedCmd(cmd.Cmd, cmd.Args)
	}
}

func evalNotSupportedCmd(cmd string, args []string) []byte {
	return Encode(fmt.Errorf("ERR unknown command: '%v', with args beginning with: '%v'", cmd, strings.Join(args, "', '")), false)
}

func evalPing(args []string) []byte {
	// - if more than one args exists return error
	if len(args) > 1 {
		return Encode(errors.New("ERR wrong number of arguments for 'ping' command"), false)
	}
	// if args empty - respond with PONG
	if len(args) == 0 {
		return Encode(pong, true)
	}
	// else - respond with the PONG 'first arg value' as a bulk string
	return Encode(args[0], false)
}

func evalSet(args []string) []byte {
	if len(args) <= 1 {
		return Encode(errors.New("ERR wrong number of arguments for 'set' command"), false)
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
				return Encode(errors.New("ERR syntax error"), false)
			}
			expiryInSec, err = strconv.ParseInt(args[i], 10, 64)
			if err != nil {
				return Encode(errors.New("ERR value is not an integer or out of range"), false)
			}

			expiryInMs = expiryInSec * 1000
		} else {
			return Encode(errors.New("ERR syntax error"), false)
		}
	}

	// deducing object type encoding
	oType, oEnc := deduceTypeEncoding(val)

	PutInStore(key, NewObj(val, expiryInMs, oType, oEnc))
	if expiryInSec != -1 {
		// storing in commands buffer for aof writes periodically
		commandsBuffer.Write(getKeyValueExpireCommandRespEncodedBytes(key, val, int(expiryInSec)))
	}
	return []byte(resp_ok)
}

func evalGet(args []string) []byte {
	if len(args) != 1 {
		return Encode(errors.New("ERR wrong number of arguments for 'get' command"), false)
	}

	obj := GetFromStore(args[0]) // args[0] represents key
	if obj == nil {
		return []byte(resp_nil)
	}
	return Encode(obj.Value, false)
}

func evalTtl(args []string) []byte {
	if len(args) != 1 {
		return Encode(errors.New("ERR wrong number of arguments for 'ttl' command"), false)
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
	return Encode(timeRemainingInSec, false)
}

func evalExpire(args []string) []byte {
	if len(args) <= 1 {
		return Encode(errors.New("ERR wrong number of arguments for 'expire' command"), false)
	}

	expiryInSecs, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		return Encode(errors.New("ERR value is not an integer or out of range"), false)
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
		return Encode(errors.New("ERR wrong number of arguments for 'del' command"), false)
	}

	countDeleted := 0
	for _, key := range args {
		if isDeleted := DelFromStore(key); isDeleted {
			countDeleted++
		}
	}

	return Encode(countDeleted, false)
}

func evalBgWriteAof(args []string) []byte {
	// TODO: Fork a separate process for rewritting the aof file.
	err := dumpStoreSnapshotToAof()
	if err != nil {
		logger.Error(err.Error())
		return Encode(errors.New("ERR performing background rewrite of AOF"), false)
	}
	return []byte(resp_ok)
}

func evalIncr(args []string) []byte {
	if len(args) != 1 {
		return Encode(errors.New("ERR wrong number of arguments for 'incr' command"), false)
	}

	key := args[0]
	obj := GetFromStore(key)
	if obj == nil {
		PutInStore(key, NewObj("0", -1, ObjectTypeString, ObjectEncodingInt))
	}

	if !assertType(uint8(obj.TypeEncoding), ObjectTypeString) {
		return Encode(errors.New("ERR object type not supported for 'incr' command"), false)
	}
	if !assertEncoding(uint8(obj.TypeEncoding), ObjectEncodingInt) {
		return Encode(errors.New("ERR object encoding not supported for 'incr' command"), false)
	}

	i, err := strconv.ParseInt(obj.Value.(string), 10, 64)
	if err != nil {
		return Encode(errors.New("ERR unable to parse object value to integer for 'incr' command"), false)
	}

	i++
	val := strconv.FormatInt(i, 10)
	obj.Value = val
	// storing in commands buffer for aof writes periodically
	commandsBuffer.Write(getKeyValueSetCommandRespEncodedBytes(key, val))
	return Encode(i, false)
}

// we just provide no of keys present info under Keyspace stats
func evalInfo(args []string) []byte {
	var b []byte
	buf := bytes.NewBuffer(b)
	buf.Write([]byte("# Keyspace\r\n"))

	var numKeys int64 = 0
	if keyspaceStats[0] != nil {
		numKeys = keyspaceStats[0]["keys"]
	}

	buf.Write([]byte(fmt.Sprintf("db0:keys=%v,expires=0,avg_ttl=0\r\n", numKeys)))
	return Encode(buf.String(), false)
}

func evalClient(args []string) []byte {
	return []byte(resp_ok)
}

func evalLatency(args []string) []byte {
	return Encode([]string{}, false)
}

func evalMulti(args []string, c *Client) []byte {
	c.MarkClientInTxnMode()
	return []byte(resp_ok)
}

func evalExec(args []string, c *Client) []byte {
	if !c.IsTxnMode() {
		return Encode(errors.New("ERR EXEC without MULTI"), false)
	}
	return c.ExecuteTxnQueuedCommands()
}

func evalDiscard(args []string, c *Client) []byte {
	if !c.IsTxnMode() {
		return Encode(errors.New("ERR DISCARD without MULTI"), false)
	}
	c.DiscardTxn()
	return []byte(resp_ok)
}
