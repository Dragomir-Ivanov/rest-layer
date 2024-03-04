package rest

import (
	"context"
	"errors"
	"net/http"

	"github.com/rs/rest-layer/resource"
)

var (
	// ErrNotFound represents a 404 HTTP error.
	ErrNotFound = &Error{http.StatusNotFound, "Not Found", nil}
	// ErrForbidden represents a 403 HTTP error.
	ErrForbidden = &Error{http.StatusForbidden, "Forbidden", nil}
	// ErrPreconditionFailed happens when a conditional request condition is not met.
	ErrPreconditionFailed = &Error{http.StatusPreconditionFailed, "Precondition Failed", nil}
	// ErrConflict happens when another thread or node modified the data
	// concurrently with our own thread in such a way we can't securely apply
	// the requested changes.
	ErrConflict = &Error{http.StatusConflict, "Conflict", nil}
	// ErrInvalidMethod happens when the used HTTP method is not supported for
	// this resource.
	ErrInvalidMethod = &Error{http.StatusMethodNotAllowed, "Invalid Method", nil}
	// ErrClientClosedRequest is returned when the client closed the connection
	// before the server was able to finish processing the request.
	ErrClientClosedRequest = &Error{499, "Client Closed Request", nil}
	// ErrNotImplemented happens when a requested feature is not implemented.
	ErrNotImplemented = &Error{http.StatusNotImplemented, "Not Implemented", nil}
	// ErrGatewayTimeout is returned when the specified timeout for the request
	// has been reached before the server was able to process it.
	ErrGatewayTimeout = &Error{http.StatusGatewayTimeout, "Deadline Exceeded", nil}
	// ErrUnknown is thrown when the origin of the error can't be identified.
	ErrUnknown = &Error{520, "Unknown Error", nil}
)

// Error defines a REST error with optional per fields error details.
type Error struct {
	// Code defines the error code to be used for the error and for the HTTP
	// status.
	Code int
	// Message is the error message.
	Message string
	// Issues holds per fields errors if any.
	Issues map[string][]interface{}
}

// NewError returns a rest.Error from an standard error.
//
// If the the inputted error is recognized, the appropriate rest.Error is mapped.
func NewError(err error) (error, int) {
	var Err *Error
	if errors.As(err, &Err) {
		return err, Err.Code
	}
	switch err {
	case context.Canceled:
		return ErrClientClosedRequest, ErrClientClosedRequest.Code
	case context.DeadlineExceeded:
		return ErrGatewayTimeout, ErrGatewayTimeout.Code
	case resource.ErrNotFound:
		return ErrNotFound, ErrNotFound.Code
	case resource.ErrForbidden:
		return ErrForbidden, ErrForbidden.Code
	case resource.ErrConflict:
		return ErrConflict, ErrConflict.Code
	case resource.ErrNotImplemented:
		return ErrNotImplemented, ErrNotImplemented.Code
	case resource.ErrNoStorage:
		return &Error{501, err.Error(), nil}, 501
	case nil:
		return nil, 0
	default:
		return &Error{520, err.Error(), nil}, 520
	}
}

// Error returns the error as string
func (e *Error) Error() string {
	return e.Message
}
