package main

import (
	"flag"

	"github.com/amanzom/re-redis/config"
	"github.com/amanzom/re-redis/server"
)

func setupFlags() {
	flag.StringVar(&config.Host, "Host", "0.0.0.0", "Host for re-redis server")
	flag.IntVar(&config.Port, "Port", 7369, "Port for re-redis server")
	flag.Parse()
}

func main() {
	setupFlags()
	server.NewAsyncTcpServer(config.Host, config.Port).StartServer()
}
