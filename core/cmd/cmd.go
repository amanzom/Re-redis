package cmd

import (
	"errors"

	"github.com/amanzom/re-redis/core/resp"
)

type RedisCmd struct {
	Cmd  string
	Args []string
}

func GetRedisCmdObject(buffer []byte, n int) (*RedisCmd, error) {
	// the command will of the form array of bulk strings
	valueInterface, err := resp.Decode(buffer)
	if err != nil {
		return nil, err
	}

	valueArray, ok := valueInterface.([]interface{})
	if !ok {
		return nil, errors.New("error typecasting interface to array from buffer")
	}

	var elems []string
	for _, elem := range valueArray {
		valueString, ok := elem.(string)
		if !ok {
			return nil, errors.New("error typecasting interface to string from interface array")
		}
		elems = append(elems, valueString)
	}

	if len(elems) == 0 {
		return nil, errors.New("no cmd provided from client")
	}

	cmd := elems[0]
	var args []string
	if len(elems) > 1 {
		args = elems[1:]
	}

	return &RedisCmd{
		Cmd:  cmd,
		Args: args,
	}, nil
}
