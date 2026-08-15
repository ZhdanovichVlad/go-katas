//go:build !solution

package batcher

import (
	"sync"

	"gitlab.com/slon/shad-go/batcher/slow"
)

type Batcher struct {
	storage     *slow.Value
	data        *data
	mu          sync.Mutex
	batcherLock chan struct{}
	waiterCount int
}

type data struct {
	res          interface{}
	notification chan struct{}
}

func NewBatcher(v *slow.Value) *Batcher {
	batcher := Batcher{
		storage:     v,
		batcherLock: make(chan struct{}, 1),
	}
	return &batcher
}

func (b *Batcher) Load() interface{} {

	b.batcherLock <- struct{}{}
	<-b.batcherLock

	isLider := false

	b.mu.Lock()
	if b.data == nil {
		b.batcherLock <- struct{}{}
		isLider = true
		b.data = &data{notification: make(chan struct{})}
	}
	b.waiterCount++
	b.mu.Unlock()

	var res interface{}
	if isLider {
		res = b.storage.Load()
		b.data.res = res
		close(b.data.notification)
	} else {
		<-b.data.notification
		res = b.data.res
	}

	b.mu.Lock()
	b.waiterCount--

	if b.waiterCount == 0 {
		<-b.batcherLock
		b.data = nil
	}
	b.mu.Unlock()

	return res
}
