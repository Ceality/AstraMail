# AstraMail 构建编排
#
# Go 后端位于仓库根目录；前端在 web/，Vite 直接输出到 web/dist，
# 再由根目录 main.go 的 //go:embed 打包进服务端二进制，最终只需要发布一个可执行文件。

WEB_DIR    := web
EMBED_DIR  := web/dist
SERVER_BIN := bin/server
DIST_DIR   := dist

COMMIT := $(shell git rev-parse --short HEAD 2>NUL || echo nogit)

# 版本号从 internal/version/version.go 解析，保持与 /api/health 一致。
# 构建时间按平台选择可用的命令，避免 Windows 上依赖 Unix 工具链。
ifeq ($(OS),Windows_NT)
VERSION := $(shell powershell -NoProfile -Command "(Select-String -Path 'internal/version/version.go' -Pattern 'Version = \"(.*)\"').Matches.Groups[1].Value")
BUILD_DATE := $(shell powershell -NoProfile -Command "[DateTime]::UtcNow.ToString('yyyy-MM-ddTHH:mm:ssZ')")
RUN_BIN := bin\server.exe
RELEASE_CMD := powershell -NoProfile -ExecutionPolicy Bypass -File scripts/release.ps1
else
VERSION := $(shell sed -n 's/^[[:space:]]*Version = "\(.*\)"/\1/p' internal/version/version.go)
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
RUN_BIN := ./$(SERVER_BIN)
RELEASE_CMD := bash scripts/release.sh
endif

# 发布构建参数：
#   -w -s        去掉 DWARF 调试信息与符号表（约 -9.4 MiB，部署不需要）
#   -trimpath    去掉构建机的绝对路径，便于复现构建（go build 参数，不属于 ldflags）
GO_LDFLAGS   := -w -s -X Astramail/internal/version.Commit=$(COMMIT) -X Astramail/internal/version.BuildDate=$(BUILD_DATE)

.PHONY: all build frontend backend backend-dev run dev clean test release

# all: 构建前端并产出内嵌前端资源的单一可执行文件
all: build

# frontend: 构建前端，产物输出到 web/dist（根目录 main.go 的内嵌目录）
frontend:
	cd $(WEB_DIR) && npm run build

# backend: 编译后端二进制（依赖前端产物已生成）
backend: frontend
	go build -trimpath -ldflags="$(GO_LDFLAGS)" -o $(SERVER_BIN) .

# backend-dev: 编译带调试信息的二进制，便于本地排查问题
backend-dev: frontend
	go build -o bin/server-debug .

# build: 等价于 all
build: backend

# run: 构建并启动服务
run: build
	$(RUN_BIN)

# dev: 启动后端开发服务（不内嵌前端产物，前端另用 vite dev）
dev:
	go run .

# test: 运行前后端全部测试
test:
	go test ./...
	cd $(WEB_DIR) && npm test

# release: 交叉编译发布产物并生成 checksums.txt（产物在 dist/）
release: frontend
	$(RELEASE_CMD)

# clean: 清理构建产物
clean:
ifeq ($(OS),Windows_NT)
	-rmdir /s /q "$(subst /,\,$(EMBED_DIR))" 2>nul
	-rmdir /s /q "bin" 2>nul
	-rmdir /s /q "$(DIST_DIR)" 2>nul
	-mkdir "$(subst /,\,$(EMBED_DIR))"
	-echo. > "$(subst /,\,$(EMBED_DIR))\.gitkeep"
else
	rm -rf $(EMBED_DIR) bin $(DIST_DIR)
	mkdir -p $(EMBED_DIR)
	touch $(EMBED_DIR)/.gitkeep
endif
