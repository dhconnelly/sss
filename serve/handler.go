package serve

import (
	"flag"
	"net/http"
)

var (
	enableCaching = flag.Bool("enableCaching", true, "whether to enable caching")
)

func newGetHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodGet {
			h.ServeHTTP(resp, req)
		} else {
			resp.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
}

func NewHandler(root http.FileSystem) http.Handler {
	var h http.Handler
	h = http.FileServer(root)
	if *enableCaching {
		h = newCachedHandler(h)
	}
	h = newGetHandler(h)
	h = newObservedHandler(h)
	return h
}
