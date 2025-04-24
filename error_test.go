package main

import (
	"errors"
	"testing"
)

func TestError(t *testing.T) {
	fakeCause := errors.New("fake cause")

	tests := []struct {
		name     string
		err      Error
		asString string
		asError  error
	}{
		{
			name: "no variables",
			err: NewError(
				ErrInternal,
				nil,
				"Got error.",
				nil,
			),
			asString: "Got error.",
			asError:  errors.New("Internal error: Got error."),
		},
		{
			name: "one variable",
			err: NewError(
				ErrInternal,
				nil,
				"Got error code {code}",
				&map[string]string{"code": "418"},
			),
			asString: "Got error code 418",
			asError:  errors.New("Internal error: Got error code 418"),
		},
		{
			name: "two variables",
			err: NewError(ErrInternal,
				nil,
				"Got error code {code} (cause: {cause})",
				&map[string]string{"code": "418", "cause": "teapot"},
			),
			asString: "Got error code 418 (cause: teapot)",
			asError:  errors.New("Internal error: Got error code 418 (cause: teapot)"),
		},
		{
			name: "unicode",
			err: NewError(
				ErrInternal,
				nil,
				"Něco se rozbilo ({cause})",
				&map[string]string{"cause": "neznámý důvod"},
			),
			asString: "Něco se rozbilo (neznámý důvod)",
			asError:  errors.New("Internal error: Něco se rozbilo (neznámý důvod)"),
		},
		{
			name: "with cause",
			err: NewError(
				ErrInternal,
				&fakeCause,
				"Library failure",
				nil,
			),
			asString: "Library failure",
			asError:  errors.New("Internal error: Library failure (fake cause)"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tt.err.String()
			if actual != tt.asString {
				t.Errorf("'%s': wanted '%v', got '%v'", tt.name, tt.asString, actual)
			}
			if tt.err.Error() != tt.asError.Error() {
				t.Errorf("'%s': wanted '%v', got '%v'", tt.name, tt.asError.Error(), tt.err.Error())
			}
		})
	}
}
