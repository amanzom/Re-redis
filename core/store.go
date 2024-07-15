package core

import (
	"time"

	"github.com/amanzom/re-redis/config"
)

type Obj struct {
	Value     interface{}
	ExpiresAt int64 // redis keeps a different map for keys which have expiry set, we for simplicity
	// have kept ExpiresAt inside main store object
	TypeEncoding   typeEncoding
	LastAccessedAt uint32 // we will be maintaining last accessed epoch's 24 bits in LastAccessedAt - the way handled in redis,
	// to save extra 8 bits per object, though we are using uint32 since go does not supports bitfields.
}

var store map[string]*Obj

func NewObj(value interface{}, expiryInMs int64, oType uint8, oEnc uint8) *Obj {
	var expiresAt int64 = -1
	if expiryInMs > 0 {
		expiresAt = time.Now().UnixMilli() + expiryInMs
	}
	return &Obj{
		Value:          value,
		ExpiresAt:      expiresAt,
		TypeEncoding:   typeEncoding(oType | oEnc),
		LastAccessedAt: getLruClockCurrentTimestamp(),
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
		// updating last accessed at for lru eviction in store and eviction pool
		val.LastAccessedAt = getLruClockCurrentTimestamp()
		evicitionPool.UpdateLastAccessedTimeForItem(k)
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
