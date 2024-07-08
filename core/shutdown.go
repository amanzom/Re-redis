package core

import (
	"os"
	"sync"
	"sync/atomic"

	"github.com/amanzom/re-redis/pkg/logger"
)

// engine status tells server's current request serving state
const (
	engineStatusBusy         int32 = 1 << 1 // 2
	engineStatusWaiting      int32 = 1 << 2 // 4
	engineStatusShuttingDown int32 = 1 << 3 // 8
)

var engineStatus int32 = engineStatusWaiting

func HandleGracefulShutdown(wg *sync.WaitGroup, signalChan chan os.Signal) {
	defer wg.Done()

	// blocking call till shutdown signals is received
	<-signalChan

	// wait for existing requests getting served - enine state busy
	for atomic.LoadInt32(&engineStatus) == engineStatusBusy {
	}

	atomic.StoreInt32(&engineStatus, engineStatusShuttingDown)

	performShutdown()
	os.Exit(0)
}

// no need of any transition check as done in MarkEngineStatusBusy() func
func MarkEngineStatusWaiting() {
	atomic.StoreInt32(&engineStatus, engineStatusWaiting)
}

// engine status can be marked busy only from waiting to busy state
// this way transition from shutdown to busy gets ruled out which
// prevents serving requests in shutdown state, see caller function
// in async server for more context
func MarkEngineStatusBusy() bool {
	return atomic.CompareAndSwapInt32(&engineStatus, engineStatusWaiting, engineStatusBusy)
}

func MarEngineStatusShuttingDown() {
	atomic.StoreInt32(&engineStatus, engineStatusShuttingDown)
}

func IsEngineStatusShuttingDown() bool {
	return atomic.LoadInt32(&engineStatus) == engineStatusShuttingDown
}

func performShutdown() {
	logger.Info("***** Shut down started *****")
	evalBgWriteAof([]string{})
	logger.Info("Shut down completed, bye bye!")
}
