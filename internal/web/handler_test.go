package web_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"Astramail/internal/web"
)

// testAssets 是最小可用的前端构建产物，用于隔离真实 Vite 构建。
func testAssets() fstest.MapFS {
	return fstest.MapFS{
		"index.html": &fstest.MapFile{
			Data: []byte("<!doctype html><html><body><script src=\"/assets/app-abc123.js\"></script></body></html>"),
		},
		"assets/app-abc123.js": &fstest.MapFile{
			Data: []byte("console.log('ok')"),
		},
	}
}

func TestServeIndex(t *testing.T) {
	t.Parallel()

	handler := web.NewHandler(testAssets())

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("expected text/html content type, got %s", ct)
	}

	if cache := w.Header().Get("Cache-Control"); cache != "no-cache" {
		t.Errorf("expected no-cache for index, got %s", cache)
	}
}

func TestServeSPAFallback(t *testing.T) {
	t.Parallel()

	handler := web.NewHandler(testAssets())

	// 无扩展名的路径属于前端路由，应回退到入口 HTML。
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/some/client/route", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("expected text/html content type, got %s", ct)
	}
}

func TestServeMissingAssetReturns404(t *testing.T) {
	t.Parallel()

	handler := web.NewHandler(testAssets())

	// 带扩展名的路径是静态资源，缺失时应返回 404 而不是首页。
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/assets/missing-abc123.js", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestServePathTraversalRejected(t *testing.T) {
	t.Parallel()

	handler := web.NewHandler(testAssets())

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/../go.mod", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestServeAssetHasLongCache(t *testing.T) {
	t.Parallel()

	handler := web.NewHandler(testAssets())

	// 从入口 HTML 中找出带内容哈希的资源名，验证其缓存策略。
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	asset := findAssetPath(t, w.Body.String())
	if asset == "" {
		t.Skip("构建产物中没有可用的静态资源")
	}

	reqAsset := httptest.NewRequestWithContext(t.Context(), http.MethodGet, asset, nil)
	wAsset := httptest.NewRecorder()
	handler.ServeHTTP(wAsset, reqAsset)

	if wAsset.Code != http.StatusOK {
		t.Fatalf("expected 200 for asset %s, got %d", asset, wAsset.Code)
	}

	cache := wAsset.Header().Get("Cache-Control")
	if !strings.Contains(cache, "immutable") {
		t.Errorf("expected immutable cache header for %s, got %s", asset, cache)
	}
}

// findAssetPath 从 HTML 中提取第一个以 /assets/ 开头的资源路径。
func findAssetPath(t *testing.T, html string) string {
	t.Helper()

	const marker = "/assets/"

	idx := strings.Index(html, marker)
	if idx < 0 {
		return ""
	}

	rest := html[idx:]
	end := strings.IndexAny(rest, `"'`)
	if end < 0 {
		return ""
	}

	return rest[:end]
}
