package store

import (
	"github.com/amanzom/re-redis/config"
	"github.com/amanzom/re-redis/core/constants"
)

// iterates on a randomized sample of keys evicts the first key
// TODO - sampling of keys can be improved - currently relying on go since ranging
// over maps keys is quite randomized.
func evictSimpleFirst() {
	for key := range store {
		delete(store, key)
		break
	}
}

func evict() {
	switch config.EvictionStrategy {
	case constants.EvictionStrategySimpleFirst:
		evictSimpleFirst()
		break
	}
}
