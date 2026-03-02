package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type testStruct struct {
	Name  string `validate:"required"`
	Age   int    `validate:"gte=0,lte=130"`
	Email string `validate:"omitempty,email"`
}

func TestValidateStruct(t *testing.T) {
	tests := []struct {
		name     string
		input    testStruct
		wantErrs map[string]string
	}{
		{
			name:     "valid",
			input:    testStruct{Name: "Alice", Age: 30, Email: "alice@example.com"},
			wantErrs: nil,
		},
		{
			name:     "missing name",
			input:    testStruct{Age: 25},
			wantErrs: map[string]string{"Name": "failed validation: required"},
		},
		{
			name:     "age out of range",
			input:    testStruct{Name: "Bob", Age: 200},
			wantErrs: map[string]string{"Age": "failed validation: lte"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidateStruct(tt.input)
			assert.Equal(t, tt.wantErrs, errs)
		})
	}
}
