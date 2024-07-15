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
	info       = "info"
	client     = "client"
	latency    = "latency"
	multi      = "multi"
	exec       = "exec"
	discard    = "discard"

	// args
	ex = "ex"

	// resp
	resp_nil    = "$-1\r\n"
	resp_ok     = "+OK\r\n"
	resp_queued = "+QUEUED\r\n"

	// eviction strategies
	evictionStrategySimpleFirst  = "simple-first"
	evictionStrategAllKeysRandom = "allkeys-random"
	evictionStrategAllKeysLru    = "allkeys-lru"
)
