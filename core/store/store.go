package store

import "time"

type Obj struct {
	Value     interface{}
	ExpiresAt int64
}

var store map[string]*Obj

func init() {
	store = make(map[string]*Obj, 0)
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

func Put(k string, obj *Obj) {
	store[k] = obj
}

func Get(k string) *Obj {
	// passively delete before get
	val, ok := store[k]
	if ok {
		if val.ExpiresAt != -1 && val.ExpiresAt <= time.Now().UnixMilli() {
			delete(store, k)
			return nil
		}
	}
	return val
}

func Del(k string) bool {
	if _, ok := store[k]; ok {
		delete(store, k)
		return true
	}
	return false
}
