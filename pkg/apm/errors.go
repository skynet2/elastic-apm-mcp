package apm

import (
	"errors"
	"fmt"
)

var (
	ErrUnauthorized = errors.New("apm: unauthorized")
	ErrNotFound     = errors.New("apm: not found")
	ErrForbidden    = errors.New("apm: forbidden")
)

type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("apm: HTTP %d: %s", e.StatusCode, e.Message)
}
