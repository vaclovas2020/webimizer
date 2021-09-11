package webimizer

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
)

/*
Define default Http Response headers
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

/*
The main struct, where You can define Handler (it is main HttpHandler, which is called only, when Http method is allowed), NotAllowHandler (it is HttpHandler, which is called only if Http method is not allowed) and AllowedMethods ([]string array, which contains allowed HTTP method names)
You must call func Build to build HttpHandler.

In version v1.1 added AllowedOrigins field (optional): use if you want to check Origin header
*/
type HttpHandlerStruct struct {
	NotAllowHandler HttpNotAllowHandler
	Handler         HttpHandler
	AllowedMethods  []string
	AllowedOrigins  []string
}

/*
Http handler for use in IfHttpMethod func
*/
type IfHttpMethodHandler func(http.ResponseWriter, *http.Request)

/*
Helper func to check r.Method and call handler only if needMethod is equal r.Method
*/
func IfHttpMethod(needMethod string, rw http.ResponseWriter, r *http.Request, handler IfHttpMethodHandler) {
	if r.Method == needMethod {
		handler(rw, r)
	}
}

/*
Helper func to check r.Method and call handler only if r.Method is GET
*/
func Get(rw http.ResponseWriter, r *http.Request, handler IfHttpMethodHandler) {
	IfHttpMethod(http.MethodGet, rw, r, handler)
}

/*
Helper func to check r.Method and call handler only if r.Method is HEAD
*/
func Head(rw http.ResponseWriter, r *http.Request, handler IfHttpMethodHandler) {
	IfHttpMethod(http.MethodHead, rw, r, handler)
}

/*
Helper func to check r.Method and call handler only if r.Method is POST
*/
func Post(rw http.ResponseWriter, r *http.Request, handler IfHttpMethodHandler) {
	IfHttpMethod(http.MethodPost, rw, r, handler)
}

/*
Helper func to check r.Method and call handler only if r.Method is PUT
*/
func Put(rw http.ResponseWriter, r *http.Request, handler IfHttpMethodHandler) {
	IfHttpMethod(http.MethodPut, rw, r, handler)
}

/*
Helper func to check r.Method and call handler only if r.Method is DELETE
*/
func Delete(rw http.ResponseWriter, r *http.Request, handler IfHttpMethodHandler) {
	IfHttpMethod(http.MethodDelete, rw, r, handler)
}

/*
Helper func to check r.Method and call handler only if r.Method is CONNECT
*/
func Connect(rw http.ResponseWriter, r *http.Request, handler IfHttpMethodHandler) {
	IfHttpMethod(http.MethodConnect, rw, r, handler)
}

/*
Helper func to check r.Method and call handler only if r.Method is OPTIONS
*/
func Options(rw http.ResponseWriter, r *http.Request, handler IfHttpMethodHandler) {
	IfHttpMethod(http.MethodOptions, rw, r, handler)
}

/*
Helper func to check r.Method and call handler only if r.Method is TRACE
*/
func Trace(rw http.ResponseWriter, r *http.Request, handler IfHttpMethodHandler) {
	IfHttpMethod(http.MethodTrace, rw, r, handler)
}

/*
Helper func to check r.Method and call handler only if r.Method is PATCH
*/
func Patch(rw http.ResponseWriter, r *http.Request, handler IfHttpMethodHandler) {
	IfHttpMethod(http.MethodPatch, rw, r, handler)
}

/*
It is HttpHandler, which is called only if Http method is not allowed
*/
type HttpNotAllowHandler func(http.ResponseWriter, *http.Request)

/*
Build HttpHandler, which can by used in http.Handle (but not in http.HandleFunc, because only http.Handle call ServeHTTP)
*/
func (builder HttpHandlerStruct) Build() HttpHandler {
	return HttpHandler(func(w http.ResponseWriter, r *http.Request) {
		builder.notAllowed(r, func(rw http.ResponseWriter, r *http.Request) {
			if builder.NotAllowHandler != nil {
				builder.NotAllowHandler(rw, r)
			} else {
				fmt.Fprint(rw, "Bad Request")
			}
		})(w, r)
	})
}

/*
It is main HttpHandler, which is called only, when Http method is allowed
*/
type HttpHandler func(http.ResponseWriter, *http.Request)

/*
Compressing Http response by using gzipResponseWriter (only if Accept-Encoding request header is set and contains gzip value) and also add DefaultHttpHeaders to Http response
*/
func (fn HttpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, v := range DefaultHTTPHeaders {
		if len(v) == 2 {
			w.Header().Set(v[0], v[1])
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

func (fn HttpHandlerStruct) checkOrigins(r *http.Request) bool {
	for _, origin := range fn.AllowedOrigins {
		if origin == r.Header.Get("Origin") {
			return true
		}
	}
	return false
}

func (fn HttpHandlerStruct) notAllowed(r *http.Request, notAllowed HttpHandler) HttpHandler {
	hasOrigins := len(fn.AllowedOrigins) > 0
	for _, method := range fn.AllowedMethods {
		if method == r.Method && (!hasOrigins || fn.checkOrigins(r)) {
			return fn.Handler
		}
	}
	return notAllowed
}

/*
struct for serving filesystem
*/
type neuteredFileSystem struct {
	fs http.FileSystem
	w  http.ResponseWriter
	r  *http.Request
}

/*
Create http Handler for serving files in fsPath directory.
If file not found return 404 status and serve error404.html if exist
*/
func NewFileServerHandler(fsPath string) HttpHandler {
	return HttpHandler(func(rw http.ResponseWriter, r *http.Request) {
		http.FileServer(neuteredFileSystem{fs: http.Dir(fsPath), w: rw, r: r}).ServeHTTP(rw, r)
	})
}

/*
Read and send requested file to client
If file not found return 404 status and serve 404 document file if error404.html exist
*/
func (nfs neuteredFileSystem) Open(path string) (http.File, error) {
	errorHandler := func() (http.File, error) {
		f, err := nfs.fs.Open("/error404.html")
		if err != nil {
			return nil, err
		}
		nfs.w.Header().Set("Content-Type", "text/html; charset=utf-8")
		nfs.w.WriteHeader(http.StatusNotFound)
		return f, nil
	}
	f, err := nfs.fs.Open(path)
	if err != nil {
		return errorHandler()
	}

	s, _ := f.Stat()
	if s.IsDir() {
		index := filepath.Join(path, "index.html")
		if _, err := nfs.fs.Open(index); err != nil {
			closeErr := f.Close()
			if closeErr != nil {
				return errorHandler()
			}

			return errorHandler()
		}
	}

	return f, nil
}
