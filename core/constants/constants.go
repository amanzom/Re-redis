package constants

const (
	// commands
	Ping   = "ping"
	Pong   = "pong"
	Set    = "set"
	Get    = "get"
	Ttl    = "ttl"
	Expire = "expire"
	Del    = "del"

	// args
	EX = "ex"

	// resp
	RESP_NIL = "$-1\r\n"
	RESP_OK  = "+OK\r\n"

	// eviction strategies
	EvictionStrategySimpleFirst = "simple-first"
)
