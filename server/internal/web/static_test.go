package web

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func serve(t *testing.T, path string) *httptest.ResponseRecorder {
	t.Helper()
	files := fstest.MapFS{
		"index.html":         {Data: []byte("x")},
		"assets/main-abc.js": {Data: []byte("x")},
	}
	rec := httptest.NewRecorder()
	Static(files).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestClientComesFromTheGivenDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !HasClient(Client(dir)) {
		t.Fatal("no index.html found in the given directory")
	}
	if HasClient(Client(t.TempDir())) {
		t.Fatal("an empty directory counted as a client")
	}
}

func TestClientIsBuiltInWithoutADirectory(t *testing.T) {
	if _, err := fs.Stat(Client(""), ".gitkeep"); err != nil {
		t.Fatalf("embedded client files missing: %v", err)
	}
}

func TestHashedAssetsAreCachedForGood(t *testing.T) {
	rec := serve(t, "/assets/main-abc.js")
	if got := rec.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Fatalf("Cache-Control = %q", got)
	}
}

func TestPagesAreRevalidated(t *testing.T) {
	rec := serve(t, "/")
	if rec.Code != http.StatusOK || rec.Header().Get("Cache-Control") != "no-cache" {
		t.Fatalf("code %d, Cache-Control %q", rec.Code, rec.Header().Get("Cache-Control"))
	}
}

func TestMissingAssetsAreNotCached(t *testing.T) {
	rec := serve(t, "/assets/main-old.js")
	if rec.Code != http.StatusNotFound || rec.Header().Get("Cache-Control") != "no-cache" {
		t.Fatalf("code %d, Cache-Control %q", rec.Code, rec.Header().Get("Cache-Control"))
	}
}
