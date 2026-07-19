//go:build !solution

package dupcall

import (
	"context"
	"sync"
)

type callState struct {
	done chan struct{}
	ctx context.Context
	cancel context.CancelFunc
	waiterCount int

	result interface{}
	err    error
}

type Call struct {
	mu        sync.Mutex
	callState *callState
}

func (o *Call) Do(
	ctx context.Context,
	cb func(context.Context) (interface{}, error),
) (result interface{}, err error) {

	o.mu.Lock()
	if o.callState == nil {

		callState := callState{
			done:        make(chan struct{}),
			waiterCount: 0,
		}

		o.callState = &callState
		cbCtx, cancel := context.WithCancel(context.Background()) 
		o.callState.ctx = cbCtx
		o.callState.cancel = cancel

		go func() {
			answer, err := cb(cbCtx)

			if ctx.Err() != nil {
				return
			}

			o.callState.result = answer
			o.callState.err = err
			close(o.callState.done)
		}()
	}
	o.callState.waiterCount++
	o.mu.Unlock()

	select {
	case <-o.callState.done:
		o.mu.Lock()
		answer := o.callState.result
		err := o.callState.err

		o.callState.waiterCount -= 1

		if o.callState.waiterCount == 0 {
			o.callState = nil
		}
		o.mu.Unlock()

		return answer, err
	case <-ctx.Done():
		o.mu.Lock()
		o.callState.waiterCount -= 1

		if o.callState.waiterCount == 0 {
			o.callState.cancel()
			o.callState = nil
		}
		o.mu.Unlock()

		return nil, ctx.Err()
	}

}
