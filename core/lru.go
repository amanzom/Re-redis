package core

import (
	"time"

	"github.com/amanzom/re-redis/config"
)

const (
	lruSampleSize = 5
)

// LRU clock is the trimmed down version of actual clock where we take out the last 24 significant
// bits from the current epoch seconds, this can hold a total of 2^24-1 seconds or 194 days(3 months).
// It will be used to set the lastAccessedAt for approx LRU algorithm to evict keys.
// For keys which haven't been accessed in last 3 months this clock will hit its cycle and reset.
// In these cases approximated LRU will start evicting wrong keys but its highly unlikely for keys to
// remain unaccessed for 3 months.
func getLruClockCurrentTimestamp() uint32 {
	return uint32(time.Now().Unix() & 0x00FFFFFF)
}

// returns time for which the key is sitting idle
func getIdleTime(lastAccessedAt uint32) uint32 {
	c := getLruClockCurrentTimestamp()
	if lastAccessedAt > c {
		return lastAccessedAt - c
	}
	return (0x00FFFFFF - lastAccessedAt) + c
}

func populateEvictionPool() {
	samplesInserted := 0
	for key, obj := range store {
		evicitionPool.Push(key, obj.LastAccessedAt)
		samplesInserted++

		if samplesInserted == lruSampleSize {
			break
		}
	}
}

// Redis does not supports real LRU due to memory overhead of maintaining linked lists and shuffles.
// It does approximated LRU based eviction:
//  1. Sample 5 keys
//  2. Push these keys inside an eviction pool of max size 16, sorted by idleTime in decreasing order
//  3. If eviction pool size full but current elements's idleTime is greater than 0th elements
//     idleTime(current key worse than the worst item) - push current element inside removing
//     the last(best element) from pool
//
// 4. Evict the req no of elements from the pool

// Edge case with LRU:
//   - whenever store gets reconstructed from aof the lastAccessedAt get reset to the current time for all
//     keys in store. In this case we may start evicting wrong keys since key's previous state's info is lost.
func triggerApproximatedLruEviction() {
	populateEvictionPool()

	numOfKeysToEvict := int(config.EvictionRatio * float64(config.NumKeysThresholdForEviction))
	for i := 0; i < numOfKeysToEvict; i++ {
		popedItem := evicitionPool.Pop()
		if popedItem == nil {
			// eviction pool empty
			break
		}
		// evicting from store
		DelFromStore(popedItem.Key)
	}
}
