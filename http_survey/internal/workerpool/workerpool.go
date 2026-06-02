package workerpool

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
	shard      = 3
)

type Getter interface {
	Get(ctx context.Context, url string) int
}

// job — единица работы для воркера: индекс в исходном списке + URL.
// Индекс нужен, чтобы воркер мог записать результат на свою позицию
// в общем срезе и сохранить порядок входа.
type job struct {
	idx int
	url string
}

func Survey(ctx context.Context, urls []string, parallel int, isMock bool) []int {
	if parallel < 1 {
		log.Println("workerpool: parallel must be greater than 0")
		return nil
	}

	results := make([]int, len(urls))

	cache, err := cache.NewCache(3)
	if err != nil {
		log.Println("workerpool: failed to create cache", err)
		return nil
	}

	wg := sync.WaitGroup{}

	chanInput := make(chan job, parallel)

	retryBudget := retrybudget.NewRetryBudget(MAX_BUDGET, TOKEN_RATE, USAGE_RATE)

	var client Getter
	if isMock {
		client = httpclient.NewHTTPClientMock()
	} else {
		client = httpclient.NewHTTPClient()
	}

	for i := 0; i < parallel; i++ {
		wg.Go(func() { worker(ctx, chanInput, results, cache, client, retryBudget) })
	}

	for i, url := range urls {
		chanInput <- job{idx: i, url: url}
	}
	close(chanInput)

	wg.Wait()

	return results
}

func worker(ctx context.Context,
	in <-chan job,
	results []int,
	cache *cache.Cache,
	client Getter,
	retryBudget *retrybudget.RetryBudget,
) {
	for j := range in {
		statusCode, ok := cache.Get(j.url)
		if !ok {
			statusCode = client.Get(ctx, j.url)
			if IsRetryableError(statusCode) && retryBudget.IsRetryAllowed() {
				statusCode = client.Get(ctx, j.url)
			}

			if statusCode >= 200 && statusCode < 400 {
				retryBudget.AddTokens()
			}

			cache.Set(j.url, statusCode)
		}
		results[j.idx] = statusCode
	}
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
