package core

import (
	"time"

	"github.com/amanzom/re-redis/config"
)

type Obj struct {
	Value        interface{}
	ExpiresAt    int64
	TypeEncoding typeEncoding
}

var store map[string]*Obj

func NewObj(value interface{}, expiryInMs int64, oType uint8, oEnc uint8) *Obj {
	var expiresAt int64 = -1
	if expiryInMs > 0 {
		expiresAt = time.Now().UnixMilli() + expiryInMs
	}
	return &Obj{
		Value:        value,
		ExpiresAt:    expiresAt,
		TypeEncoding: typeEncoding(oType | oEnc),
	}
}

func PutInStore(k string, obj *Obj) {
	// evict keys if total nums of keys goes beyond a threshold
	// TODO - think of linking eviction to memory and not num keys
	if len(store) >= config.NumKeysThresholdForEviction {
		evict()
	}

	if _, ok := store[k]; !ok {
		// updating stats
		incrementKeyspaceStats("keys")
	}

	store[k] = obj

	// storing in commands buffer for aof writes periodically
	commandsBuffer.Write(getKeyValueSetCommandRespEncodedBytes(k, obj.Value))
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

		// updating stats
		decrementKeyspaceStats("keys")

		// storing in commands buffer for aof writes periodically
		commandsBuffer.Write(getKeyValueDeleteCommandRespEncodedBytes(k))
		return true
	}
	return false
}
