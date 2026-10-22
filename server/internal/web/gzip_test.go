package web

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

var script = strings.Repeat("let x = 1;\n", 500)

var client = fstest.MapFS{
	"index.html":         {Data: []byte("<!doctype html>" + script)},
	"assets/main-abc.js": {Data: []byte(script)},
	"assets/tiny.js":     {Data: []byte("x")},
	"icon.png":           {Data: []byte(script)},
}

func get(t *testing.T, h http.Handler, path, accept string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if accept != "" {
		req.Header.Set("Accept-Encoding", accept)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func gunzip(t *testing.T, b []byte) string {
	t.Helper()
	r, err := gzip.NewReader(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

func TestScriptsAreGzippedForBrowsersThatAcceptIt(t *testing.T) {
	rec := get(t, Static(client), "/assets/main-abc.js", "gzip, deflate, br")
	h := rec.Header()
	if h.Get("Content-Encoding") != "gzip" || !strings.HasPrefix(h.Get("Content-Type"), "text/javascript") {
		t.Fatalf("encoding %q, type %q", h.Get("Content-Encoding"), h.Get("Content-Type"))
	}
	if h.Get("Cache-Control") != forever || h.Get("Vary") != "Accept-Encoding" {
		t.Fatalf("Cache-Control %q, Vary %q", h.Get("Cache-Control"), h.Get("Vary"))
	}
	if rec.Body.Len() >= len(script) || gunzip(t, rec.Body.Bytes()) != script {
		t.Fatalf("body of %d bytes is not the script compressed", rec.Body.Len())
	}
}

func TestThePageIsGzippedAndRevalidated(t *testing.T) {
	rec := get(t, Static(client), "/", "gzip")
	h := rec.Header()
	if h.Get("Content-Encoding") != "gzip" || !strings.HasPrefix(h.Get("Content-Type"), "text/html") || h.Get("Cache-Control") != revalidate {
		t.Fatalf("encoding %q, type %q, Cache-Control %q", h.Get("Content-Encoding"), h.Get("Content-Type"), h.Get("Cache-Control"))
	}
}

func TestPlainFilesForBrowsersThatRefuseGzip(t *testing.T) {
	for _, accept := range []string{"", "br", "gzip;q=0"} {
		rec := get(t, Static(client), "/assets/main-abc.js", accept)
		if rec.Header().Get("Content-Encoding") != "" || rec.Body.String() != script {
			t.Fatalf("Accept-Encoding %q: got encoding %q", accept, rec.Header().Get("Content-Encoding"))
		}
	}
}

func TestPicturesAndTinyFilesAreSentPlain(t *testing.T) {
	for _, path := range []string{"/icon.png", "/assets/tiny.js"} {
		if rec := get(t, Static(client), path, "gzip"); rec.Header().Get("Content-Encoding") != "" {
			t.Fatalf("%s was gzipped", path)
		}
	}
}

func TestAChangedFileIsCompressedAfresh(t *testing.T) {
	dir := t.TempDir()
	page := filepath.Join(dir, "index.html")
	write := func(body string, at time.Time) {
		if err := os.WriteFile(page, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(page, at, at); err != nil {
			t.Fatal(err)
		}
	}
	h := Static(Client(dir))
	write("old"+script, time.Unix(1000, 0))
	get(t, h, "/", "gzip")
	write("new"+script, time.Unix(2000, 0))
	if got := gunzip(t, get(t, h, "/", "gzip").Body.Bytes()); !strings.HasPrefix(got, "new") {
		t.Fatalf("served the old page: %.10q", got)
	}
}
