package main

import (
	"strings"
)

type ErrorType int

const (
	ErrInternal ErrorType = iota
)

// Error provides a high-level object useful for communicating errors.
type Error struct {
	// category is a type of error, useful for distinguishing between error states.
	category ErrorType
	// original is an error raised by an underlying Go library used by libinsights.
	original *error
	// message is a string with placeholders for values, possibly translated.
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
		original:  original,
		message:   message,
		variables: variables,
	}
}

func (e *Error) String() string {
	result := e.message
	if e.variables == nil {
		return result
	}
	for key, value := range *e.variables {
		result = strings.Replace(result, "{"+key+"}", value, -1)
	}
	return result
}
