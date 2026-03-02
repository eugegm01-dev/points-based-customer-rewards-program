package accrual_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/accrual"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_GetOrderStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/orders/123", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"order":"123","status":"PROCESSED","accrual":500}`))
	}))
	defer ts.Close()

	client := accrual.NewClient(ts.URL)
	resp, err := client.GetOrderStatus(context.Background(), "123")
	require.NoError(t, err)
	assert.Equal(t, "123", resp.Order)
	assert.Equal(t, "PROCESSED", resp.Status)
	assert.NotNil(t, resp.Accrual)
	assert.Equal(t, 500.0, *resp.Accrual)
}

func TestClient_GetOrderStatus_TooManyRequests(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer ts.Close()

	client := accrual.NewClient(ts.URL)
	_, err := client.GetOrderStatus(context.Background(), "123")
	assert.ErrorIs(t, err, accrual.ErrTooManyRequests)
}
func TestClient_GetOrderStatusWithRetry_SuccessAfterRetries(t *testing.T) {
	attempts := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"order":"123","status":"PROCESSED","accrual":500}`))
	}))
	defer ts.Close()

	client := accrual.NewClient(ts.URL)
	resp, err := client.GetOrderStatusWithRetry(context.Background(), "123")
	require.NoError(t, err)
	assert.Equal(t, "123", resp.Order)
	assert.Equal(t, "PROCESSED", resp.Status)
	assert.Equal(t, 500.0, *resp.Accrual)
	assert.Equal(t, 3, attempts) // should have retried twice
}

func TestClient_GetOrderStatusWithRetry_ContextCancel(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer ts.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately
	client := accrual.NewClient(ts.URL)
	_, err := client.GetOrderStatusWithRetry(ctx, "123")
	assert.ErrorIs(t, err, context.Canceled)
}

func TestClient_GetOrderStatusWithRetry_NonRetryableError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	client := accrual.NewClient(ts.URL)
	_, err := client.GetOrderStatusWithRetry(context.Background(), "123")
	assert.Error(t, err) // should not be retried, returns immediately
}
func TestGetRetryAfter(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   time.Duration
	}{
		{"empty header", "", time.Minute},
		{"seconds", "30", 30 * time.Second},
		{"invalid", "abc", time.Minute},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{Header: http.Header{"Retry-After": []string{tt.header}}}
			got := accrual.GetRetryAfter(resp)
			assert.Equal(t, tt.want, got)
		})
	}
	// Test nil response
	assert.Equal(t, time.Minute, accrual.GetRetryAfter(nil))
}
