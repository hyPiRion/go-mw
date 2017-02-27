package mw

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

// ErrHandled is the error returned when an error has been translated into a
// response.
var ErrHandled = errors.New("Error is handled and is inside the response")

// IsErrUnhandled returns true if the error is not nill and is not ErrHandled.
func IsErrUnhandled(err error) bool {
	return err != nil && err != ErrHandled
}

// IsErrMissingContextValue returns true if the error is because of a missing
// context value
func IsErrMissingContextValue(err error) bool {
	_, ok := err.(*ErrMissingContextValue)
	return ok
}

// ErrMissingContextValue is returned from functions which extracts data from
// contexts.
type ErrMissingContextValue struct {
	What string
}

func (e *ErrMissingContextValue) Error() string {
	return fmt.Sprintf("Could not find value %s in context", e.What)
}

// NewResponse returns an empty Response to be used by middleware and handlers.
func NewResponse(w http.ResponseWriter) *Response {
	return &Response{Headers: w.Header()}
}

// A Response is the current built up response for this request. Middleware will
// typically set additional headers or add additional context in the
// http.Request.
type Response struct {
	StatusCode int
	Body       interface{}
	Headers    http.Header
}

// Header returns the header map of a Response.
func (htr *Response) Header() http.Header {
	return htr.Headers
}

// Write is a method which always panics. It's implemented so that users can
// utilize functions that works on http.ResponseWriters, which only require
// Header() to work as intended (e.g. http.SetCookie)
func (htr *Response) Write(_ []byte) (int, error) {
	panic("github.com/hypirion/go-mw.Response does not implement Write")
}

// WriteHeader is a method which always panics. It's implemented so that users
// can utilize functions that works on http.ResponseWriters, which only require
// Header() to work as intended (e.g. http.SetCookie)
func (htr *Response) WriteHeader(_ int) {
	panic("github.com/hypirion/go-mw.Response does not implement WriteHeader")
}

// Handler is a function which takes in a respone and a request. Note that, in
// contrast to http.ResponseWriter, this handler can NOT stream data to the
// user.
//
// It's not impossible to stream data, but this is the responsibility of the
// function writing Response into the actual http.ResponseWriter.
type Handler func(*Response, *http.Request) error

// Middleware is a function which takes a Handler and returns one.
type Middleware func(Handler) Handler

// Then chains two Middleware functions and returns a new Middleware function. f
// will be called first, then g.
func (f Middleware) Then(g Middleware) Middleware {
	return func(h Handler) Handler {
		return f(g(h))
	}
}

// id is the identity Middleware: It just returns the handler it was given.
func id(h Handler) Handler {
	return h
}

// Chain takes an arbitrary amount of Middlewares and chains them together. The
// first middleware will be activated first, the second second etc.
func Chain(fs ...Middleware) Middleware {
	var f Middleware = id
	for _, g := range fs {
		f = f.Then(g)
	}
	return f
}

// WithContextValue updates the context of the provided request such that it
// associates key with val. The updated *http.Request is returned.
func WithContextValue(r *http.Request, key, val interface{}) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), key, val))
}
