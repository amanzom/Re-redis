package core

import (
	"github.com/amanzom/re-redis/config"
)

// iterates on a randomized sample of keys evicts the first key
// TODO - sampling of keys can be improved - currently relying on go since ranging
// over maps keys is quite randomized.
func evictSimpleFirst() {
	for key := range store {
		DelFromStore(key)
		break
	}
}

func evict() {
	switch config.EvictionStrategy {
	case evictionStrategySimpleFirst:
		evictSimpleFirst()
		break
	}
}
