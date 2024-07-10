package server

import (
	"net"
	"syscall"
	"time"

	"github.com/amanzom/re-redis/core"
	"github.com/amanzom/re-redis/core/iomultiplexer"
	"github.com/amanzom/re-redis/pkg/logger"
)

const (
	maxClients = 2000
)

type AsyncTcpServer struct {
	Host string
	Port int
}

func NewAsyncTcpServer(host string, port int) Server {
	return &AsyncTcpServer{
		Host: host,
		Port: port,
	}
}

var cronLastExecTime = time.Now()
var cronFreq = 1 * time.Second

func (s *AsyncTcpServer) StartServer() {
	// marking engine status as shutting down on function termination
	defer core.MarEngineStatusShuttingDown()

	logger.Info("Starting new async tcp server at host: %v and port: %v", s.Host, s.Port)
	// creating the socket
	// params description:
	// syscall.AF_INET - represents we are using ipv4 address family
	// syscall.SOCK_STREAM - represents 2 way communication i.e. streams
	// protocol we want to use: Protocol 0 in SOCK_STREAM sockets corresponds to TCP.
	socketFD, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
	if err != nil {
		logger.Error("error creating the socket, err: %v", err)
		return
	}
	defer syscall.Close(socketFD)

	// setting socket in non blocking mode
	// needed for handling multiple incoming requests over the socket i.e. the socket will not block for new requests if its processing
	// the previous one. In blocking mode the new requests would have been refused, while processing for some previous request is still going on.
	// Note: we are just accepting the new requests which will go into the socket listen queue(requests here could be either making a new tcp conn
	// or requesting/sending data over a tcp conn over this socket), not processing them.  Processing these requests will happen based on whether we
	// are ready for that particular IO. Concurrent processing of requests is handled below by IO multiplexing.
	if err := syscall.SetNonblock(socketFD, true); err != nil {
		logger.Error("error setting socket in non blocking mode, err: %v", err)
		return
	}

	// binding the host and port address for this socket for incoming requests over this address
	ip4 := net.ParseIP(s.Host)
	if err := syscall.Bind(socketFD, &syscall.SockaddrInet4{
		Port: s.Port,
		Addr: [4]byte{ip4[0], ip4[1], ip4[2], ip4[3]},
	}); err != nil {
		logger.Error("error binding the address, err: %v", err)
		return
	}

	// start listening for incoming requests
	// setting backlog - 2000: represent maximum no of pending requests that can go in socket's listen queue.
	// if the number of incoming requests over the socket exceeds the backlog, additional request attempts may be refused.
	if err := syscall.Listen(socketFD, maxClients); err != nil {
		logger.Error("error listening over %v, %v, err: %v", s.Host, s.Port, err)
		return
	}

	// IO multiplexing starts here
	ioMultiplexer, err := iomultiplexer.New(maxClients)
	if err != nil {
		logger.Error("failed creating kqueue fd, err: %v", err)
		return
	}
	defer ioMultiplexer.Close()

	// subscribing over socketFD for incoming connections over socket
	err = ioMultiplexer.Subscribe(socketFD)
	if err != nil {
		logger.Error(err.Error())
	}

	// event loop pooling till engine not in shutting down state
	clientsConnected := 0
	for !core.IsEngineStatusShuttingDown() {
		// executing crons
		if time.Now().After(cronLastExecTime.Add(cronFreq)) {
			// auto deletion of expired keys
			core.DeleteExpiredKeys()

			// syncing aof file from buffer
			if err := core.TriggerAofWriteFromBuffer(); err != nil {
				logger.Info("error writing to aof from buffer: %v", err)
			}
			cronLastExecTime = time.Now()
		}

		// polling for ready fds for i/o
		newEvents, err := ioMultiplexer.Poll()
		if err != nil {
			logger.Error("failed pooling events from kqueue, err: %v", err)
			continue
		}

		// mark engine status busy before reading fds to prevent shutdown in
		// bw servering requests, if unable to transition from waiting to busy
		// it means engine was in some other state - if shutting down state then return from here
		if !core.MarkEngineStatusBusy() {
			if core.IsEngineStatusShuttingDown() {
				return
			}
		}

		for _, currentEvent := range newEvents {
			currentEventFD := int(currentEvent.Fd)

			if currentEvent.CloseConnection { // client closing connection
				core.MarkClientDisconnected(currentEventFD)
				clientsConnected--
				logger.Info("Closing connection on: %v, %v with total concurrent clients: %v", s.Host, s.Port, clientsConnected)
				syscall.Close(currentEventFD)
			} else if currentEventFD == socketFD { // new tcp connection request over socket
				// accepting the incoming tcp connection
				socketConnectionFD, _, err := syscall.Accept(currentEventFD)
				if err != nil {
					logger.Error("failed accepting client connection over socket, err: %v", err)
					continue
				}

				if err := syscall.SetNonblock(socketConnectionFD, true); err != nil {
					logger.Error("failed setting socket connection in non blocking mode, err: %v", err)
					syscall.Close(socketConnectionFD)
					continue
				}

				// subscribing over socketConnectionFD for incoming requests over connection
				err = ioMultiplexer.Subscribe(socketConnectionFD)
				if err != nil {
					logger.Error("failed registering socket change event to kqueue, err: %v", err)
					syscall.Close(socketConnectionFD)
					continue
				}

				// marking client connected
				core.MarkClientConnected(socketConnectionFD, core.NewClient(socketConnectionFD))

				clientsConnected++
				logger.Info("new client connected with total concurrent clients: %v", clientsConnected)

			} else { // request to read some data over socket
				client := core.GetConnectedClientFromFD(currentEventFD)
				cmds, err := readCommands(client)
				if err != nil {
					core.MarkClientDisconnected(currentEventFD)
					clientsConnected--
					logger.Info("Closing connection on: %v, %v with total concurrent clients: %v", s.Host, s.Port, clientsConnected)
					syscall.Close(currentEventFD)
					continue
				}
				printCommands(cmds)
				err = respond(cmds, client)
				if err != nil {
					logger.Error("error writing response at host, port: %v, %v, err: %v", s.Host, s.Port, err.Error())
				}
			}
		}

		// mark engine state waiting after processing fds requests
		core.MarkEngineStatusWaiting()
	}
}

func printCommands(cmds []*core.RedisCmd) {
	logger.Info("************* cmds from client start *************")
	for _, cmd := range cmds {
		logger.Info("cmd: %v, arg: %v", cmd.Cmd, cmd.Args)
	}
	logger.Info("************* cmds from client end *************")
}
