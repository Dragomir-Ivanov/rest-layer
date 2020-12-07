package rest

import (
	"context"
	"errors"
	"testing"

	"github.com/rs/rest-layer/resource"
	"github.com/stretchr/testify/assert"
)

func TestNewError(t *testing.T) {
	assert.Equal(t, ErrClientClosedRequest, NewError(context.Canceled))
	assert.Equal(t, ErrGatewayTimeout, NewError(context.DeadlineExceeded))
	assert.Equal(t, ErrForbidden, NewError(resource.ErrForbidden))
	assert.Equal(t, ErrNotFound, NewError(resource.ErrNotFound))
	assert.Equal(t, ErrConflict, NewError(resource.ErrConflict))
	assert.Equal(t, ErrNotImplemented, NewError(resource.ErrNotImplemented))
	assert.Nil(t, NewError(nil))
	assert.Equal(t, &Error{Code: 520, Err: errors.New("test")}, NewError(errors.New("test")))
	assert.Equal(t, ErrNotFound, NewError(ErrNotFound))
}

func TestError(t *testing.T) {
	e := &Error{Code: 123, Message: "message"}
	assert.Equal(t, "message", e.Error())
}
