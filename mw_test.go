package mw

import (
	"errors"
	"net/http"
	"testing"
)

func noopHandler(r *http.Request) (*Response, error) {
	return NewResponse(), nil
}

type bodySetter struct {
	Body interface{}
}

func (b *bodySetter) setBody(h Handler) Handler {
	return func(r *http.Request) (resp *Response, err error) {
		resp, err = h(r)
		if err == nil {
			resp.Body = b.Body
		}
		return
	}
}

func TestChaining1(t *testing.T) {
	set1 := bodySetter{1}
	set2 := bodySetter{2}
	handler := Chain(set1.setBody, set2.setBody)(noopHandler)
	r, _ := handler(&http.Request{})
	if r.Body != set1.Body {
		t.Error("set1 earlier in chain than set2, yet body does not contain 1: Contains %#v", r.Body)
	}
	handler = Middleware(set2.setBody).Then(set1.setBody)(noopHandler)
	r, _ = handler(&http.Request{})
	if r.Body != set2.Body {
		t.Error("set2 earlier in chain than set1, yet body does not contain 2: Contains %#v", r.Body)
	}
}

type contextMw struct {
	key interface{}
	val interface{}
}

func (cmw *contextMw) setContext(h Handler) Handler {
	return func(r *http.Request) (resp *Response, err error) {
		return h(WithContextValue(r, cmw.key, cmw.val))
	}
}

func (cmw *contextMw) checkContext(h Handler) Handler {
	return func(r *http.Request) (resp *Response, err error) {
		if r.Context().Value(cmw.key) != cmw.val {
			return nil, errors.New("Context value not found")
		}
		resp, err = h(r)
		return
	}
}

func TestWithContextValue(t *testing.T) {
	cmw1 := contextMw{1, 2}
	cmw2 := contextMw{2, 3}
	handler := Chain(cmw1.setContext, cmw1.checkContext, cmw2.setContext, cmw1.checkContext, cmw2.checkContext)(noopHandler)
	_, err := handler(&http.Request{})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
}
