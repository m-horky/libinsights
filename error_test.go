package main

import (
	"testing"
)

func TestNewError(t *testing.T) {
	tests := []struct {
		name     string
		err      Error
		expected string
	}{
		{
			name: "no variables",
			err: NewError(
				ErrInternal,
				nil,
				"Got error.",
				nil,
			),
			expected: "Got error.",
		},
		{
			name: "one variable",
			err: NewError(
				ErrInternal,
				nil,
				"Got error code {code}",
				&map[string]string{"code": "418"},
			),
			expected: "Got error code 418",
		},
		{
			name: "two variables",
			err: NewError(ErrInternal,
				nil,
				"Got error code {code} (cause: {cause})",
				&map[string]string{"code": "418", "cause": "teapot"},
			),
			expected: "Got error code 418 (cause: teapot)",
		},
		{
			name: "unicode",
			err: NewError(
				ErrInternal,
				nil,
				"Něco se rozbilo ({cause})",
				&map[string]string{"cause": "neznámý důvod"},
			),
			expected: "Něco se rozbilo (neznámý důvod)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tt.err.String()
			if actual != tt.expected {
				t.Errorf("'%s': wanted '%v', got '%v'", tt.name, tt.expected, actual)
			}
		})
	}
}
