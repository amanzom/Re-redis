package core

import "sort"

const (
	evictionPoolMaxSize = 16
)

var evicitionPool *EvictionPool

type PoolItem struct {
	Key            string
	LastAccessedAt uint32
}

func newPoolItem(key string, lastAccessedAt uint32) *PoolItem {
	return &PoolItem{
		Key:            key,
		LastAccessedAt: lastAccessedAt,
	}
}

// **********  req for sorting
type PoolItems []*PoolItem

func (pi PoolItems) Len() int {
	return len(pi)
}

func (pi PoolItems) Less(i int, j int) bool {
	return getIdleTime(pi[i].LastAccessedAt) > getIdleTime(pi[j].LastAccessedAt)
}

func (pi PoolItems) Swap(i int, j int) {
	pi[i], pi[j] = pi[j], pi[i]
}

// *************************

type EvictionPool struct {
	PoolItems    PoolItems
	PoolItemsMap map[string]*PoolItem // maintaining just for easy lookups
}

func newEvictionPool() *EvictionPool {
	return &EvictionPool{
		PoolItems:    make([]*PoolItem, 0),
		PoolItemsMap: make(map[string]*PoolItem),
	}
}

func (ep *EvictionPool) Push(key string, lastAccessedAt uint32) {
	// don't insert if key already present
	if _, ok := ep.PoolItemsMap[key]; ok {
		return
	}

	if len(ep.PoolItems) < evictionPoolMaxSize {
		// insert item and sort the pool by idle time in decreasing order
		poolItem := newPoolItem(key, lastAccessedAt)
		ep.PoolItems = append(ep.PoolItems, poolItem)
		ep.PoolItemsMap[key] = poolItem
		sort.Sort(ep.PoolItems)
	} else if getIdleTime(lastAccessedAt) > getIdleTime(ep.PoolItems[0].LastAccessedAt) {
		// removing the last item(with least idleTime), add new item at 0th index(no need to sort)
		delete(ep.PoolItemsMap, ep.PoolItems[evictionPoolMaxSize-1].Key)
		remainingPoolItems := ep.PoolItems[:evictionPoolMaxSize-1]

		poolItem := newPoolItem(key, lastAccessedAt)
		ep.PoolItems = []*PoolItem{poolItem}
		ep.PoolItems = append(ep.PoolItems, remainingPoolItems...)
		ep.PoolItemsMap[key] = poolItem
	}
}

// pop will evict the first item from pool since its sorted by idleTime in decreasing order
func (ep *EvictionPool) Pop() *PoolItem {
	if len(ep.PoolItems) == 0 {
		return nil
	}

	popedItem := ep.PoolItems[0]
	ep.PoolItems = ep.PoolItems[1:]
	delete(ep.PoolItemsMap, popedItem.Key)
	return popedItem
}

func (ep *EvictionPool) UpdateLastAccessedTimeForItem(key string) {
	item, ok := ep.PoolItemsMap[key]
	if ok {
		item.LastAccessedAt = getLruClockCurrentTimestamp()
	}
}
