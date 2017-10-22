package lm

import (
	"testing"
	"time"
)

const (
	waitTime = 1 * time.Millisecond
)

func TestLM(t *testing.T) {
	lock := NewLock()

	runFullTest := func(key string) {
		readers := 5
		writing := 0
		reading := 0

		readerLock := make(chan bool)
		readerUnlock := make(chan bool)
		writerLock := make(chan bool)
		writerUnlock := make(chan bool)
		secondWriterLock := make(chan bool)
		secondWriterUnlock := make(chan bool)

		go func() {
			<-writerLock
			lock.Lock(key)
			writing++
			<-writerUnlock
			writing--
			lock.Unlock(key)
		}()

		for i := 0; i < readers; i++ {
			go func() {
				<-readerLock
				lock.RLock(key)
				reading++
				<-readerUnlock
				reading--
				lock.RUnlock(key)
			}()
		}

		go func() {
			<-secondWriterLock
			lock.Lock(key)
			writing++
			<-secondWriterUnlock
			writing--
			lock.Unlock(key)
		}()

		if writing != 0 || reading != 0 {
			t.Fail()
		}

		for i := 0; i < readers-2; i++ {
			readerLock <- true
		}

		time.Sleep(waitTime)

		if reading != readers-2 || writing != 0 {
			t.Fail()
		}

		writerLock <- true
		time.Sleep(waitTime)

		if reading != readers-2 || writing != 0 {
			t.Fail()
		}

		readerUnlock <- true
		time.Sleep(waitTime)

		if reading != readers-3 || writing != 0 {
			t.Fail()
		}

		for i := 0; i < readers-3; i++ {
			readerUnlock <- true
		}

		time.Sleep(waitTime)

		if writing != 1 || reading != 0 {
			t.Fail()
		}

		readerLock <- true
		time.Sleep(waitTime)
		secondWriterLock <- true
		time.Sleep(waitTime)

		if writing != 1 || reading != 0 {
			t.Fail()
		}

		writerUnlock <- true
		time.Sleep(waitTime)

		if writing != 1 || reading != 0 {
			t.Fail()
		}

		secondWriterUnlock <- true
		time.Sleep(waitTime)

		if writing != 0 || reading != 1 {
			t.Fail()
		}

		readerUnlock <- true
		time.Sleep(waitTime)

		if writing != 0 || reading != 0 {
			t.Fail()
		}

	}

	runShortTest := func(key string) {
		c1 := make(chan bool)
		c2 := make(chan bool)

		go func() {
			lock.Lock(key)
			lock.Unlock(key)
			c1 <- true
		}()

		go func() {
			lock.RLock(key)
			lock.RUnlock(key)
			c2 <- true
		}()

		<-c1
		<-c2
	}

	runFullTest("foo")
	runFullTest("bar")
	runShortTest("foo")
	runShortTest("bar")
	runShortTest("baz")
	runShortTest("foobaz")
}
