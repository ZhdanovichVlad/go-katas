package client

import (
	"context"
	"fmt"
	"math/rand/v2"
	"net/http"
	"time"

	"github.com/ZhdanovichVlad/go-katas/http_survey/internal/surveyerr"
	"golang.org/x/sync/singleflight"
)

var answers = []int{200, 201, 202, 204, 400, 401, 403, 404, 500, 503}
var lenAnswers = len(answers)

type httpClient struct {
	client *http.Client
	sf     singleflight.Group
}

func NewHTTPClient() *httpClient {
	return &httpClient{
		client: &http.Client{},
		sf:     singleflight.Group{},
	}
}

func (h *httpClient) Get(ctx context.Context, url string) int {
	v, err, _ := h.sf.Do(url, func() (interface{}, error) {
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return surveyerr.ErrCode, nil
		}

		resp, err := h.client.Do(req)
		if err != nil {
			return surveyerr.ErrCode, nil
		}
		defer resp.Body.Close()

		return resp.StatusCode, nil
	})

	if err != nil {
		return surveyerr.ErrCode
	}

	res, ok := v.(int)
	if !ok {
		fmt.Println("v is not an int")
		return surveyerr.ErrCode
	}

	return res
}

type httpClientMock struct {
	sf singleflight.Group
}

func NewHTTPClientMock() *httpClientMock {
	return &httpClientMock{
		sf: singleflight.Group{},
	}
}

func (h *httpClientMock) Get(ctx context.Context, url string) int {
	v, err, _ := h.sf.Do(url, func() (interface{}, error) {
		time.Sleep(100 * time.Millisecond)

		return answers[rand.IntN(lenAnswers)], nil
	})

	if err != nil {
		return surveyerr.ErrCode
	}

	res, ok := v.(int)
	if !ok {
		fmt.Println("v is not an int")
		return surveyerr.ErrCode
	}

	return res
}
