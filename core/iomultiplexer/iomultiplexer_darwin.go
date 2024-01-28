package iomultiplexer

import (
	"errors"
	"fmt"
	"syscall"
)

type IoMultiplexerDarwin struct {
	Fd         int
	MaxClients int
}

// using kqueue in go: https://dev.to/frosnerd/writing-a-simple-tcp-server-using-kqueue-cah
func New(maxClients int) (IoMultiplexerInterface, error) {
	// creating kernel event queue
	kqueueFD, err := syscall.Kqueue()
	if err != nil {
		return nil, err
	}

	return &IoMultiplexerDarwin{
		Fd:         kqueueFD,
		MaxClients: maxClients,
	}, nil
}

func (i *IoMultiplexerDarwin) Subscribe(eventFd int) error {
	changeEvent := syscall.Kevent_t{
		Ident:  uint64(eventFd),     // fd we want to suscribe to
		Filter: syscall.EVFILT_READ, // Filter that processes the event. Set to EVFILT_READ,
		// which, when used in combination with a listening socket, indicates that we are interested in incoming connection events.
		Flags: syscall.EV_ADD | syscall.EV_ENABLE, // Flags that indicate what actions to perform with this event. In our case
		// we want to add the event to kqueue (EV_ADD), i.e. subscribing to it, and enable it (EV_ENABLE).
		Fflags: 0,
		Data:   0,
		Udata:  nil,
	}
	changeEventRegistered, err := syscall.Kevent(
		i.Fd,
		[]syscall.Kevent_t{changeEvent},
		nil,
		nil,
	)
	if err != nil || changeEventRegistered == -1 {
		return errors.New(fmt.Sprintf("failed registering change event to kqueue, err: %v", err))
	}

	return nil
}

func (i *IoMultiplexerDarwin) Poll() ([]*PolledEvent, error) {
	newEvents := make([]syscall.Kevent_t, i.MaxClients)
	numNewEvents, err := syscall.Kevent( // blocking call till any of the fds are available for IO
		i.Fd,
		nil,
		newEvents,
		nil,
	)
	if err != nil {
		return nil, err
	}

	polledEvents := make([]*PolledEvent, 0)
	for i := 0; i < numNewEvents; i++ {
		currentEvent := newEvents[i]
		polledEvents = append(polledEvents, &PolledEvent{
			Fd:              int(currentEvent.Ident),
			CloseConnection: currentEvent.Flags&syscall.EV_EOF != 0,
		})
	}
	return polledEvents, nil
}

func (i *IoMultiplexerDarwin) Close() {
	syscall.Close(i.Fd)
}
