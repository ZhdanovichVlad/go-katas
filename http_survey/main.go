package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"
	"github.com/ZhdanovichVlad/go-katas/http_survey/semaphore"
	//"github.com/ZhdanovichVlad/go-katas/http_survey/worker_pool"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pprofServer := &http.Server{Addr: "localhost:6061"}
	go func() {
		log.Println("pprof listening on http://localhost:6061/debug/pprof/")
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

	urls := make([]string, 0, 1000)
	for len(urls) < 100 {
		urls = append(urls, baseURLs...)
	}
	

	for ctx.Err() == nil {
		start := time.Now()
		results := semaphore.Survey(ctx, urls, 8, true)
		//results := worker_pool.Survey(ctx, urls, 8, true)

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
