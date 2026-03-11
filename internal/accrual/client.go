// Package accrual provides a client for the external accrual system.
package accrual

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// ErrTooManyRequests indicates the accrual system rate limit was hit.
var ErrTooManyRequests = errors.New("too many requests")

// AccrualResponse represents the JSON response from the accrual system.
type AccrualResponse struct {
	Order   string   `json:"order"`
	Status  string   `json:"status"`
	Accrual *float64 `json:"accrual,omitempty"`
}

// Client communicates with the external accrual system.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// RateLimitError includes Retry-After duration from 429 response.
type RateLimitError struct {
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string { return "rate limit exceeded" }

// NewClient creates a new AccrualClient.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// doRequest makes a single HTTP request to accrual system (no retries).
func (c *Client) doRequest(ctx context.Context, number string) (*AccrualResponse, error) {
	url := fmt.Sprintf("%s/api/orders/%s", c.baseURL, number)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var accrual AccrualResponse
		if err := json.NewDecoder(resp.Body).Decode(&accrual); err != nil {
			return nil, err
		}
		return &accrual, nil
	case http.StatusNoContent:
		return nil, nil
	case http.StatusTooManyRequests:
		retryAfter := GetRetryAfter(resp) // ✅ извлекаем из хедера
		return nil, &RateLimitError{RetryAfter: retryAfter}
	default:
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
}

// GetRetryAfter extracts the Retry-After header value in seconds.
func GetRetryAfter(resp *http.Response) time.Duration {
	if resp == nil {
		return time.Minute
	}
	retry := resp.Header.Get("Retry-After")
	if retry == "" {
		return time.Minute
	}
	if seconds, err := strconv.Atoi(retry); err == nil {
		return time.Duration(seconds) * time.Second
	}
	return time.Minute
}

const (
	maxRetries = 5
	baseDelay  = 100 * time.Millisecond
	maxDelay   = 10 * time.Second
)

// GetOrderStatus calls accrual system with transparent retries on 429.
// This is the PUBLIC method — callers don't know about retries.
func (c *Client) GetOrderStatus(ctx context.Context, number string) (*AccrualResponse, error) {
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Экспоненциальный бэк-офф с джиттером
			delay := baseDelay * (1 << attempt)
			if delay > maxDelay {
				delay = maxDelay
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		resp, err := c.doRequest(ctx, number) // приватный метод без ретраев
		if err == nil {
			return resp, nil
		}

		// Ретраим только на 429
		var rlErr *RateLimitError
		if errors.As(err, &rlErr) {
			delay := rlErr.RetryAfter
			if delay == 0 {
				delay = baseDelay * (1 << attempt)
				if delay > maxDelay {
					delay = maxDelay
				}
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
			continue
		}

		lastErr = err
	}
	return nil, lastErr
}
