package eval

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/amanzom/re-redis/core/cmd"
	"github.com/amanzom/re-redis/core/constants"
	"github.com/amanzom/re-redis/core/resp"
	"github.com/amanzom/re-redis/core/store"
)

func EvalCmds(cmds []*cmd.RedisCmd) []byte {
	var response []byte
	buf := bytes.NewBuffer(response)
	for _, cmd := range cmds {
		switch strings.ToLower(cmd.Cmd) {
		case constants.Ping:
			buf.Write(evalPing(cmd.Args))
		case constants.Set:
			buf.Write(evalSet(cmd.Args))
		case constants.Get:
			buf.Write(evalGet(cmd.Args))
		case constants.Ttl:
			buf.Write(evalTtl(cmd.Args))
		case constants.Expire:
			buf.Write(evalExpire(cmd.Args))
		case constants.Del:
			buf.Write(evalDel(cmd.Args))
		default:
			buf.Write(evalNotSupportedCmd(cmd.Cmd, cmd.Args))
		}
	}
	return buf.Bytes()
}

func evalNotSupportedCmd(cmd string, args []string) []byte {
	return resp.Encode(fmt.Errorf("ERR unknown command: '%v', with args beginning with: '%v'", cmd, strings.Join(args, "', '")), false)
}

func evalPing(args []string) []byte {
	// - if more than one args exists return error
	if len(args) > 1 {
		return resp.Encode(errors.New("ERR wrong number of arguments for 'ping' command"), false)
	}
	// if args empty - respond with PONG
	if len(args) == 0 {
		return resp.Encode(constants.Pong, true)
	}
	// else - respond with the PONG 'first arg value' as a bulk string
	return resp.Encode(args[0], false)
}

func evalSet(args []string) []byte {
	if len(args) <= 1 {
		return resp.Encode(errors.New("ERR wrong number of arguments for 'set' command"), false)
	}

	// first 2 agrs as key and value
	key, val := args[0], args[1]
	var expiryInMs int64 = -1 // -1 will be treated as no expiry
	// last 2 elements treated as ex and expiresAtInMs
	for i := 2; i < len(args); i++ {
		if strings.ToLower(args[i]) == constants.EX {
			i++
			if i == len(args) {
				return resp.Encode(errors.New("ERR syntax error"), false)
			}
			expiryInSec, err := strconv.ParseInt(args[i], 10, 64)
			if err != nil {
				return resp.Encode(errors.New("ERR value is not an integer or out of range"), false)
			}

			expiryInMs = expiryInSec * 1000
		} else {
			return resp.Encode(errors.New("ERR syntax error"), false)
		}
	}

	store.Put(key, store.NewObj(val, expiryInMs))
	return []byte(constants.RESP_OK)
}

func evalGet(args []string) []byte {
	if len(args) != 1 {
		return resp.Encode(errors.New("ERR wrong number of arguments for 'get' command"), false)
	}

	obj := store.Get(args[0]) // args[0] represents key
	if obj == nil {
		return []byte(constants.RESP_NIL)
	}
	return resp.Encode(obj.Value, false)
}

func evalTtl(args []string) []byte {
	if len(args) != 1 {
		return resp.Encode(errors.New("ERR wrong number of arguments for 'ttl' command"), false)
	}

	obj := store.Get(args[0]) // args[0] represents key
	if obj == nil {           // obj not found or has expired
		return []byte(":-2\r\n")
	}
	if obj.ExpiresAt == -1 { // ttl not set
		return []byte(":-1\r\n")
	}

	timeRemainingInSec := (obj.ExpiresAt - time.Now().UnixMilli()) / 1000
	return resp.Encode(timeRemainingInSec, false)
}

func evalExpire(args []string) []byte {
	if len(args) <= 1 {
		return resp.Encode(errors.New("ERR wrong number of arguments for 'expire' command"), false)
	}

	expiryInSecs, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		return resp.Encode(errors.New("ERR value is not an integer or out of range"), false)
	}

	val := store.Get(args[0])
	if val == nil { // key not present or has expired
		return []byte(":0\r\n")
	}

	val.ExpiresAt = time.Now().UnixMilli() + expiryInSecs*1000
	return []byte(":1\r\n")
}

func evalDel(args []string) []byte {
	if len(args) == 0 {
		return resp.Encode(errors.New("ERR wrong number of arguments for 'del' command"), false)
	}

	countDeleted := 0
	for _, key := range args {
		if isDeleted := store.Del(key); isDeleted {
			countDeleted++
		}
	}

	return resp.Encode(countDeleted, false)
}
