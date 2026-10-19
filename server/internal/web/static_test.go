package web

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func serve(t *testing.T, path string) *httptest.ResponseRecorder {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"index.html", "assets/main-abc.js"} {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	rec := httptest.NewRecorder()
	Static(dir).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
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
