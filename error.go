package libinsights

import (
	"fmt"
	"strings"
)

type ErrorType string

const (
	ErrInternal ErrorType = "Internal error"
	ErrParsing  ErrorType = "Parsing error"
)

// Error provides a high-level object useful for communicating errors.
type Error struct {
	// category is a type of error, useful for distinguishing between error states.
	category ErrorType
	// cause is an error raised by an underlying Go library used by libinsights.
	cause *error
	// message is a asString with placeholders for values, possibly translated.
	message string
	// variables are
	variables *map[string]string
}

// NewError creates a message.
func NewError(
	category ErrorType,
	original *error,
	message string,
	variables *map[string]string,
) Error {
	return Error{
		category:  category,
		cause:     original,
		message:   message,
		variables: variables,
	}
}

// String provides a human-friendly error message.
func (e Error) String() string {
	result := e.message
	if e.variables == nil {
		return result
	}
	for key, value := range *e.variables {
		result = strings.Replace(result, "{"+key+"}", value, -1)
	}
	return result
}

// Error provides developer-friendly error message.
func (e Error) Error() string {
	result := fmt.Sprintf("%s: %s", string(e.category), e.String())
	if e.cause != nil {
		result = fmt.Sprintf("%s (%v)", result, *e.cause)
	}
	return result
}
