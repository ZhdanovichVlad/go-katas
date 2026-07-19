//go:build !solution

package ratelimit

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

type Answer struct {
	isSuccessful bool
	err          error
}

type Task struct {
	chanAnswer chan struct{}
	ctx        context.Context
}

var ErrMaxLimit = fmt.Errorf("max_limit_per_timer")
var ErrLimiterClose = fmt.Errorf("err_limiter_close")

// Limiter is precise rate limiter with context support.
type Limiter struct {
	successfulTime     []time.Time
	limiterCloseSignal chan struct{}
	chTasks            chan Task
	wg                 sync.WaitGroup
	mu                 sync.RWMutex
	maxCount           int
	current            int
	windowTime         time.Duration
	isLimierMode       bool
	isLimiterClose     bool
}

var ErrStopped = errors.New("limiter stopped")

// NewLimiter returns limiter that throttles rate of successful Acquire() calls
// to maxSize events at any given interval.
func NewLimiter(maxCount int, interval time.Duration) *Limiter {

	limiter := Limiter{
		maxCount:       maxCount,
		mu:             sync.RWMutex{},
		successfulTime: make([]time.Time, 0),
	}

	if interval == 0 {
		limiter.isLimierMode = false
	} else {
		limiter.isLimierMode = true
		limiter.windowTime = interval
		chanTasks := make(chan Task, 1)
		limiter.chTasks = chanTasks

		limiter.limiterCloseSignal = make(chan struct{}, 1)

		limiter.current = 0

		limiter.wg.Add(1)
		go limiter.StartOrcestrator()
	}

	return &limiter
}

func (l *Limiter) Acquire(ctx context.Context) error {

	if !l.isLimierMode {
		l.mu.RLock()
		defer l.mu.RUnlock()
		if l.isLimiterClose {
			return ErrStopped
		}

		return nil
	}

	chanAnswer := make(chan struct{}, 1)

	task := Task{
		chanAnswer: chanAnswer,
		ctx:        ctx,
	}

	select {
	case <-ctx.Done():
		close(chanAnswer)
		return ctx.Err()

	case l.chTasks <- task:

	case <-l.limiterCloseSignal:
		close(chanAnswer)
		return ErrStopped
	}

	select {
	case <-ctx.Done():
		return ctx.Err()

	case <-chanAnswer:
		return nil

	case <-l.limiterCloseSignal:
		return ErrStopped
	}
}

func (l *Limiter) Stop() {
	l.mu.Lock()
	if l.isLimierMode {
		close(l.limiterCloseSignal)
		close(l.chTasks)
	}
	l.isLimiterClose = true
	l.mu.Unlock()
	l.wg.Wait()
}

func (l *Limiter) StartOrcestrator() {

	for {

		if l.isLimierMode {
			l.ClearTime()
		}

		if l.current == l.maxCount {
			continue
		}

		select {
		case task, ok := <-l.chTasks:
			if !ok {
				l.wg.Done()
				return
			}

			select {
			case <-task.ctx.Done():
				continue

			default:
				task.chanAnswer <- struct{}{}
			}

			l.successfulTime = append(l.successfulTime, time.Now())
			l.current++
		}

	}
}

func (l *Limiter) ClearTime() {
	if len(l.successfulTime) == 0 {
		return
	}

	timeClose := time.Now().Add(-1 * l.windowTime)
	for len(l.successfulTime) > 0 && l.successfulTime[0].Before(timeClose) {
		l.successfulTime = l.successfulTime[1:]
		l.current--
	}

}

// Управляющая горутина здесь должна быть “единственным владельцем состояния лимитера”.

// То есть все Acquire() не должны сами решать, можно проходить или ждать. Они отправляют запрос управляющей горутине, а она решает:

// можно ли сейчас разрешить вызов;
// надо ли поставить запрос в очередь;
// отменился ли ctx;
// вызвали ли Stop();
// когда по таймеру можно выпустить следующих ожидающих.
// По сути она хранит историю успешных Acquire за последний interval. Правило такое: в любом окне длиной interval успешных завершений должно быть не больше maxCount.

// Примерный смысл вопросов для себя:

// Когда приходит новый Acquire, сколько успешных вызовов было за последние interval?
// Если меньше maxCount, можно ли сразу ответить “успех”?
// Если уже maxCount, до какого времени нужно ждать, чтобы самый старый успешный вызов “выпал” из окна?
// Если есть очередь ожидающих, кого выпускать первым, когда появляется место?
// Если у ожидающего запроса отменился ctx, как убрать его из очереди и вернуть ctx.Err()?
// Если вызвали Stop(), как всем текущим и будущим Acquire вернуть ErrStopped, а управляющую горутину завершить?
// Минимальная схема: Acquire() создаёт маленький запрос с ctx и reply chan error, отправляет его в канал управляющей горутине, потом ждёт либо reply, либо отмену контекста. Управляющая горутина держит очередь запросов и список времён успешных acquire.
