package domain

import "testing"

func TestIsValidNumber(t *testing.T) {
	tests := []struct {
		number string
		valid  bool
	}{
		{"12345678903", true}, // valid Luhn
		{"1234567890", false}, // invalid Luhn
		{"", false},
		{"1", false},
		{"abc", false},
	}
	for _, tt := range tests {
		if got := IsValidNumber(tt.number); got != tt.valid {
			t.Errorf("IsValidNumber(%q) = %v, want %v", tt.number, got, tt.valid)
		}
	}
}
