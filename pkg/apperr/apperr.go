// Package apperr defines the single error shape returned by every HTTP
// endpoint in the service: {"error": {"code": "...", "message": "..."}}.
package apperr

import "net/http"

// Error is a domain error carrying an HTTP status, a stable machine-readable
// code, and a human-readable message. Handlers across all modules construct
// these instead of returning ad-hoc errors so the HTTP layer can render a
// single, consistent JSON shape.
type Error struct {
	Status  int    `json:"-"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string {
	return e.Message
}

// New builds an Error with an arbitrary status code.
func New(status int, code, message string) *Error {
	return &Error{Status: status, Code: code, Message: message}
}

func BadRequest(code, message string) *Error {
	return New(http.StatusBadRequest, code, message)
}

func Unauthorized(code, message string) *Error {
	return New(http.StatusUnauthorized, code, message)
}

func Forbidden(code, message string) *Error {
	return New(http.StatusForbidden, code, message)
}

func NotFound(code, message string) *Error {
	return New(http.StatusNotFound, code, message)
}

func Conflict(code, message string) *Error {
	return New(http.StatusConflict, code, message)
}

func Internal(code, message string) *Error {
	return New(http.StatusInternalServerError, code, message)
}

// Envelope is the JSON body shape: {"error": {...}}.
type Envelope struct {
	Error *Error `json:"error"`
}
