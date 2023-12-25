package store

import (
	"time"

	"github.com/amanzom/re-redis/core/logger"
)

const (
	sampleAutoDeleteRetriggerThreshold = 0.25
	sampleSize                         = 20
)

// Takes a random sample of 20 keys whose expiration is set and not yet deleted from `store` hashmap,
// deletes the expired keys and returns the fraction of keys deleted to the sample size selected(20)
func expireSample() float64 {
	leftNonExpiredKeysToIterate := sampleSize // will maintain count of keys which have expiration set and not deleted
	noOfDeletedKeys := 0
	for key, val := range store {
		if val != nil && val.ExpiresAt != -1 {
			leftNonExpiredKeysToIterate--
			if val.ExpiresAt <= time.Now().UnixMilli() {
				delete(store, key)
				noOfDeletedKeys++
			}
		}

		if leftNonExpiredKeysToIterate == 0 {
			break
		}
	}

	logger.Info("keys deleted: %v", noOfDeletedKeys)
	return float64(noOfDeletedKeys) / float64(sampleSize)
}

// deletes the expired keys - active way
// ref: https://redis.io/commands/expire/
func DeleteExpiredKeys() {
	logger.Info("\r\n")
	logger.Info("Running Auto Delete Cron")
	for {
		fraction := expireSample()
		// if the sample of keys having expiration set had >= 25% keys expired(which got deleted),
		// repeat this - since high probability to find more such expired keys, else break the loop.
		if fraction < sampleAutoDeleteRetriggerThreshold {
			break
		}
	}
	logger.Info("Num of keys left: %v", len(store))
	logger.Info("\r\n")
}
