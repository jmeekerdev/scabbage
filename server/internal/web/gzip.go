package web

import (
	"bytes"
	"compress/gzip"
	"io/fs"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	minGzipBytes   = 1024     // smaller files gain less than gzip's own overhead
	maxCachedBytes = 32 << 20 // bounds memory when a served directory is rebuilt many times
)

// compressible lists the text types worth gzipping; pictures and sounds are compressed already.
var compressible = map[string]bool{
	".html": true, ".js": true, ".mjs": true, ".css": true, ".svg": true,
	".json": true, ".webmanifest": true, ".txt": true, ".map": true, ".wasm": true,
}

type gzipped struct {
	mod  time.Time
	size int64
	data []byte // nil when gzip doesn't make the file smaller
}

// gzipCache compresses each file once, on its first request, and again only when it changes.
type gzipCache struct {
	files   fs.FS
	mu      sync.Mutex
	entries map[string]gzipped
	bytes   int
}

func newGzipCache(files fs.FS) *gzipCache {
	return &gzipCache{files: files, entries: map[string]gzipped{}}
}

// get returns the named file gzipped, or false when it is missing, not text, or doesn't shrink.
func (c *gzipCache) get(name string) ([]byte, time.Time, bool) {
	if !compressible[path.Ext(name)] {
		return nil, time.Time{}, false
	}
	info, err := fs.Stat(c.files, name)
	if err != nil || !info.Mode().IsRegular() || info.Size() < minGzipBytes {
		return nil, time.Time{}, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[name]
	if !ok || !e.mod.Equal(info.ModTime()) || e.size != info.Size() {
		e = gzipped{mod: info.ModTime(), size: info.Size(), data: c.compress(name)}
		c.store(name, e)
	}
	return e.data, e.mod, e.data != nil
}

func (c *gzipCache) compress(name string) []byte {
	raw, err := fs.ReadFile(c.files, name)
	if err != nil {
		return nil
	}
	var buf bytes.Buffer
	w, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression) // a valid level never errors
	_, _ = w.Write(raw)                                     // writes to a bytes.Buffer don't fail
	_ = w.Close()
	if buf.Len() >= len(raw) {
		return nil
	}
	return buf.Bytes()
}

func (c *gzipCache) store(name string, e gzipped) {
	c.bytes += len(e.data) - len(c.entries[name].data)
	if c.bytes > maxCachedBytes {
		c.entries, c.bytes = map[string]gzipped{}, len(e.data)
	}
	c.entries[name] = e
}

// fileName is the file a request path names, with directories meaning their index.html.
func fileName(urlPath string) string {
	name := strings.TrimPrefix(path.Clean("/"+urlPath), "/")
	if name == "" || strings.HasSuffix(urlPath, "/") {
		name = path.Join(name, "index.html")
	}
	return name
}

// acceptsGzip reports whether an Accept-Encoding header allows gzip.
func acceptsGzip(header string) bool {
	for _, part := range strings.Split(header, ",") {
		coding, params, _ := strings.Cut(part, ";")
		if strings.TrimSpace(coding) != "gzip" {
			continue
		}
		q, found := strings.CutPrefix(strings.TrimSpace(params), "q=")
		if !found {
			return true
		}
		weight, err := strconv.ParseFloat(q, 64)
		return err == nil && weight > 0
	}
	return false
}
