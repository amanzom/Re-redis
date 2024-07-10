package core

import (
	"bytes"
	"fmt"
	"syscall"
)

var connectedClients = make(map[int]*Client, 0)

type Client struct {
	ConectionFD  int  // client's connection fd
	IsInTxnMode  bool // client in transaction mode
	TxnCmdsQueue []*RedisCmd
}

func NewClient(conectionFD int) *Client {
	return &Client{
		ConectionFD:  conectionFD,
		TxnCmdsQueue: make([]*RedisCmd, 0),
	}
}

func (c *Client) Read(b []byte) (int, error) {
	return syscall.Read(c.ConectionFD, b)
}

func (c *Client) Write(b []byte) (int, error) {
	return syscall.Write(c.ConectionFD, b)
}

func (c *Client) IsTxnMode() bool {
	return c.IsInTxnMode
}

func (c *Client) MarkClientInTxnMode() {
	c.IsInTxnMode = true
}

func (c *Client) EnqueueTxnCommand(cmd *RedisCmd) {
	c.TxnCmdsQueue = append(c.TxnCmdsQueue, cmd)
}

func (c *Client) ExecuteTxnQueuedCommands() []byte {
	var out []byte
	buf := bytes.NewBuffer(out)
	for _, cmd := range c.TxnCmdsQueue {
		buf.Write(executeCommand(cmd, c))
	}
	queueLength := len(c.TxnCmdsQueue)
	c.DiscardTxn()
	return []byte(fmt.Sprintf("*%v\r\n%v", queueLength, string(buf.Bytes())))
}

func (c *Client) DiscardTxn() {
	c.TxnCmdsQueue = []*RedisCmd{}
	c.IsInTxnMode = false
}

func GetConnectedClientFromFD(fd int) *Client {
	return connectedClients[fd]
}

func MarkClientConnected(fd int, client *Client) {
	connectedClients[fd] = client
}

func MarkClientDisconnected(fd int) {
	delete(connectedClients, fd)
}
