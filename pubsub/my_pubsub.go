//go:build !solution

package pubsub

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
)

var (
	ErrTopickNotFound = fmt.Errorf("err topick not found")
	ErrPubSubIsClosed = fmt.Errorf("err pub/sub is closed")
)

var _ Subscription = (*MySubscription)(nil)

type MySubscription struct {
	id          int
	topick      string
	sendMessage func(msg interface{})
	once        sync.Once
	pubSub      *MyPubSub

	msQueue           []interface{}
	muMsQueue         sync.Mutex
	countSent         int
	closeNotification chan struct{}
}

func NewMySubscription(
	id int,
	topick string,
	sendMessage func(msg interface{}),
	pubsub *MyPubSub,
) *MySubscription {

	sub := MySubscription{
		id:                id,
		topick:            topick,
		msQueue:           make([]interface{}, 0),
		closeNotification: make(chan struct{}, 1),
		sendMessage:       sendMessage,
		pubSub:            pubsub}

	return &sub
}

func (s *MySubscription) Start() {
	for {
		select {
		case <-s.closeNotification:
			return
		default:
			s.muMsQueue.Lock()
			if len(s.msQueue) <= s.countSent {
				s.muMsQueue.Unlock()
				continue
			}
			msg := s.msQueue[s.countSent]
			s.countSent++
			s.muMsQueue.Unlock()

			s.sendMessage(msg)
			s.pubSub.pending.Add(-1)
		}
	}
}

func (s *MySubscription) stop() {
	s.once.Do(func() {
		close(s.closeNotification)

		s.muMsQueue.Lock()
		left := len(s.msQueue) - s.countSent
		s.countSent = len(s.msQueue)
		s.muMsQueue.Unlock()

		if left > 0 {
			s.pubSub.pending.Add(-int64(left))
		}
	})
}

func (s *MySubscription) Unsubscribe() {
	s.stop()

	s.pubSub.muStorage.Lock()
	defer s.pubSub.muStorage.Unlock()

	subs, ok := s.pubSub.storage[s.topick]
	if !ok {
		return
	}

	delete(subs.subscribers, s.id)
	if len(subs.subscribers) == 0 {
		delete(s.pubSub.storage, s.topick)
	}
}

var _ PubSub = (*MyPubSub)(nil)

type MyPubSub struct {
	muStorage  sync.RWMutex
	storage    map[string]*Subscribers
	ids        int
	muIsClosed sync.RWMutex
	isClosed   bool
	pending    atomic.Int64
}

func NewPubSub() PubSub {
	return &MyPubSub{
		storage: make(map[string]*Subscribers),
	}
}

type Subscribers struct {
	subscribers map[int]*MySubscription
}

func (p *MyPubSub) Subscribe(subj string, cb MsgHandler) (Subscription, error) {
	p.muStorage.Lock()
	defer p.muStorage.Unlock()

	p.muIsClosed.Lock()
	if p.isClosed {
		p.muIsClosed.Unlock()
		return nil, ErrPubSubIsClosed
	}
	p.muIsClosed.Unlock()

	subscriberId := p.ids + 1
	p.ids++

	mySb := NewMySubscription(subscriberId, subj, cb, p)
	go mySb.Start()

	v, ok := p.storage[subj]
	if !ok {
		sub := Subscribers{subscribers: make(map[int]*MySubscription)}
		sub.subscribers[subscriberId] = mySb
		p.storage[subj] = &sub
	} else {
		v.subscribers[subscriberId] = mySb
	}

	return mySb, nil
}

func (p *MyPubSub) Publish(subj string, msg interface{}) error {
	p.muStorage.RLock()
	defer p.muStorage.RUnlock()

	p.muIsClosed.RLock()
	closed := p.isClosed
	p.muIsClosed.RUnlock()
	if closed {
		return ErrPubSubIsClosed
	}

	v, ok := p.storage[subj]
	if !ok {
		return ErrTopickNotFound
	}

	for _, sub := range v.subscribers {
		sub.muMsQueue.Lock()
		sub.msQueue = append(sub.msQueue, msg)
		p.pending.Add(1)
		sub.muMsQueue.Unlock()
	}

	return nil
}

func (p *MyPubSub) stopAll() {
	p.muStorage.Lock()
	defer p.muStorage.Unlock()

	for _, v := range p.storage {
		for _, sub := range v.subscribers {
			sub.stop()
		}
	}
	p.storage = make(map[string]*Subscribers)
}

func (p *MyPubSub) Close(ctx context.Context) error {
	p.muIsClosed.Lock()
	if p.isClosed {
		p.muIsClosed.Unlock()
		return nil
	}
	p.isClosed = true
	p.muIsClosed.Unlock()

	for p.pending.Load() > 0 {
		select {
		case <-ctx.Done():
			p.stopAll()
			return ctx.Err()
		default:
			runtime.Gosched()
		}
	}

	p.stopAll()
	return nil
}
