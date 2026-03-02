package middleware_test

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/middleware"
	"github.com/stretchr/testify/assert"
)

func TestGzip(t *testing.T) {
	handler := middleware.Gzip(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		if _, err := w.Write([]byte("hello")); err != nil {
			t.Fatal(err)
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, "gzip", rr.Header().Get("Content-Encoding"))
	assert.Equal(t, "text/plain", rr.Header().Get("Content-Type"))

	// Decompress and check content
	gr, err := gzip.NewReader(rr.Body)
	assert.NoError(t, err)
	defer gr.Close()
	body, err := io.ReadAll(gr)
	assert.NoError(t, err)
	assert.Equal(t, "hello", string(body))
}
func TestGzip_NoAcceptEncoding(t *testing.T) {
	handler := middleware.Gzip(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		if _, err := w.Write([]byte("hello")); err != nil {
			t.Fatal(err)
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// No Accept-Encoding header
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Empty(t, rr.Header().Get("Content-Encoding"))
	assert.Equal(t, "text/plain", rr.Header().Get("Content-Type"))
	assert.Equal(t, "hello", rr.Body.String())
}
