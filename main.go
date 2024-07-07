package main

import (
	"flag"
	"os"
	"strconv"

	"github.com/amanzom/re-redis/config"
	"github.com/amanzom/re-redis/pkg/logger"
	"github.com/amanzom/re-redis/server"
)

func setupFlags() {
	port, err := strconv.Atoi(os.Getenv("PORT_RE_REDIS"))
	if err != nil {
		logger.Error("error getting port: %v", err)
		return
	}

	flag.StringVar(&config.Host, "Host", "0.0.0.0", "Host for re-redis server")
	flag.IntVar(&config.Port, "Port", port, "Port for re-redis server")
	flag.Parse()
}

func main() {
	setupFlags()
	server.NewAsyncTcpServer(config.Host, config.Port).StartServer()
}
