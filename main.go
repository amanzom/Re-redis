package main

import (
	"flag"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/amanzom/re-redis/config"
	"github.com/amanzom/re-redis/core"
	"github.com/amanzom/re-redis/server"
)

func setupFlags() {
	flag.StringVar(&config.Host, "Host", "0.0.0.0", "Host for re-redis server")
	flag.IntVar(&config.Port, "Port", 7369, "Port for re-redis server")
	flag.Parse()
}

func main() {
	setupFlags()

	// chan to listen shutdown signals
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGTERM, syscall.SIGINT)

	var wg sync.WaitGroup
	wg.Add(2)

	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		server.NewAsyncTcpServer(config.Host, config.Port).StartServer()
	}(&wg)
	go core.HandleGracefulShutdown(&wg, signalChan)

	wg.Wait()
}
