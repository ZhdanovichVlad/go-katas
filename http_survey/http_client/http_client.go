package http_client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ZhdanovichVlad/go-katas/http_survey/errors"
	"golang.org/x/sync/singleflight"
)

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
			return errors.ErrCode, nil
		}

		resp, err := h.client.Do(req)
		if err != nil {
			return errors.ErrCode, nil
		}
		defer resp.Body.Close()

		return resp.StatusCode, nil
	})

	if err != nil {
		return errors.ErrCode
	}

	res, ok := v.(int)
	if !ok {
		fmt.Println("v is not an int")
		return errors.ErrCode
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
		return 200, nil
	})

	if err != nil {
		return errors.ErrCode
	}

	res, ok := v.(int)
	if !ok {
		fmt.Println("v is not an int")
		return errors.ErrCode
	}

	return res
}
