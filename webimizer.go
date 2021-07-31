package webimizer

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}
type HttpHandlerStruct struct {
	NotAllowHandler HttpNotAllowHandler
	Handler         HttpHandler
	AllowedMethods  []string
}

type HttpNotAllowHandler func(http.ResponseWriter, *http.Request)

func (builder HttpHandlerStruct) Build() HttpHandler {
	return HttpHandler(func(w http.ResponseWriter, r *http.Request) {
		builder.notAllowed(w, r, builder.Handler, func(rw http.ResponseWriter, r *http.Request) {
			rw.WriteHeader(http.StatusBadRequest)
			if builder.NotAllowHandler != nil {
				builder.NotAllowHandler(rw, r)
			} else {
				fmt.Fprint(rw, "Bad Request")
			}
		}, builder.AllowedMethods)(w, r)
	})
}

type HttpHandler func(http.ResponseWriter, *http.Request)

func (fn HttpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("x-content-type-options", "nosniff")
	w.Header().Set("x-frame-options", "SAMEORIGIN")
	w.Header().Set("x-xss-protection", "1; mode=block")
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
