package server

import (
	"io"

	"github.com/amanzom/re-redis/core"
)

type Server interface {
	StartServer()
}

func readCommands(conn io.ReadWriter) ([]*core.RedisCmd, error) {
	// TODO: Max read in one shot is 512 bytes
	// To allow input > 512 bytes, then repeated read until
	// we get EOF or designated delimiter

	buffer := make([]byte, 512)
	n, err := conn.Read(buffer[:])
	if err != nil {
		return nil, err
	}
	return core.GetRedisCmdObjects(buffer, n)
}

func respond(cmds []*core.RedisCmd, conn io.ReadWriter) error {
	buffer := core.EvalCmds(cmds)
	if _, err := conn.Write(buffer); err != nil {
		return err
	}
	return nil
}
