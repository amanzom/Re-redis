package core

import (
	"bytes"

	"github.com/amanzom/re-redis/config"
	"github.com/amanzom/re-redis/pkg/logger"
)

func init() {
	store = make(map[string]*Obj, 0)

	// initialising buffer for storing commands for aof writes periodically
	var b []byte
	commandsBuffer = bytes.NewBuffer(b)

	// reconstructing the store on boot up from aof
	if config.StoreReconstructEnabledOnBootUp {
		if err := reconstructStoreFromAof(); err != nil {
			logger.Error("error reconstructing store from aof, err: %v", err)
		}
	}
}
