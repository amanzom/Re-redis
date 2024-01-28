package iomultiplexer

type IoMultiplexerInterface interface {
	// register/subscribe the fds whose change events we want to listen to. ex: socketFD for accepting connections over socket
	// or socketConnectionFD for accepting requets over a connection
	Subscribe(eventFd int) error
	// polling for fds which are ready for i/o
	Poll() ([]*PolledEvent, error)
	// closing Io Multiplexer instance i.e. kqueue/epoll fds
	Close()
}

type PolledEvent struct {
	Fd              int
	CloseConnection bool
}
