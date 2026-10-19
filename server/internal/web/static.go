// Package web serves the built game client.
package web

import (
	"net/http"
	"strings"
)

const (
	forever    = "public, max-age=31536000, immutable"
	revalidate = "no-cache"
)

// Static serves dir. Hashed build files under /assets/ are cached for good; everything else is revalidated.
func Static(dir string) http.Handler {
	files := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		files.ServeHTTP(&cacheWriter{ResponseWriter: w, hashed: strings.HasPrefix(r.URL.Path, "/assets/")}, r)
	})
}

// cacheWriter picks Cache-Control once the status is known, so a missing asset is never cached.
type cacheWriter struct {
	http.ResponseWriter
	hashed  bool
	written bool
}

func (c *cacheWriter) WriteHeader(code int) {
	if !c.written {
		c.written = true
		policy := revalidate
		if c.hashed && code == http.StatusOK {
			policy = forever
		}
		c.Header().Set("Cache-Control", policy)
	}
	c.ResponseWriter.WriteHeader(code)
}

func (c *cacheWriter) Write(b []byte) (int, error) {
	if !c.written {
		c.WriteHeader(http.StatusOK)
	}
	return c.ResponseWriter.Write(b)
}
