package serve

import (
	"bytes"
	"expvar"
	"flag"
	"log"
	"net/http"

	"github.com/dhconnelly/sss/cache"
)

var (
	cacheSize   = flag.Int("cacheSize", 10*1000*1000 /* 10 MB */, "cache size in bytes")
	cacheHits   = expvar.NewMap("cacheHits")
	cacheMisses = expvar.NewMap("cacheMisses")
)

type cachedHandler struct {
	cache *cache.Cache
	h     http.Handler
}

func (h *cachedHandler) serveCached(
	resp http.ResponseWriter, req *http.Request, data cache.CachedData,
) {
	resp.Header().Add("Content-Type", data.ContentType)
	resp.WriteHeader(http.StatusOK)
	_, err := resp.Write(data.Data)
	if err != nil {
		log.Printf("failed to write cached response for %s: %s", req.URL.Path, err)
	}
}

type cachedResponseWriter struct {
	req *http.Request
	http.ResponseWriter
	header http.Header
	buf    *bytes.Buffer
	status int
}

func (w cachedResponseWriter) Write(b []byte) (int, error) {
	w.buf.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w cachedResponseWriter) WriteHeader(status int) {
	w.header = w.Header().Clone()
	w.ResponseWriter.WriteHeader(status)
	if status < 400 {
		cacheMisses.Add(w.req.URL.Path, 1)
	}
}

func (h *cachedHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	// try to fetch it from the cache
	if data, ok := h.cache.Get(req.URL.Path); ok {
		cacheHits.Add(req.URL.Path, 1)
		h.serveCached(resp, req, data)
		return
	}

	// otherwise delegate to the underlying handler
	w := cachedResponseWriter{req: req, ResponseWriter: resp, buf: &bytes.Buffer{}}
	h.h.ServeHTTP(w, req)

	// cache the response, but only if we actually read some data
	if len(w.buf.Bytes()) > 0 {
		data := cache.CachedData{
			Data:        w.buf.Bytes(),
			ContentType: w.Header().Get("Content-Type"),
		}
		h.cache.Put(req.URL.Path, data)
	}
}

func newCachedHandler(h http.Handler) http.Handler {
	return &cachedHandler{cache: cache.New(*cacheSize), h: h}
}
