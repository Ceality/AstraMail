// Package handler provides HTTP request routing, middleware, and endpoint handlers.
package handler

import (
	"io/fs"
	"net/http"
	"runtime"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"Astramail/internal/config"
	"Astramail/internal/service/graph"
	"Astramail/internal/service/imap"
	"Astramail/internal/version"
	"Astramail/internal/web"
)

// NewRouter 组装 API 路由，并把未命中的非 API 请求交给内嵌前端。
//
// assets 是 Vite 构建产物（以 dist 为根）的只读文件系统，
// 由调用方负责 //go:embed，路由层不感知资源来自磁盘还是二进制。
func NewRouter(_ *config.Config, assets fs.FS) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(corsOptions))

	// Dependency injection
	imapSvc := imap.NewService()
	graphSvc := graph.NewService()

	accountHandler := NewAccountHandler(imapSvc, graphSvc)
	mailHandler := NewMailHandler(imapSvc, graphSvc)

	r.Route("/api", func(api chi.Router) {
		// 无需凭据：健康检查，以及凭据随请求体携带的跨账号查询
		api.Get("/health", health)
		api.Post("/messageQueries", mailHandler.QueryMessages)

		// 带凭据路由：Authorization 头解析为账号上下文
		api.Group(func(r chi.Router) {
			r.Use(RequireCredentials)
			r.Get("/account", accountHandler.CheckAccount)
			r.Get("/folders", mailHandler.ListFolders)
			r.Get("/messages", mailHandler.ListMessages)
			r.Get("/messages/{id}", mailHandler.FetchMessageDetail)
		})

		// /api 前缀内未匹配的路由返回统一 JSON 404
		api.NotFound(NotFound)
		api.MethodNotAllowed(MethodNotAllowed)
	})

	// 前端构建产物已内嵌进二进制，静态资源与单页应用回退统一交给 web 包处理。
	r.NotFound(spaHandler(assets))

	return r
}

// spaHandler 返回处理未命中路由的处理器：API 路径返回统一 JSON 404，其余路径回退到前端页面。
func spaHandler(assets fs.FS) http.HandlerFunc {
	static := web.NewHandler(assets)

	return func(w http.ResponseWriter, r *http.Request) {
		if isAPIPath(r.URL.Path) {
			NotFound(w, r)

			return
		}

		static.ServeHTTP(w, r)
	}
}

// isAPIPath 判断请求路径是否属于 API 前缀。
func isAPIPath(urlPath string) bool {
	return urlPath == "/api" || strings.HasPrefix(urlPath, "/api/")
}

type HealthResponse struct {
	Status    string `json:"status"`
	Goroutine int    `json:"goroutine"`
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"buildDate"`
}

// health returns the service health status.
func health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, HealthResponse{
		Status:    "ok",
		Goroutine: runtime.NumGoroutine(),
		Version:   version.Version,
		Commit:    version.Commit,
		BuildDate: version.BuildDate,
	})
}

// corsOptions allows the Vite dev server to call the API, including the credential header.
var corsOptions = cors.Options{
	AllowedOrigins: []string{"*"},
	AllowedMethods: []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodOptions,
	},
	AllowedHeaders: []string{"Content-Type", "Authorization", "Accept"},
}
