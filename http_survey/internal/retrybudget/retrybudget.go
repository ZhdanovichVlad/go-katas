package retrybudget

import (
	"sync"
)

type RetryBudget struct {
	mu            sync.Mutex
	currentBudget int
	maxBudget     int
	tokenRate     int
	usageRate     int
}

/*
 * @param maxBudget - максимальное количество токенов в бюджете
 * @param tokenRate - количество токенов, которое добавляется в бюджет за успешное выполнение запроса
 * @param usageRate - количество токенов, которое используется за одну повторную попытку
 * @return *RetryBudget
 */
func NewRetryBudget(maxBudget, tokenRate, usageRate int) *RetryBudget {
	return &RetryBudget{
		mu:            sync.Mutex{},
		currentBudget: 0,
		maxBudget:     maxBudget,
		tokenRate:     tokenRate,
		usageRate:     usageRate,
	}
}

func (rb *RetryBudget) AddTokens() {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.currentBudget += rb.tokenRate
	if rb.currentBudget > rb.maxBudget {
		rb.currentBudget = rb.maxBudget
	}
}

func (rb *RetryBudget) IsRetryAllowed() bool {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if rb.currentBudget < rb.usageRate {
		return false
	}

	rb.currentBudget -= rb.usageRate

	return true
}
