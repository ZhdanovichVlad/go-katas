package semaphore

import (
	"context"
	"log"
	"sync"

	"github.com/ZhdanovichVlad/go-katas/http_survey/internal/cache"
	httpclient "github.com/ZhdanovichVlad/go-katas/http_survey/internal/client"
	"github.com/ZhdanovichVlad/go-katas/http_survey/internal/retrybudget"
	"github.com/ZhdanovichVlad/go-katas/http_survey/internal/surveyerr"
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

	cache, err := cache.NewCache(3)
	if err != nil {
		log.Println("semaphore: failed to create cache", err)
		return nil
	}

	wg := sync.WaitGroup{}

	retryBudget := retrybudget.NewRetryBudget(MAX_BUDGET, TOKEN_RATE, USAGE_RATE)

	var client Getter
	if isMock {
		client = httpclient.NewHTTPClientMock()
	} else {
		client = httpclient.NewHTTPClient()
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
	switch {
	case statusCode == surveyerr.ErrCode:
		return true
	case statusCode == 429:
		return true
	case statusCode >= 500 && statusCode < 600:
		return true
	default:
		return false
	}
}
