package server

import (
	"fmt"
	"io"

	"github.com/amanzom/re-redis/core/cmd"
	"github.com/amanzom/re-redis/core/eval"
)

type Server interface {
	StartServer()
}

func readCommand(conn io.ReadWriter) (*cmd.RedisCmd, error) {
	// TODO: Max read in one shot is 512 bytes
	// To allow input > 512 bytes, then repeated read until
	// we get EOF or designated delimiter

	buffer := make([]byte, 512)
	n, err := conn.Read(buffer[:])
	if err != nil {
		return nil, err
	}
	return cmd.GetRedisCmdObject(buffer, n)
}

func respond(cmd *cmd.RedisCmd, conn io.ReadWriter) error {
	buffer, err := eval.EvalCmd(cmd)
	if err != nil {
		buffer = []byte(fmt.Sprintf("-%v\r\n", err))
	}
	if _, err = conn.Write(buffer); err != nil {
		return err
	}
	return nil
}
