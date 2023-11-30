package eval

import (
	"errors"

	"github.com/amanzom/re-redis/core/cmd"
	"github.com/amanzom/re-redis/core/constants"
	"github.com/amanzom/re-redis/core/resp"
)

func EvalCmd(cmd *cmd.RedisCmd) ([]byte, error) {
	switch cmd.Cmd {
	case constants.Ping:
		return evalPing(cmd.Args)
	default:
		// temp default handling
		return evalPing(cmd.Args)
	}
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
