package logger

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewWithWriter(t *testing.T) {
	var buf bytes.Buffer
	log := NewWithWriter(&buf)
	log.Info().Msg("hello")

	var m map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &m)
	assert.NoError(t, err)
	assert.Equal(t, "info", m["level"])
	assert.Equal(t, "hello", m["message"])
}
