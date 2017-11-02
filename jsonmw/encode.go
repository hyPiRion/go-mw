// Copyright 2017 Jean Niklas L'orange.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package jsonmw

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/hypirion/go-mw"
)

type handlerError struct {
	Err error `json:"error"`
}

func (he handlerError) Error() string {
	return he.Err.Error()
}

type encoder struct {
	sub mw.Handler
}

func (e *encoder) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.Contains(r.Header.Get("Accept"), "application/json") {
		http.Error(w, http.StatusText(http.StatusNotAcceptable), http.StatusNotAcceptable)
		return
	}
	resp := mw.NewResponse(w)
	err := e.sub(resp, r)
	resp.Headers.Set("Content-Type", "application/json")
	// Unhandled errors? People should really add an error transformer into a
	// proper response. Perhaps panic is more appropriate.
	if mw.IsErrUnhandled(err) {
		resp.Body = handlerError{err}
		resp.StatusCode = http.StatusInternalServerError
	}
	w.WriteHeader(resp.StatusCode)
	json.NewEncoder(w).Encode(resp.Body)
	// TODO: Hook to notify encoding failed somehow?
}

// NewEncoder takes a go-mw Handler and converts it into an http.Handler. It
// will encode the body as JSON.
func NewEncoder(h mw.Handler) http.Handler {
	return &encoder{h}
}
