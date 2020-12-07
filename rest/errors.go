package rest

import (
	"context"
	"net/http"

	"github.com/rs/rest-layer/resource"
)

var (
	// ErrNotFound represents a 404 HTTP error.
	ErrNotFound = &Error{Code: http.StatusNotFound}
	// ErrForbidden represents a 403 HTTP error.
	ErrForbidden = &Error{Code: http.StatusForbidden}
	// ErrPreconditionFailed happens when a conditional request condition is not met.
	ErrPreconditionFailed = &Error{Code: http.StatusPreconditionFailed, Message: "Precondition Failed"}
	// ErrConflict happens when another thread or node modified the data
	// concurrently with our own thread in such a way we can't securely apply
	// the requested changes.
	ErrConflict = &Error{Code: http.StatusConflict}
	// ErrInvalidMethod happens when the used HTTP method is not supported for
	// this resource.
	ErrInvalidMethod = &Error{Code: http.StatusMethodNotAllowed, Message: "Invalid Method"}
	// ErrClientClosedRequest is returned when the client closed the connection
	// before the server was able to finish processing the request.
	ErrClientClosedRequest = &Error{Code: 499, Message: "Client Closed Request"}
	// ErrNotImplemented happens when a requested feature is not implemented.
	ErrNotImplemented = &Error{Code: http.StatusNotImplemented}
	// ErrGatewayTimeout is returned when the specified timeout for the request
	// has been reached before the server was able to process it.
	ErrGatewayTimeout = &Error{Code: http.StatusGatewayTimeout, Message: "Deadline Exceeded"}
	// ErrUnknown is thrown when the origin of the error can't be identified.
	ErrUnknown = &Error{Code: 520, Message: "Unknown Error"}
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
	// Wrapped original error
	Err error
}

// NewError returns a rest.Error from an standard error.
//
// If the the inputted error is recognized, the appropriate rest.Error is mapped.
func NewError(err error) *Error {
	if err == nil {
		return nil
	}
	e, ok := err.(*Error)
	if ok {
		return e
	}
	switch err {
	case context.Canceled:
		e = ErrClientClosedRequest
	case context.DeadlineExceeded:
		e = ErrGatewayTimeout
	case resource.ErrNotFound:
		e = ErrNotFound
	case resource.ErrForbidden:
		e = ErrForbidden
	case resource.ErrConflict:
		e = ErrConflict
	case resource.ErrNotImplemented:
		e = ErrNotImplemented
	case resource.ErrNoStorage:
		e = &Error{Code: 501}
	default:
		e = &Error{Code: 520}
	}
	//Wrap the original error
	e.Err = err

	return e
}

// Error returns the error as string
func (e *Error) Error() string {
	txt := e.Message
	if e.Err != nil {
		if len(txt) > 0 {
			txt += ": "
		}
		txt += e.Err.Error()
	}
	return txt
}

func (e *Error) Unwrap() error {
	return e.Err
}
