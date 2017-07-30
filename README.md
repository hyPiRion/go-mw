# go-mw

[![GoDoc](https://godoc.org/github.com/hyPiRion/go-mw?status.svg)](https://godoc.org/github.com/hyPiRion/go-mw)

HTTP middleware for Go 1.8 and beyond.

## What is go-mw?

go-mw is a small library for making http middleware in Go. It is not a web
framework: Handling routes, rate limiting et al is up to some other library (or
framework, your pick), but handling middleware is where this is key.

## Why go-mw?

go-mw is based upon two observations I have done while programming web servers
with [Ring](https://github.com/ring-clojure/ring) in Clojure:

1. Passing data around as values gives you the power to separate domain logic
   with serialisation and socket concerns
2. Wrapping the context (db transactions, caches, user id etc) in custom types
   means you can implicitly provide Context to the user, and avoid unnecessarily
   many error checks.

The former is not as common in Go as far as I know: All the HTTP frameworks I
have seen so far focus on embracing the `(w *http.ResponseWriter, r
http.Request)` pattern in some way – either by adding additional functions on
top of the `ResponseWriter` (typically `WriteJSON` or something similar) or just
using the plain old `http.ServeHTTP`.

Of course, this is necessary if you can't opt out of the web framework and need
to stream while processing/doing domain logic; whether it is due to performance
or something else. And passing by value will be slower. However, for the vast
majority of users, pass-by-value will not likely be slow enough for it to
matter.

## License

Copyright © 2017 Jean Niklas L'orange

Distributed under the BSD 3-clause license, which is available in the file
LICENSE.
