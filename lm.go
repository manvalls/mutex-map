// Package lm implements a keyed RWMutex
package lm

import "sync"

type consumer struct {
	reader  bool
	channel *chan bool
}

type resource struct {
	readers uint
	writers uint
	queue   []consumer
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
	r := lm.resource(key)

	if r.writers == 0 && r.readers == 0 {
		r.writers++
		lm.mutex.Unlock()
		return
	}

	c := make(chan bool, 1)

	r.queue = append(r.queue, consumer{
		reader:  false,
		channel: &c,
	})

	lm.mutex.Unlock()
	<-c

}

// RLock acquires a read lock on the given key
func (lm *Lock) RLock(key string) {

	lm.mutex.Lock()
	r := lm.resource(key)

	if r.writers == 0 {
		r.readers++
		lm.mutex.Unlock()
		return
	}

	c := make(chan bool, 1)

	r.queue = append(r.queue, consumer{
		reader:  true,
		channel: &c,
	})

	lm.mutex.Unlock()
	<-c

}

// RUnlock releases a read lock on the given key
func (lm *Lock) RUnlock(key string) {
	lm.mutex.Lock()
	defer lm.mutex.Unlock()
	r := lm.resource(key)

	if r.readers > 0 {

		r.readers--

		if r.readers == 0 {

			if len(r.queue) > 0 {
				c := r.queue[0]
				r.queue = r.queue[1:]
				r.writers++
				*c.channel <- true
			}

		}

	}

	if r.writers == 0 && r.readers == 0 {
		delete(lm.locks, key)
	}
}

// Unlock releases the write lock on the given key
func (lm *Lock) Unlock(key string) {
	lm.mutex.Lock()
	defer lm.mutex.Unlock()
	r := lm.resource(key)

	if r.writers > 0 {

		r.writers--

		for len(r.queue) > 0 {
			c := r.queue[0]
			r.queue = r.queue[1:]

			if c.reader {
				r.readers++
				*c.channel <- true
			} else {
				r.writers++
				*c.channel <- true
				break
			}
		}

	}

	if r.writers == 0 && r.readers == 0 {
		delete(lm.locks, key)
	}
}

func (lm *Lock) resource(key string) *resource {

	r, ok := lm.locks[key]

	if ok {
		return r
	}

	r = &resource{
		readers: 0,
		writers: 0,
		queue:   make([]consumer, 1),
	}

	lm.locks[key] = r
	return r
}
