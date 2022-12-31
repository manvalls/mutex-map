package mutexmap

import "sync"

type keyInfo struct {
	readers    uint64
	writting   bool
	readQueue  []chan bool
	writeQueue []chan bool
}

type MutexMap[K comparable] struct {
	mutex   sync.Mutex
	keyInfo map[K]*keyInfo
}

func NewMutexMap[K comparable]() *MutexMap[K] {
	return &MutexMap[K]{sync.Mutex{}, make(map[K]*keyInfo)}
}

func (m *MutexMap[K]) Lock(key K) {
	m.mutex.Lock()

	info, ok := m.keyInfo[key]
	if !ok {
		info = &keyInfo{}
		m.keyInfo[key] = info
	}

	if !info.writting && info.readers == 0 {
		info.writting = true
		m.mutex.Unlock()
		return
	}

	ch := make(chan bool)
	info.writeQueue = append(info.writeQueue, ch)
	m.mutex.Unlock()
	<-ch
}

func (m *MutexMap[K]) Unlock(key K) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	info, ok := m.keyInfo[key]
	if !ok || !info.writting {
		return
	}

	if len(info.writeQueue) > 0 {
		info.writeQueue[0] <- true
		info.writeQueue = info.writeQueue[1:]
		return
	}

	info.writting = false

	if len(info.readQueue) > 0 {
		info.readers += uint64(len(info.readQueue))
		for _, ch := range info.readQueue {
			ch <- true
		}

		info.readQueue = nil
		return
	}

	delete(m.keyInfo, key)
}

func (m *MutexMap[K]) RLock(key K) {
	m.mutex.Lock()

	info, ok := m.keyInfo[key]
	if !ok {
		info = &keyInfo{}
		m.keyInfo[key] = info
	}

	if !info.writting && len(info.writeQueue) == 0 {
		info.readers++
		m.mutex.Unlock()
		return
	}

	ch := make(chan bool)
	info.readQueue = append(info.readQueue, ch)
	m.mutex.Unlock()
	<-ch
}

func (m *MutexMap[K]) RUnlock(key K) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	info, ok := m.keyInfo[key]
	if !ok || info.readers == 0 {
		return
	}

	info.readers--
	if info.readers > 0 {
		return
	}

	if len(info.writeQueue) > 0 {
		info.writting = true
		info.writeQueue[0] <- true
		info.writeQueue = info.writeQueue[1:]
		return
	}

	delete(m.keyInfo, key)
}

type MutexMapKey[K comparable] struct {
	m *MutexMap[K]
	k K
}

func (m *MutexMap[K]) Key(key K) *MutexMapKey[K] {
	return &MutexMapKey[K]{m, key}
}

func (k *MutexMapKey[K]) Lock() {
	k.m.Lock(k.k)
}

func (k *MutexMapKey[K]) Unlock() {
	k.m.Unlock(k.k)
}

func (k *MutexMapKey[K]) RLock() {
	k.m.RLock(k.k)
}

func (k *MutexMapKey[K]) RUnlock() {
	k.m.RUnlock(k.k)
}
