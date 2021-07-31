package webimizer

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"strings"
)

/* Define default Http Response headers
Example:
    [][]string{
		{"x-content-type-options", "nosniff"},
		{"x-frame-options", "SAMEORIGIN"},
		{"x-xss-protection", "1; mode=block"},
	} // define default headers
*/
var DefaultHTTPHeaders [][]string

type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

/* The main struct, where You can define Handler (it is main HttpHandler, which is called only, when Http method is allowed), NotAllowHandler (it is HttpHandler, which is called only if Http method is not allowed) and AllowedMethods ([]string array, which contains allowed HTTP method names)
You must call func Build to build HttpHandler  */
type HttpHandlerStruct struct {
	NotAllowHandler HttpNotAllowHandler
	Handler         HttpHandler
	AllowedMethods  []string
}

/* It is HttpHandler, which is called only if Http method is not allowed */
type HttpNotAllowHandler func(http.ResponseWriter, *http.Request)

/* Build HttpHandler, which can by used in http.Handle and http.HandleFunc */
func (builder HttpHandlerStruct) Build() HttpHandler {
	return HttpHandler(func(w http.ResponseWriter, r *http.Request) {
		builder.notAllowed(w, r, builder.Handler, func(rw http.ResponseWriter, r *http.Request) {
			if builder.NotAllowHandler != nil {
				builder.NotAllowHandler(rw, r)
			} else {
				fmt.Fprint(rw, "Bad Request")
			}
		}, builder.AllowedMethods)(w, r)
	})
}

/* It is main HttpHandler, which is called only, when Http method is allowed */
type HttpHandler func(http.ResponseWriter, *http.Request)

/* Compressing Http response by using gzipResponseWriter (only if Accept-Encoding request header is set and contains gzip value) and also add DefaultHttpHeaders to Http response */
func (fn HttpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if DefaultHTTPHeaders != nil {
		for _, v := range DefaultHTTPHeaders {
			if len(v) == 2 {
				w.Header().Set(v[0], v[1])
			}
		}
	}
	if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		fn(w, r)
		return
	}
	w.Header().Set("Content-Encoding", "gzip")
	gz := gzip.NewWriter(w)
	defer gz.Close()
	gzr := gzipResponseWriter{Writer: gz, ResponseWriter: w}
	fn(gzr, r)
}

func (w gzipResponseWriter) Write(b []byte) (int, error) {
	if w.Header().Get("Content-Type") == "" {
		// If no content type, apply sniffing algorithm to un-gzipped body. Test
		w.Header().Set("Content-Type", http.DetectContentType(b))
	}
	return w.Writer.Write(b)
}

func (fn HttpHandlerStruct) notAllowed(w http.ResponseWriter, r *http.Request, mainHandler HttpHandler, notAllowed HttpHandler, supportedMethods []string) HttpHandler {
	for _, method := range supportedMethods {
		if method == r.Method {
			return mainHandler
		}
	}
	return notAllowed
}
