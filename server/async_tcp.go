package server

import (
	"net"
	"syscall"
	"time"

	"github.com/amanzom/re-redis/core/comm"
	"github.com/amanzom/re-redis/core/logger"
	"github.com/amanzom/re-redis/core/store"
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

var autoDeletionCronLastExecTime = time.Now()
var autoDeletionCronFreq = 1 * time.Second

// using kqueue in go: https://dev.to/frosnerd/writing-a-simple-tcp-server-using-kqueue-cah
func (s *AsyncTcpServer) StartServer() {
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
	// creating kernel event queue
	kqueueFD, err := syscall.Kqueue()
	if err != nil {
		logger.Error("failed creating kqueue fd, err: %v", err)
		return
	}
	defer syscall.Close(kqueueFD)

	// registering the fds(in this case socketFD) whose change events we want to listen to.
	// since we want to subscribe for incoming connection events over socket.
	changeEvent := syscall.Kevent_t{
		Ident:  uint64(socketFD),    // fd we want to suscribe to
		Filter: syscall.EVFILT_READ, // Filter that processes the event. Set to EVFILT_READ,
		// which, when used in combination with a listening socket, indicates that we are interested in incoming connection events.
		Flags: syscall.EV_ADD | syscall.EV_ENABLE, // Flags that indicate what actions to perform with this event. In our case
		// we want to add the event to kqueue (EV_ADD), i.e. subscribing to it, and enable it (EV_ENABLE).
		Fflags: 0,
		Data:   0,
		Udata:  nil,
	}
	changeEventRegistered, err := syscall.Kevent(
		kqueueFD,
		[]syscall.Kevent_t{changeEvent},
		nil,
		nil,
	)
	if err != nil || changeEventRegistered == -1 {
		logger.Error("failed registering change event to kqueue, err: %v", err)
		return
	}

	// event loop pooling
	clientsConnected := 0
	for {
		// auto deletion of expired keys
		if time.Now().After(autoDeletionCronLastExecTime.Add(autoDeletionCronFreq)) {
			store.DeleteExpiredKeys()
			autoDeletionCronLastExecTime = time.Now()
		}

		// when registered fds are ready for IO, will be pushed into kqueue which are polled out in newEvents
		newEvents := make([]syscall.Kevent_t, maxClients)
		numNewEvents, err := syscall.Kevent( // blocking call till any of the fds are available for IO
			kqueueFD,
			nil,
			newEvents,
			nil,
		)
		if err != nil {
			logger.Error("failed pooling events from kqueue, err: %v", err)
			continue
		}

		for i := 0; i < numNewEvents; i++ {
			currentEvent := newEvents[i]
			currentEventFD := int(currentEvent.Ident)

			if currentEvent.Flags&syscall.EV_EOF != 0 { // client closing connection
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

				// registering/subscribing to socketConnectionFD over kqueue for change events
				// to be able to process requests over this tcp connection
				socketEvent := syscall.Kevent_t{
					Ident:  uint64(socketConnectionFD),
					Filter: syscall.EVFILT_READ,
					Flags:  syscall.EV_ADD | syscall.EV_ENABLE,
					Fflags: 0,
					Data:   0,
					Udata:  nil,
				}
				socketEventRegistered, err := syscall.Kevent(
					kqueueFD,
					[]syscall.Kevent_t{socketEvent},
					nil,
					nil,
				)
				if err != nil || socketEventRegistered == -1 {
					logger.Error("failed registering socket change event to kqueue, err: %v", err)
					syscall.Close(socketConnectionFD)
					continue
				}

				clientsConnected++
				logger.Info("new client connected with total concurrent clients: %v", clientsConnected)

			} else { // request to read some data over socket
				comm := &comm.FDComm{Fd: currentEventFD}
				cmds, err := readCommands(comm)
				if err != nil {
					clientsConnected--
					logger.Info("Closing connection on: %v, %v with total concurrent clients: %v", s.Host, s.Port, clientsConnected)
					syscall.Close(currentEventFD)
					continue
				}
				logger.Info("cmds from client: %v", cmds)
				err = respond(cmds, comm)
				if err != nil {
					logger.Error("error writing response at host, port: %v, %v, err: %v", s.Host, s.Port, err.Error())
				}
			}
		}
	}
}
