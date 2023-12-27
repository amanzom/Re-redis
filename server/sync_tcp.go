package server

import (
	"io"
	"net"
	"strconv"

	"github.com/amanzom/re-redis/core/logger"
)

type SyncTcpServer struct {
	Host string
	Port int
}

func NewSyncTcpServer(host string, port int) Server {
	return &SyncTcpServer{
		Host: host,
		Port: port,
	}
}

func (s *SyncTcpServer) StartServer() {
	logger.Info("Starting new sync tcp server at host: %v, port: %v", s.Host, s.Port)

	clientsConnected := 0
	// server listening over the host port
	list, err := net.Listen("tcp", s.Host+":"+strconv.Itoa(s.Port))
	if err != nil {
		logger.Error("error in configuring server to listen at %v, %v, err: %v", s.Host, s.Port, err.Error())
		return
	}

	for {
		// this will be a blocking call till some client connects over the network
		conn, err := list.Accept()
		if err != nil {
			logger.Error("error in accepting connection at %v, %v, err: %v", s.Host, s.Port, err.Error())
			panic(err)
		}
		clientsConnected++
		logger.Info("new client connected with address: %v, with total concurrent clients: %v", conn.RemoteAddr(), clientsConnected)

		for {
			// over the socket, continuously read the command and print it out
			cmds, err := readCommands(conn)
			if err != nil {
				if err == io.EOF {
					clientsConnected--
					logger.Info("Closing connection on: %v, %v, for client with address: %v, with total concurrent clients: %v", s.Host, s.Port, conn.RemoteAddr(), clientsConnected)
					conn.Close()
					break
				}
				logger.Error("error reading for client with address: %v, on: %v, %v, err: %v", conn.RemoteAddr(), s.Host, s.Port, err.Error())
				continue
			}
			logger.Info("cmd from client: %v", cmds)

			err = respond(cmds, conn)
			if err != nil {
				logger.Error("error writing response for client with address: %v, on: %v, %v, err: %v", conn.RemoteAddr(), s.Host, s.Port, err.Error())
			}
		}
	}
}
