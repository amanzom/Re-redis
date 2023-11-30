package server

import (
	"fmt"
	"io"
	"net"
	"strconv"

	"github.com/amanzom/re-redis/core/cmd"
	"github.com/amanzom/re-redis/core/eval"
)

type SyncTcpServer struct {
	Host string
	Port int
}

func NewSyncTcpServer(host string, port int) *SyncTcpServer {
	return &SyncTcpServer{
		Host: host,
		Port: port,
	}
}

func (s *SyncTcpServer) StartSyncTcpServer() {
	fmt.Println("Starting new sync tcp server at host: ", s.Host, " Port: ", s.Port)

	clientsConnected := 0
	// server listening over the host port
	list, err := net.Listen("tcp", s.Host+":"+strconv.Itoa(s.Port))
	if err != nil {
		fmt.Errorf("error in configuring server to listen at %v, %v, err: %v", s.Host, s.Port, err.Error())
		return
	}

	for {
		// this will be a blocking call till some client connects over the network
		conn, err := list.Accept()
		if err != nil {
			fmt.Errorf("error in accepting connection at %v, %v, err: %v", s.Host, s.Port, err.Error())
			panic(err)
		}
		clientsConnected++
		fmt.Println("New client connected with address: ", conn.RemoteAddr(), " with total concurrent clients ", clientsConnected)

		for {
			// over the socket, continuously read the command and print it out
			cmd, err := readCommand(conn)
			if err != nil {
				if err == io.EOF {
					clientsConnected--
					fmt.Println("Closing connection on ", s.Host, s.Port, " for client with address ", conn.RemoteAddr(), " with total concurrent clients ", clientsConnected)
					conn.Close()
					break
				}
				fmt.Errorf("Error writing reading for client with address: %v, on: %v, %v, err: %v", conn.RemoteAddr(), s.Host, s.Port, err.Error())
				continue
			}
			fmt.Println("cmd from client: ", cmd)

			err = respond(cmd, conn)
			if err != nil {
				fmt.Errorf("Error writing response for client with address: %v, on: %v, %v, err: %v", conn.RemoteAddr(), s.Host, s.Port, err.Error())
			}
		}
	}
}

func readCommand(conn net.Conn) (*cmd.RedisCmd, error) {
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

func respond(cmd *cmd.RedisCmd, conn net.Conn) error {
	buffer, err := eval.EvalCmd(cmd)
	if err != nil {
		buffer = []byte(fmt.Sprintf("-%v\r\n", err))
	}
	if _, err = conn.Write(buffer); err != nil {
		return err
	}
	return nil
}
