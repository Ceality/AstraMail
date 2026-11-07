// Package main is the entry point for the AstraMail Core server application.
package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"Astramail/internal/config"
	"Astramail/internal/handler"
	"Astramail/internal/version"
)

const (
	// serverReadTimeout 限制读取整个请求（含请求体）的最长时间。
	serverReadTimeout = 60 * time.Second
	// serverWriteTimeout 限制写出响应的最长时间。
	serverWriteTimeout = 60 * time.Second
	// serverIdleTimeout 限制 keep-alive 空闲连接的保持时间。
	serverIdleTimeout = 120 * time.Second
	// shutdownTimeout 是优雅退出时等待存量请求结束的最长时间。
	shutdownTimeout = 10 * time.Second
)

// webDist 内嵌 Vite 构建产物。
//
// all: 前缀保证以 "." 或 "_" 开头的文件（例如占位文件 .gitkeep）也会被嵌入，
// 从而让 dist 尚未生成真实产物时 go build 依然可以通过。
//
//go:embed all:web/dist
var webDist embed.FS

func main() {
	cfg := config.Load()

	assets, err := fs.Sub(webDist, "web/dist")
	if err != nil {
		log.Fatalf("内嵌前端资源挂载失败: %v", err)
	}

	r := handler.NewRouter(cfg, assets)

	addr := fmt.Sprintf("%s:%d", cfg.ServerHost, cfg.ServerPort)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  serverReadTimeout,
		WriteTimeout: serverWriteTimeout,
		IdleTimeout:  serverIdleTimeout,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("AstraMail %s", version.Long())
		log.Printf("Listening on http://%s", addr)

		err := srv.ListenAndServe()

		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down gracefully, press Ctrl+C again to force")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	err = srv.Shutdown(shutdownCtx)
	if err != nil {
		log.Printf("Server forced shutdown: %v", err)
	}

	log.Println("Server exiting")

}
