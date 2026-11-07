package web

import (
	"bytes"
	"io/fs"
	"log"
	"net/http"
	"path"
	"strings"
	"time"
)

const (
	// indexFileName 是单页应用的入口 HTML 文件名。
	indexFileName = "index.html"
	// assetsPrefix 是 Vite 产物中存放带内容哈希静态资源的目录前缀。
	assetsPrefix = "assets/"
	// cacheControlImmutable 是带内容哈希资源的缓存策略：一年且不可重新校验。
	cacheControlImmutable = "public, max-age=31536000, immutable"
	// cacheControlDefault 是普通静态资源的缓存策略：一小时。
	cacheControlDefault = "public, max-age=3600"
	// contentTypeHTML 是 HTML 响应的 Content-Type。
	contentTypeHTML = "text/html; charset=utf-8"
)

// fallbackHTML 是前端尚未构建时返回的提示页面。
const fallbackHTML = `<!doctype html>
<html lang="zh-CN">
  <head>
    <meta charset="UTF-8" />
    <title>AstraMail</title>
  </head>
  <body>
    <h1>AstraMail</h1>
    <p>前端资源尚未构建，请先在 web 目录执行 <code>npm run build</code>。</p>
  </body>
</html>
`

// Handler 提供内嵌前端资源的静态服务，并实现单页应用的历史路由回退。
type Handler struct {
	// assets 是以构建产物根目录为根的只读文件系统。
	assets fs.FS
	// index 是入口 HTML 的内容；前端未构建时为 nil。
	index []byte
}

// NewHandler 基于给定的构建产物文件系统创建静态资源处理器。
//
// assets 必须已经以 Vite 的 dist 目录为根。嵌入动作由 cmd/server 负责，
// 这里只消费文件系统，便于测试时注入内存文件系统。
//
// 前端尚未构建时（dist 目录只有占位文件）不会返回错误，
// 此时访问首页会得到一段提示 HTML，其余资源返回 404。
func NewHandler(assets fs.FS) *Handler {
	return &Handler{
		assets: assets,
		index:  readIndex(assets),
	}
}

// ServeHTTP 按以下顺序响应请求：
//
//  1. 命中构建产物时直接返回该文件，并为带哈希的资源设置长期缓存。
//  2. 未命中且路径不含扩展名时按单页应用规则回退到入口 HTML。
//  3. 未命中且路径含扩展名时返回 404，避免把缺失的静态资源误报为首页。
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	name, ok := resolvePath(r.URL.Path)
	if !ok {
		http.NotFound(w, r)

		return
	}

	if name == indexFileName {
		h.serveIndex(w, r)

		return
	}

	content, err := fs.ReadFile(h.assets, name)
	if err != nil {
		h.serveFallback(w, r, name)

		return
	}

	setStaticHeaders(w, name)

	// http.ServeContent 会依据文件扩展名自动推导 Content-Type，并处理 HEAD 与 Range 请求。
	http.ServeContent(w, r, path.Base(name), time.Time{}, bytes.NewReader(content))
}

// serveIndex 返回单页应用入口 HTML。
func (h *Handler) serveIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", contentTypeHTML)
	// 入口 HTML 必须每次校验，否则前端发布后用户会继续命中旧的入口文件。
	w.Header().Set("Cache-Control", "no-cache")

	if h.index == nil {
		if _, err := w.Write([]byte(fallbackHTML)); err != nil {
			log.Printf("web: 写入提示页面失败: %v", err)
		}

		return
	}

	http.ServeContent(w, r, indexFileName, time.Time{}, bytes.NewReader(h.index))
}

// serveFallback 处理未命中资源的回退：带扩展名视为缺失的静态资源，其余按前端路由处理。
func (h *Handler) serveFallback(w http.ResponseWriter, r *http.Request, name string) {
	if path.Ext(name) != "" {
		http.NotFound(w, r)

		return
	}

	h.serveIndex(w, r)
}

// resolvePath 把 URL 路径转换为嵌入文件系统中的资源名。
//
// 返回 false 表示路径非法（例如包含越界的 ".."），此时应当返回 404。
func resolvePath(urlPath string) (string, bool) {
	cleaned := path.Clean("/" + urlPath)
	name := strings.TrimPrefix(cleaned, "/")
	if name == "" {
		return indexFileName, true
	}

	if !fs.ValidPath(name) {
		return "", false
	}

	return name, true
}

// readIndex 读取入口 HTML；前端未构建时返回 nil。
func readIndex(assets fs.FS) []byte {
	raw, err := fs.ReadFile(assets, indexFileName)
	if err != nil {
		return nil
	}

	return raw
}

// setStaticHeaders 为静态资源设置缓存策略：带内容哈希的资源长缓存，其余短缓存。
func setStaticHeaders(w http.ResponseWriter, name string) {
	if strings.HasPrefix(name, assetsPrefix) {
		w.Header().Set("Cache-Control", cacheControlImmutable)

		return
	}

	w.Header().Set("Cache-Control", cacheControlDefault)
}
