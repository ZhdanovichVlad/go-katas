//go:build !solution

package keylock

import (
	"slices"
)

type data struct {
	mu chan struct{}
}

type KeyLock struct {
	data map[string]data
	mu   chan struct{}
}

func New() *KeyLock {
	return &KeyLock{data: make(map[string]data), mu: make(chan struct{}, 1)}
}

func (l *KeyLock) LockKeys(keys []string, cancel <-chan struct{}) (canceled bool, unlock func()) {
	if len(keys) == 0 {
		return false, nil
	}

	l.createLocks(keys)

	for {
		select {
		case <-cancel:
			return true, nil
		default:
			sucses := []string{}

			copyKeys := make([]string, len(keys))
			copy(copyKeys, keys)

			slices.Sort(copyKeys)

			for _, v := range copyKeys {

				select {
				case <-cancel:
					for _, v := range sucses {
						<-l.data[v].mu
					}

					return true, nil

				case l.data[v].mu <- struct{}{}:
					sucses = append(sucses, v)
				}
			}

			return false, func() {
				for _, key := range copyKeys {
					v, _ := l.data[key]
					<-v.mu
				}
			}
		}
	}

}

func (l *KeyLock) createLocks(keys []string) {
	l.mu <- struct{}{}
	for _, key := range keys {
		_, ok := l.data[key]
		if !ok {
			l.data[key] = data{make(chan struct{}, 1)}
		}
	}
	<-l.mu
}
