package mw

import (
	"net/http"
)

type responseWriterKey int

type handlerAdapter struct {
	sub Handler
}

func (adapter *handlerAdapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	resp := NewResponse(w)
	r = WithContextValue(r, responseWriterKey(0), w)
	adapter.sub(resp, r)
	// Ignore error here I guess, although I would probably reconsider that.
}

// IntoHTTPMiddleware converts go-mw middleware into a net/http middleware, with
// one caveat: The Response input should be treated just like a ResponseWriter,
// i.e. modification of headers after it's handed downstream will (most likely)
// have no effect.
func IntoHTTPMiddleware(m Middleware) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		handler := func(resp *Response, r *http.Request) error {
			w := r.Context().Value(responseWriterKey(0)).(http.ResponseWriter)
			h.ServeHTTP(w, r)
			return nil
		}
		return &handlerAdapter{sub: m(handler)}
	}
}

type handlerInverter struct {
	http.Handler
}

func (hi *handlerInverter) serve(resp *Response, r *http.Request) error {
	w := r.Context().Value(responseWriterKey(0)).(http.ResponseWriter)
	hi.ServeHTTP(w, r)
	return nil
}

// IntoHandler converts an HTTP handler into a go-mw Handler. This is useful
// when you want to utilise go-mw middleware, and also want to print middleware
// errors properly to the client.
func IntoHandler(h http.Handler) Handler {
	return (&handlerInverter{h}).serve
}
