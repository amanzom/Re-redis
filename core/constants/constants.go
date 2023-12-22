package constants

const (
	// commands
	Ping = "ping"
	Pong = "pong"
	Set  = "set"
	Get  = "get"
	Ttl  = "ttl"

	// args
	EX = "ex"

	// resp
	RESP_NIL = "$-1\r\n"
	RESP_OK  = "+OK\r\n"
)
