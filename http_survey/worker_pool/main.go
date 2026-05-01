package main

import (
	"context"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/ZhdanovichVlad/go-katas/http_survey/cache"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pprofServer := &http.Server{Addr: "localhost:6060"}
	go func() {
		log.Println("pprof listening on http://localhost:6060/debug/pprof/")
		if err := pprofServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("pprof server error: %v", err)
		}
	}()

	baseURLs := []string{
		"https://example.com",
		"https://google.com",
		"https://go.dev",
		"https://pkg.go.dev",
		"https://github.com",
		"https://httpbin.org/status/200",
		"https://httpbin.org/status/204",
		"https://httpbin.org/status/404",
		"https://httpbin.org/status/500",
		"https://example.org",
		"https://iana.org",
		"https://cloudflare.com",
		"https://wikipedia.org",
		"https://golang.org",
		"https://httpbin.org/delay/1",
		"https://httpbin.org/get",
		"https://httpbin.org/uuid",
		"https://httpbin.org/headers",
		"https://httpstat.us/200",
		"https://httpstat.us/201",
		"https://httpstat.us/301",
		"https://httpstat.us/400",
		"https://httpstat.us/404",
		"https://httpstat.us/500",

		"https://example.com",
		"https://google.com",
		"https://go.dev",
		"https://github.com",
		"https://httpbin.org/status/200",
		"https://httpbin.org/status/404",
		"https://cloudflare.com",
		"https://wikipedia.org",

		"https://example.com",
		"https://google.com",
		"https://go.dev",
		"https://pkg.go.dev",
		"https://github.com",
		"https://httpbin.org/status/200",
		"https://httpbin.org/status/500",
		"https://iana.org",
	}

	urls := make([]string, 0, 100)
	for len(urls) < 100 {
		urls = append(urls, baseURLs...)
	}
	urls = urls[:100]

	for ctx.Err() == nil {
		start := time.Now()
		results := Survey(ctx, urls, 8, true)
		log.Printf("survey done: urls=%d results=%d elapsed=%s", len(urls), len(results), time.Since(start))

		select {
		case <-ctx.Done():
			break
		case <-time.After(2 * time.Second):
		}
	}

	log.Println("shutdown signal received, stopping...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := pprofServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("pprof server shutdown error: %v", err)
	}

	log.Println("shutdown complete")
}

type Getter interface {
	Get(ctx context.Context, url string) int
}

const ErrCode = -1

// Есть список ссылок
// Необходимо написать функцию которая опросит каждый из url-ов в этом списке
// и вернет статусы в том же порядке, в котором они были переданы.
// Кол-во параллельных запросов регулируется параметров parallel
func Survey(ctx context.Context, urls []string, parallel int, isMock bool) []int {
	results := make([]int, 0, len(urls))

	cache := cache.NewCache()

	wg := sync.WaitGroup{}

	chanInput := make(chan string, parallel)
	chanOutput := make(chan int, parallel)

	for i := 0; i < parallel; i++ {
		wg.Go(func() { worker(ctx, chanInput, chanOutput, cache, isMock) })
	}

	go func() {
		for _, url := range urls {
			chanInput <- url
		}
		close(chanInput)
	}()

	go func() {
		wg.Wait()
		close(chanOutput)
	}()

	for v := range chanOutput {
		results = append(results, v)

	}

	return results
}

func worker(ctx context.Context, inChang chan string, outChan chan int, cache *cache.Cache, isMock bool) {

	var client Getter
	if isMock {
		client = NewHTTPClientMock()
	} else {
		client = NewHTTPClient()
	}

	for url := range inChang {

		statusCode, ok := cache.Get(url)
		if !ok {
			statusCode = client.Get(ctx, url)
			cache.Set(url, statusCode)
		}
		outChan <- statusCode

	}
}

type httpClient struct {
	client *http.Client
}

func NewHTTPClient() *httpClient {
	return &httpClient{
		client: &http.Client{},
	}
}

func (h *httpClient) Get(ctx context.Context, url string) int {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return ErrCode
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return ErrCode
	}
	return resp.StatusCode
}

type httpClientMock struct {
}

func NewHTTPClientMock() *httpClientMock {
	return &httpClientMock{}
}

func (h *httpClientMock) Get(ctx context.Context, url string) int {
	time.Sleep(1 * time.Millisecond)
	return 200
}

/*
\\ go >- n worker (parallel) >- go append(result)


wg
results - mutext()
go >- n worker (parallel)
*/

// https://pikabu.ru/story/gde_provodit_livecoding_10699921

/*
ДЗ:
1. net/http
2. Вспомнить синтаксис go
3. Сделать реалищацию воркеров + реализацию на семафорах
4. _test.go, benchmark, benchstat, pprof, тесты табличные (https://go.dev/wiki/TableDrivenTests)
5. обработка ошибок 400, 500, ретраи(https://habr.com/ru/companies/yandex/articles/762678/)
6. ["ya.ru", "google.com", "ya.ru"], кеширование для урлов
7. singleflight

github repo, меня как ревьюера
*/
