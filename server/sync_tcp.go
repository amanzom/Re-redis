package server

import (
	"fmt"
	"io"
	"net"
	"strconv"
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
		fmt.Println("new client connected with address: ", conn.RemoteAddr(), " with total concurrent clients ", clientsConnected)

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
				fmt.Errorf("error reading for client with address: %v, on: %v, %v, err: %v", conn.RemoteAddr(), s.Host, s.Port, err.Error())
				continue
			}
			fmt.Println("cmd from client: ", cmd)

			err = respond(cmd, conn)
			if err != nil {
				fmt.Errorf("error writing response for client with address: %v, on: %v, %v, err: %v", conn.RemoteAddr(), s.Host, s.Port, err.Error())
			}
		}
	}
}
