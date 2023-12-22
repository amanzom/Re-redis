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
	return store[k]
}
