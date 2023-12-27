package server

import (
	"io"

	"github.com/amanzom/re-redis/core/cmd"
	"github.com/amanzom/re-redis/core/eval"
)

type Server interface {
	StartServer()
}

func readCommands(conn io.ReadWriter) ([]*cmd.RedisCmd, error) {
	// TODO: Max read in one shot is 512 bytes
	// To allow input > 512 bytes, then repeated read until
	// we get EOF or designated delimiter

	buffer := make([]byte, 512)
	n, err := conn.Read(buffer[:])
	if err != nil {
		return nil, err
	}
	return cmd.GetRedisCmdObjects(buffer, n)
}

func respond(cmds []*cmd.RedisCmd, conn io.ReadWriter) error {
	buffer := eval.EvalCmds(cmds)
	if _, err := conn.Write(buffer); err != nil {
		return err
	}
	return nil
}
