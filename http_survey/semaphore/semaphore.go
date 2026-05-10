package semaphore

import (
	"context"
	"log"
	_ "net/http/pprof"
	"sync"

	"github.com/ZhdanovichVlad/go-katas/http_survey/cache"
	"github.com/ZhdanovichVlad/go-katas/http_survey/errors"
	"github.com/ZhdanovichVlad/go-katas/http_survey/http_client"
	"github.com/ZhdanovichVlad/go-katas/http_survey/retry_budget"
)

const (
	MAX_BUDGET = 100
	TOKEN_RATE = 10
	USAGE_RATE = 1
)

type Getter interface {
	Get(ctx context.Context, url string) int
}

// Есть список ссылок
// Необходимо написать функцию которая опросит каждый из url-ов в этом списке
// и вернет статусы в том же порядке, в котором они были переданы.
// Кол-во параллельных запросов регулируется параметров parallel
func Survey(ctx context.Context, urls []string, parallel int, isMock bool) []int {
	if parallel < 1 {
		log.Println("semaphore: parallel must be greater than 0")
		return nil
	}
	
	results := make([]int, len(urls))

	cache := cache.NewCache()

	wg := sync.WaitGroup{}

	retryBudget := retry_budget.NewRetryBudget(MAX_BUDGET, TOKEN_RATE, USAGE_RATE)

	var client Getter
	if isMock {
		client = http_client.NewHTTPClientMock()
	} else {
		client = http_client.NewHTTPClient()
	}

	semaphore := make(chan struct{}, parallel)

	for i, url := range urls {
		semaphore <- struct{}{}

		wg.Go(func() {
			defer func() { <-semaphore }()

			statusCode, ok := cache.Get(url)
			if !ok {
				statusCode = client.Get(ctx, url)
				if IsRetryableError(statusCode) && retryBudget.IsRetryAllowed() {
					statusCode = client.Get(ctx, url)
				}

				if statusCode >= 200 && statusCode < 400 {
					retryBudget.AddTokens()
				}

				cache.Set(url, statusCode)
			}

			results[i] = statusCode
		})
	}

	wg.Wait()
	close(semaphore)

	return results
}


func IsRetryableError(statusCode int) bool {
	switch  {
	case statusCode == errors.ErrCode:
		return true
	case statusCode == 429:
		return true
	case statusCode >= 500 && statusCode < 600:
		return true
	default:
		return false
	}
}