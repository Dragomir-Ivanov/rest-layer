package rest

import (
	"context"
	"errors"
	"testing"

	"github.com/rs/rest-layer/resource"
	"github.com/stretchr/testify/assert"
)

func TestNewError(t *testing.T) {
	{
		e, code := NewError(context.Canceled)
		assert.Equal(t, e, ErrClientClosedRequest)
		assert.Equal(t, code, ErrClientClosedRequest.Code)
	}
	{
		e, code := NewError(context.DeadlineExceeded)
		assert.Equal(t, e, ErrGatewayTimeout)
		assert.Equal(t, code, ErrGatewayTimeout.Code)
	}
	{
		e, code := NewError(resource.ErrForbidden)
		assert.Equal(t, e, ErrForbidden)
		assert.Equal(t, code, ErrForbidden.Code)
	}
	{
		e, code := NewError(resource.ErrConflict)
		assert.Equal(t, e, ErrConflict)
		assert.Equal(t, code, ErrConflict.Code)
	}
	{
		e, code := NewError(resource.ErrNotImplemented)
		assert.Equal(t, e, ErrNotImplemented)
		assert.Equal(t, code, ErrNotImplemented.Code)
	}
	{
		e, code := NewError(nil)
		assert.Nil(t, e)
		assert.Equal(t, code, 0)
	}
	{
		e, code := NewError(errors.New("test"))
		assert.Equal(t, e.Error(), "test")
		assert.Equal(t, code, 520)
	}
	{
		e, code := NewError(resource.ErrNotFound)
		assert.Equal(t, e, ErrNotFound)
		assert.Equal(t, code, ErrNotFound.Code)
	}
}

func TestError(t *testing.T) {
	e := &Error{123, "message", nil}
	assert.Equal(t, "message", e.Error())
}
