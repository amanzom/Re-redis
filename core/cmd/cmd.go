package cmd

import (
	"errors"

	"github.com/amanzom/re-redis/core/resp"
)

type RedisCmd struct {
	Cmd  string
	Args []string
}

func GetRedisCmdObjects(buffer []byte, n int) ([]*RedisCmd, error) {
	// the command will of the form list of(array of bulk strings) since we need to cater pipelining case as well
	commandsArrayInterface, err := resp.Decode(buffer[:n])
	if err != nil {
		return nil, err
	}

	commandsArray, ok := commandsArrayInterface.([]interface{})
	if !ok {
		return nil, errors.New("error typecasting commands array interface to array")
	}

	var redisCmds []*RedisCmd
	for _, commandsInterface := range commandsArray {
		commands, ok := commandsInterface.([]interface{})
		if !ok {
			return nil, errors.New("error typecasting commands interface to array")
		}
		var elems []string
		for _, commandInterface := range commands {
			command, ok := commandInterface.(string)
			if !ok {
				return nil, errors.New("error typecasting command interface to command")
			}
			elems = append(elems, command)
		}

		if len(elems) == 0 {
			// log error here?
			continue
		}
		cmd := elems[0]
		var args []string
		if len(elems) > 1 {
			args = elems[1:]
		}
		redisCmds = append(redisCmds, &RedisCmd{
			Cmd:  cmd,
			Args: args,
		})
	}
	return redisCmds, nil
}
