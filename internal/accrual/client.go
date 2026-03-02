// Package accrual provides a client for the external accrual system.
package accrual

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
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

// NewClient creates a new AccrualClient.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetOrderStatus fetches the status of an order from the accrual system.
// It handles 429 Too Many Requests by returning ErrTooManyRequests.
func (c *Client) GetOrderStatus(ctx context.Context, number string) (*AccrualResponse, error) {
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
		// Order not registered in accrual system yet
		return nil, nil
	case http.StatusTooManyRequests:
		return nil, ErrTooManyRequests
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
	// Parse HTTP date format if needed, defaulting to 1 min for simplicity
	return time.Minute
}

const (
	maxRetries = 5
	baseDelay  = 100 * time.Millisecond
	maxDelay   = 10 * time.Second
)

// GetOrderStatusWithRetry calls GetOrderStatus with exponential backoff on 429.
func (c *Client) GetOrderStatusWithRetry(ctx context.Context, number string) (*AccrualResponse, error) {
	var resp *AccrualResponse
	var err error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff with jitter
			delay := baseDelay * (1 << attempt)
			if delay > maxDelay {
				delay = maxDelay
			}
			// Add jitter (±25%)
			jitter := time.Duration(rand.Int63n(int64(delay / 4)))
			delay = delay - jitter/2 + jitter

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		resp, err = c.GetOrderStatus(ctx, number)
		if err == nil {
			return resp, nil
		}
		if err != ErrTooManyRequests {
			return nil, err // non-retryable error
		}
		// else retry
	}
	return nil, err // last error
}
