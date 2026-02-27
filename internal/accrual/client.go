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
