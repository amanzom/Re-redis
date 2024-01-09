package core

import (
	"bytes"
	"time"

	"github.com/amanzom/re-redis/config"
	"github.com/amanzom/re-redis/pkg/logger"
)

type Obj struct {
	Value     interface{}
	ExpiresAt int64
}

var store map[string]*Obj

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

func NewObj(value interface{}, expiryInMs int64) *Obj {
	var expiresAt int64 = -1
	if expiryInMs > 0 {
		expiresAt = time.Now().UnixMilli() + expiryInMs
	}
	return &Obj{
		Value:     value,
		ExpiresAt: expiresAt,
	}
}

func PutInStore(k string, obj *Obj) {
	// evict keys if total nums of keys goes beyond a threshold
	// TODO - think of linking eviction to memory and not num keys
	if len(store) >= config.NumKeysThresholdForEviction {
		evict()
	}

	// storing in commands buffer for aof writes periodically
	commandsBuffer.Write(getKeyValueSetCommandRespEncodedBytes(k, obj.Value))
	store[k] = obj
}

func GetFromStore(k string) *Obj {
	// passively delete before get
	val, ok := store[k]
	if ok {
		if val.ExpiresAt != -1 && val.ExpiresAt <= time.Now().UnixMilli() {
			DelFromStore(k)
			return nil
		}
	}
	return val
}

func DelFromStore(k string) bool {
	if _, ok := store[k]; ok {
		delete(store, k)

		// storing in commands buffer for aof writes periodically
		commandsBuffer.Write(getKeyValueDeleteCommandRespEncodedBytes(k))
		return true
	}
	return false
}
