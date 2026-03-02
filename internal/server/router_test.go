package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestRouter_Health(t *testing.T) {
	log := zerolog.Nop()
	deps := &Dependencies{} // minimal, handlers will panic if called, but health doesn't need deps
	router := NewRouter(log, deps)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRouter_Unknown(t *testing.T) {
	log := zerolog.Nop()
	deps := &Dependencies{}
	router := NewRouter(log, deps)

	req := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}
