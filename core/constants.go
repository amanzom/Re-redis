package core

const (
	// commands
	ping       = "ping"
	pong       = "pong"
	set        = "set"
	get        = "get"
	ttl        = "ttl"
	expire     = "expire"
	del        = "del"
	bgWriteAof = "bgwriteaof"
	incr       = "incr"

	// args
	ex = "ex"

	// resp
	resp_nil = "$-1\r\n"
	resp_ok  = "+OK\r\n"

	// eviction strategies
	evictionStrategySimpleFirst = "simple-first"
)
