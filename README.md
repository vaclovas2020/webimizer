# webimizer
Lightweight HTTP framework module written in Go
# Code example
```go
package main

import (
	"fmt"
	"log"
	"net/http"

	app "github.com/vaclovas2020/webimizer"
)

func httpNotAllowFunc(rw http.ResponseWriter, r *http.Request) {
	rw.WriteHeader(http.StatusBadRequest)
	fmt.Fprint(rw, "Bad Request")
}

func main() {
	app.DefaultHTTPHeaders = [][]string{
		{"x-content-type-options", "nosniff"},
		{"x-frame-options", "SAMEORIGIN"},
		{"x-xss-protection", "1; mode=block"},
	} // define default headers
	http.Handle("/", app.HttpHandlerStruct{
		Handler: app.HttpHandler(func(rw http.ResponseWriter, r *http.Request) {
			app.Get(rw, r, func(rw http.ResponseWriter, r *http.Request) {
				fmt.Fprint(rw, "Hello from webimizer. HTTP GET method was used.")
			})
			app.Post(rw, r, func(rw http.ResponseWriter, r *http.Request) {
				fmt.Fprint(rw, "Hello from webimizer. HTTP POST method was used.")
			})
		}), // app.HttpHandler call only if method is allowed
		NotAllowHandler: app.HttpNotAllowHandler(httpNotAllowFunc), // app.HtttpNotAllowHandler call if method is not allowed
		AllowedMethods:  []string{"GET","POST"},                           // define allowed methods
	}.Build())
	log.Fatal(http.ListenAndServe(":8080", nil)) // example server listen on port 8080
}
```
