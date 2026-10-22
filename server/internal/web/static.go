// Package web serves the built game client.
package web

import (
	"embed"
	"io/fs"
	"net/http"
	"os"
	"strings"
)

// The build copies client/dist here, so the binary carries the game client.
//
//go:embed all:dist
var embedded embed.FS

const (
	forever    = "public, max-age=31536000, immutable"
	revalidate = "no-cache"
)

// Client is the built client in dir, or the copy built into the binary when dir is empty.
func Client(dir string) fs.FS {
	if dir != "" {
		return os.DirFS(dir)
	}
	files, err := fs.Sub(embedded, "dist")
	if err != nil {
		panic(err) // "dist" is a fixed, valid path
	}
	return files
}

// HasClient reports whether files hold a built client's page.
func HasClient(files fs.FS) bool {
	_, err := fs.Stat(files, "index.html")
	return err == nil
}

// Static serves files. Hashed build files under /assets/ are cached for good; everything else is revalidated.
func Static(client fs.FS) http.Handler {
	files := http.FileServer(http.FS(client))
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
