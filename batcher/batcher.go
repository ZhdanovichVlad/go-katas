//go:build !solution

package batcher

import (
	"gitlab.com/slon/shad-go/batcher/slow"
)

type Batcher struct {
	value       *slow.Value
	isOpen      bool
	isOpenMu    chan struct{}
	batchRiding chan struct{}
	waiterCount int
	data        *data
}

type data struct {
	res         interface{}
	liderToken  chan struct{}
	waitingChan chan struct{}
}

func NewBatcher(v *slow.Value) *Batcher {
	batcher := Batcher{
		value:       v,
		isOpenMu:    make(chan struct{}, 1),
		batchRiding: make(chan struct{}, 1),
	}
	return &batcher
}

func (b *Batcher) Load() interface{} {

	b.batchRiding <- struct{}{}
	<-b.batchRiding
	b.addWating()

	select {
	case b.data.liderToken <- struct{}{}:
		b.batchRiding <- struct{}{}
		res := b.value.Load()

		b.data.res = res
		close(b.data.waitingChan)

		close(b.data.liderToken)
		<-b.data.liderToken

		b.minusWating()
		return res

	case <-b.data.waitingChan:
		res := b.data.res
		b.minusWating()
		return res

	}

}

func (b *Batcher) minusWating() {
	b.isOpenMu <- struct{}{}

	b.waiterCount--

	if b.waiterCount == 0 {
		b.data = nil
		b.isOpen = false
		<-b.batchRiding
	}

	<-b.isOpenMu
}

func (b *Batcher) addWating() {
	b.isOpenMu <- struct{}{}
	if !b.isOpen {
		b.isOpen = true
		b.data = &data{liderToken: make(chan struct{}, 1),
			waitingChan: make(chan struct{}, 1)}
	}
	b.waiterCount++
	<-b.isOpenMu
}
