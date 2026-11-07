package handler_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"Astramail/internal/config"
	"Astramail/internal/handler"
	"Astramail/internal/model"
)

// testAssets 提供最小的前端产物，避免路由测试依赖真实 Vite 构建。
var testAssets = fstest.MapFS{
	"index.html": &fstest.MapFile{
		Data: []byte("<!doctype html><html><body><div id=\"root\"></div></body></html>"),
	},
}

func makeTestAuthHeader() string {
	raw := []byte(`{"email":"test@example.com","password":"secret","protocol":"imap"}`)

	return "AstraMail " + base64.StdEncoding.EncodeToString(raw)
}

func TestHealthEndpoint(t *testing.T) {
	t.Parallel()

	r := handler.NewRouter(&config.Config{}, testAssets)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var res struct {
		Status    string `json:"status"`
		Goroutine int    `json:"goroutine"`
		Version   string `json:"version"`
	}

	err := json.Unmarshal(w.Body.Bytes(), &res)
	if err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if res.Status != "ok" {
		t.Errorf("expected status ok, got %s", res.Status)
	}
}

func TestNotFoundAndMethodNotAllowed(t *testing.T) {
	t.Parallel()

	r := handler.NewRouter(&config.Config{}, testAssets)

	// 404 test
	req404 := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/nonexistent", nil)
	w404 := httptest.NewRecorder()
	r.ServeHTTP(w404, req404)

	if w404.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w404.Code)
	}

	// 405 test
	req405 := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/health", nil)
	w405 := httptest.NewRecorder()
	r.ServeHTTP(w405, req405)

	if w405.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w405.Code)
	}
}

func TestMessageQueriesValidation(t *testing.T) {
	t.Parallel()

	r := handler.NewRouter(&config.Config{}, testAssets)

	// Bad JSON payload
	reqBad := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/api/messageQueries",
		bytes.NewBufferString("{invalid"),
	)
	reqBad.Header.Set("Content-Type", "application/json")

	wBad := httptest.NewRecorder()
	r.ServeHTTP(wBad, reqBad)

	if wBad.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for malformed json, got %d", wBad.Code)
	}

	// Valid empty payload
	payload, err := json.Marshal(model.MessageQueryRequest{Accounts: []model.AccountCredentials{}})
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	reqValid := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/api/messageQueries",
		bytes.NewReader(payload),
	)
	reqValid.Header.Set("Content-Type", "application/json")

	wValid := httptest.NewRecorder()
	r.ServeHTTP(wValid, reqValid)

	if wValid.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", wValid.Code)
	}

	var res model.MessageQueryResponse

	err = json.Unmarshal(wValid.Body.Bytes(), &res)
	if err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if res.Total != 0 || len(res.Items) != 0 {
		t.Errorf("expected 0 items, got total %d, len %d", res.Total, len(res.Items))
	}
}

func TestMessageQueriesProtocolMismatchReports(t *testing.T) {
	t.Parallel()

	r := handler.NewRouter(&config.Config{}, testAssets)

	// 只有 IMAP 凭据的账号，但请求 both 协议：
	// Graph 凭据缺失应明确报错，而不是静默跳过
	both := "both"
	payload, err := json.Marshal(model.MessageQueryRequest{
		Accounts: []model.AccountCredentials{
			{
				Email:    "imap-only@example.com",
				Password: "secret",
				Protocol: "imap",
			},
		},
		Protocol: &both,
	})
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/api/messageQueries",
		bytes.NewReader(payload),
	)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var res model.MessageQueryResponse
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	found := false
	for _, msg := range res.Errors {
		if strings.Contains(msg, "Graph credentials missing") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected Graph credentials missing error, got errors: %v", res.Errors)
	}
}

func TestQueryParamValidation(t *testing.T) {
	t.Parallel()

	r := handler.NewRouter(&config.Config{}, testAssets)
	auth := makeTestAuthHeader()

	// Invalid page
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/messages?page=-5", nil)
	req.Header.Set("Authorization", auth)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for negative page, got %d", w.Code)
	}

	// DELETE on /api/messages is not supported (405 Method Not Allowed)
	reqDel := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/api/messages", nil)
	reqDel.Header.Set("Authorization", auth)

	wDel := httptest.NewRecorder()
	r.ServeHTTP(wDel, reqDel)

	if wDel.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for DELETE /api/messages, got %d", wDel.Code)
	}
}

func TestAPIPathReturnsJSONNotFound(t *testing.T) {
	t.Parallel()

	r := handler.NewRouter(&config.Config{}, testAssets)

	// 未命中的 API 路径必须保持统一的 JSON 错误体，不能回退到前端页面。
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}

	if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("expected JSON content type, got %s", ct)
	}
}

func TestNonAPIPathServesSPA(t *testing.T) {
	t.Parallel()

	r := handler.NewRouter(&config.Config{}, testAssets)

	// 非 API 路径应回退到内嵌的前端页面。
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/dashboard", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("expected text/html content type, got %s", ct)
	}
}
