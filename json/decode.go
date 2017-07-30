package mwjson

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"

	"github.com/hyPiRion/go-mw"
)

// ErrUnsupportedMediaType is returned from a JSON decoder if the Content-Type
// provided is not equal to "application/json"
var ErrUnsupportedMediaType error = handlerError{errors.New(`Unsupported media type: Expected "application/json"`)}

var errorType = reflect.TypeOf((*error)(nil)).Elem()
var responseType = reflect.TypeOf((*mw.Response)(nil))
var requestType = reflect.TypeOf((*http.Request)(nil))

// NewDecoder takes a function on the form `func(*mw.Response, *http.Request,
// *T) error` and transforms it into an mw.Handler.
func NewDecoder(fn interface{}) mw.Handler {
	rfn := reflect.ValueOf(fn)
	rtyp := rfn.Type()
	if rtyp.Kind() != reflect.Func {
		panic("Fn is not function")
	}
	if rtyp.NumIn() != 3 || rtyp.NumOut() != 1 {
		panic(fmt.Sprintf("Fn must have 3 input arguments and 1 result value"))
	}
	if rtyp.In(0) != responseType {
		panic("First argument in fn must be *mw.Response")
	}
	if rtyp.In(1) != requestType {
		panic("Second argument in fn must be *http.Request")
	}
	if rtyp.Out(0) != errorType {
		panic("Output argument from function must be error")
	}
	inputType := rtyp.In(2)
	if inputType.Kind() != reflect.Ptr || inputType.Elem().Kind() != reflect.Struct {
		panic("Third function argument must be a pointer to a struct")
	}
	return (&decoder{inputType, rfn}).handler
}

type decoder struct {
	inputType reflect.Type
	rfn       reflect.Value
}

func (dec *decoder) handler(resp *mw.Response, r *http.Request) error {
	if r.Header.Get("Content-Type") != "application/json" {
		resp.Body = ErrUnsupportedMediaType
		resp.StatusCode = http.StatusUnsupportedMediaType
		return mw.ErrHandled
	}
	rinput := reflect.New(dec.inputType)
	input := rinput.Interface()
	err := json.NewDecoder(r.Body).Decode(input)
	if err != nil {
		resp.Body = handlerError{errors.New("Malformed request body: " + err.Error())}
		resp.StatusCode = http.StatusBadRequest
		return mw.ErrHandled
	}
	rresp := reflect.ValueOf(resp)
	rr := reflect.ValueOf(r)
	fnRes := dec.rfn.Call([]reflect.Value{rresp, rr, rinput})
	err, _ = fnRes[0].Interface().(error)
	return err
}
