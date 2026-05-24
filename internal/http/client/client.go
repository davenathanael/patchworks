package client

import (
	"net/http"
	"time"
)

const fetchTimeout = 1500 * time.Millisecond

type Client struct {
	httpClient *http.Client
}

func New() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: fetchTimeout},
	}
}
