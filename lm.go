// Package lm implements a keyed RWMutex
package lm

import (
	"sync"
)

type resource struct {
	readers    uint64
	writers    uint64
	readQueue  []*chan bool
	writeQueue []*chan bool
}

// Lock is a keyed RWMutex
type Lock struct {
	mutex *sync.Mutex
	locks map[string]*resource
}

// NewLock returns a new Lock
func NewLock() *Lock {
	return &Lock{
		mutex: &sync.Mutex{},
		locks: make(map[string]*resource),
	}
}

// Lock acquires the write lock on the given key
func (lm *Lock) Lock(key string) {

	lm.mutex.Lock()
	r := lm.resource(key, true)

	if r.writers == 0 && r.readers == 0 {
		r.writers++
		lm.mutex.Unlock()
		return
	}

	c := make(chan bool, 1)
	r.writeQueue = append(r.writeQueue, &c)

	lm.mutex.Unlock()
	<-c

}

// RLock acquires a read lock on the given key
func (lm *Lock) RLock(key string) {

	lm.mutex.Lock()
	r := lm.resource(key, true)

	if r.writers == 0 && len(r.writeQueue) == 0 {
		r.readers++
		lm.mutex.Unlock()
		return
	}

	c := make(chan bool, 1)
	r.readQueue = append(r.readQueue, &c)

	lm.mutex.Unlock()
	<-c

}

// RUnlock releases a read lock on the given key
func (lm *Lock) RUnlock(key string) {
	lm.mutex.Lock()
	defer lm.mutex.Unlock()
	r := lm.resource(key, false)

	if r != nil && r.readers > 0 {

		r.readers--

		if r.readers == 0 {

			if len(r.writeQueue) > 0 {
				c := r.writeQueue[0]
				r.writeQueue = r.writeQueue[1:]
				r.writers++
				*c <- true
			} else {
				delete(lm.locks, key)
			}

		}

	}

}

// Unlock releases the write lock on the given key
func (lm *Lock) Unlock(key string) {
	lm.mutex.Lock()
	defer lm.mutex.Unlock()
	r := lm.resource(key, false)

	if r != nil && r.writers > 0 {

		r.writers--

		if len(r.writeQueue) > 0 {
			c := r.writeQueue[0]
			r.writeQueue = r.writeQueue[1:]
			r.writers++
			*c <- true
		} else if len(r.readQueue) > 0 {
			r.readers += uint64(len(r.readQueue))

			for _, c := range r.readQueue {
				*c <- true
			}

			r.readQueue = make([]*chan bool, 0, 1)
		} else {
			delete(lm.locks, key)
		}

	}

}

func (lm *Lock) resource(key string, create bool) *resource {

	r, ok := lm.locks[key]

	if ok || !create {
		return r
	}

	r = &resource{
		readers:    0,
		readQueue:  make([]*chan bool, 0, 1),
		writers:    0,
		writeQueue: make([]*chan bool, 0, 1),
	}

	lm.locks[key] = r
	return r
}
