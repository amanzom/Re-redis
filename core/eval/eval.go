package eval

import (
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

func EvalCmd(cmd *cmd.RedisCmd) ([]byte, error) {
	switch strings.ToLower(cmd.Cmd) {
	case constants.Ping:
		return evalPing(cmd.Args)
	case constants.Set:
		return evalSet(cmd.Args)
	case constants.Get:
		return evalGet(cmd.Args)
	case constants.Ttl:
		return evalTtl(cmd.Args)
	default:
		return evalNotSupportedCmd(cmd.Cmd, cmd.Args)
	}
}

func evalNotSupportedCmd(cmd string, args []string) ([]byte, error) {
	return nil, fmt.Errorf("ERR unknown command: '%v', with args beginning with: '%v'", cmd, strings.Join(args, "', '"))
}

func evalPing(args []string) ([]byte, error) {
	// - if more than one args exists return error
	if len(args) > 1 {
		return nil, errors.New("ERR wrong number of arguments for 'ping' command")
	}
	// if args empty - respond with PONG
	if len(args) == 0 {
		return resp.Encode(constants.Pong, true)
	}
	// else - respond with the PONG 'first arg value' as a bulk string
	return resp.Encode(args[0], false)
}

func evalSet(args []string) ([]byte, error) {
	if len(args) <= 1 {
		return nil, errors.New("ERR wrong number of arguments for 'set' command")
	}

	// first 2 agrs as key and value
	key, val := args[0], args[1]
	var expiryInMs int64 = -1 // -1 will be treated as no expiry
	// last 2 elements treated as ex and expiresAtInMs
	for i := 2; i < len(args); i++ {
		if strings.ToLower(args[i]) == constants.EX {
			i++
			if i == len(args) {
				return nil, errors.New("ERR syntax error")
			}
			expiryInSec, err := strconv.ParseInt(args[i], 10, 64)
			if err != nil {
				return nil, errors.New("ERR value is not an integer or out of range")
			}

			expiryInMs = expiryInSec * 1000
		} else {
			return nil, errors.New("ERR syntax error")
		}
	}

	store.Put(key, store.NewObj(val, expiryInMs))
	return []byte(constants.RESP_OK), nil
}

func evalGet(args []string) ([]byte, error) {
	if len(args) != 1 {
		return nil, errors.New("ERR wrong number of arguments for 'get' command")
	}

	obj := store.Get(args[0]) // args[0] represents key
	if obj == nil || (obj.ExpiresAt != -1 && time.Now().UnixMilli() >= obj.ExpiresAt) {
		return []byte(constants.RESP_NIL), nil
	}
	return resp.Encode(obj.Value, false)
}

func evalTtl(args []string) ([]byte, error) {
	if len(args) != 1 {
		return nil, errors.New("ERR wrong number of arguments for 'ttl' command")
	}

	obj := store.Get(args[0])                                                           // args[0] represents key
	if obj == nil || (obj.ExpiresAt != -1 && time.Now().UnixMilli() >= obj.ExpiresAt) { // obj not found or has expired
		return []byte(":-2\r\n"), nil
	}
	if obj.ExpiresAt == -1 { // ttl not set
		return []byte(":-1\r\n"), nil
	}

	timeRemainingInSec := (obj.ExpiresAt - time.Now().UnixMilli()) / 1000
	return resp.Encode(timeRemainingInSec, false)
}
