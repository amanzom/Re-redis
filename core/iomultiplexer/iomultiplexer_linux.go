package iomultiplexer

import (
	"syscall"
)

type IoMultiplexerLinux struct {
	Fd         int
	MaxClients int
}

func New(maxClients int) (IoMultiplexerInterface, error) {
	// creating epoll instance
	fd, err := syscall.EpollCreate1(0)
	if err != nil {
		return nil, err
	}

	return &IoMultiplexerLinux{
		Fd:         fd,
		MaxClients: maxClients,
	}, nil
}

func (i *IoMultiplexerLinux) Subscribe(eventFd int) error {
	if err := syscall.EpollCtl(i.Fd, syscall.EPOLL_CTL_ADD, eventFd, &syscall.EpollEvent{
		Fd:     int32(eventFd),
		Events: syscall.EPOLLIN | syscall.EPOLLERR | syscall.EPOLLHUP | syscall.EPOLLRDHUP,
	}); err != nil {
		return err
	}
	return nil
}

func (i *IoMultiplexerLinux) Poll() ([]*PolledEvent, error) {
	newEvents := make([]syscall.EpollEvent, i.MaxClients)
	nEvents, err := syscall.EpollWait(i.Fd, newEvents, -1) // blocking call till any of the fds are available for IO
	if err != nil {
		return nil, err
	}

	polledEvents := make([]*PolledEvent, 0)
	for i := 0; i < nEvents; i++ {
		currentEvent := newEvents[i]
		polledEvents = append(polledEvents, &PolledEvent{
			Fd:              int(currentEvent.Fd),
			CloseConnection: (currentEvent.Events&syscall.EPOLLERR != 0) || (currentEvent.Events&syscall.EPOLLHUP != 0) || (currentEvent.Events&syscall.EPOLLRDHUP != 0),
		})
	}

	return polledEvents, nil
}

func (i *IoMultiplexerLinux) Close() {
	syscall.Close(i.Fd)
}
