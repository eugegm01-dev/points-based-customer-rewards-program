// Package middleware provides HTTP middleware (auth, logging, compression).
package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"sync"
)

// GzipResponseWriter wraps http.ResponseWriter to provide gzip compression.
type GzipResponseWriter struct {
	http.ResponseWriter
	writer *gzip.Writer
	code   int
}

// WriteHeader captures the status code and sets Content-Encoding header.
func (w *GzipResponseWriter) WriteHeader(code int) {
	if w.code != 0 {
		return // Header already written
	}
	w.code = code
	if code != http.StatusNoContent && code != http.StatusNotModified {
		w.Header().Set("Content-Encoding", "gzip")
	}
	w.ResponseWriter.WriteHeader(code)
}

// Write compresses the response body if status is not 204.
func (w *GzipResponseWriter) Write(b []byte) (int, error) {
	// Ensure WriteHeader is called first
	if w.code == 0 {
		w.WriteHeader(http.StatusOK)
	}
	if w.code == http.StatusNoContent {
		return 0, nil
	}
	return w.writer.Write(b)
}

// Close closes the gzip writer and flushes any remaining data.
func (w *GzipResponseWriter) Close() error {
	if w.writer != nil {
		return w.writer.Close()
	}
	return nil
}

// gzipPool provides a pool of gzip writers to reduce allocations.
var gzipPool = sync.Pool{
	New: func() interface{} {
		return gzip.NewWriter(io.Discard)
	},
}

// Gzip middleware compresses HTTP responses when client supports it.
func Gzip(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		gz := gzipPool.Get().(*gzip.Writer)
		defer gzipPool.Put(gz)
		gz.Reset(w)
		defer gz.Close()

		gzw := &GzipResponseWriter{
			ResponseWriter: w,
			writer:         gz,
			code:           0,
		}
		next.ServeHTTP(gzw, r)
	})
}
