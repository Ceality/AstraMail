# AstraMail — 自动化多邮箱管理与聚合系统

AstraMail 是一套多邮箱聚合管理系统：后端用 **Go（chi + go-imap + go-message）** 实现，
前端用 **React 19 + TypeScript + TailwindCSS v4 + Base UI + Vite** 构建。

前端构建产物输出到 `web/dist`，由根目录 `main.go` 的 `//go:embed`
直接打包进服务端二进制，**最终只需要发布一个可执行文件**，无需单独部署前端服务。

---

## 目录

- [核心特性](#核心特性)
- [目录结构](#目录结构)
- [快速开始](#快速开始)
- [构建与发布](#构建与发布)
- [接口说明](#接口说明)
- [账号导入格式](#账号导入格式)
- [测试](#测试)

---

## 核心特性

### 1. 灵活的账号导入

- 统一格式：`账号----密码----clientid----刷新令牌`
- 支持多行文本批量粘贴，导入前有实时解析预览，逐行显示有效性。
- 字段容错：任意字段为空都能正确解析并自动推断协议模式。

### 2. 双协议支持（IMAP 与 Microsoft Graph）

| 模式 | 需要的凭据 | 说明 |
| --- | --- | --- |
| `imap` | 账号 + 密码（应用专用密码） | 自动根据邮箱域名推断 IMAP 服务器地址 |
| `graph` | 账号 + Client ID + 刷新令牌 | 自动用刷新令牌换取 Access Token |
| `both` | 以上全部 | 双通道并发连通性测试，可自由切换 |

IMAP 服务器地址按域名自动推断，无需手工填写：

| 邮箱域名 | IMAP 服务器 |
| --- | --- |
| `outlook.com` / `hotmail.com` / `live.com` / `office365.com` | `outlook.office365.com:993` |
| `gmail.com` | `imap.gmail.com:993` |
| `qq.com` / `foxmail.com` | `imap.qq.com:993` |
| `163.com` | `imap.163.com:993` |
| `126.com` | `imap.126.com:993` |
| `sina.com` | `imap.sina.com:993` |
| `yahoo.com` | `imap.mail.yahoo.com:993` |
| 其它域名 | 按 `imap.<域名>` 推断，例如 `imap.example.com` |
| 无域名 | 使用默认地址 `outlook.office365.com:993` |

> 也可在导入时用 `imapHost` / `imapPort` 字段显式指定，覆盖自动推断结果。

### 3. 浏览器本地存储，后端保持无状态

- 邮箱账号与私密凭据（密码、刷新令牌）全部保存在浏览器 **LocalStorage**，随请求头传给后端。
- 后端**不落库、不建会话、不持久化任何用户凭据**，随时重启不丢数据。
- 支持一键导出为标准格式文本备份。

### 4. 多账号聚合取件

- 「**选中取件**」只处理已勾选的账号；「**全部取件**」处理所有账号。两者互不混淆。
- 支持 IMAP 与 Graph 双协议并发取件，结果自动去重。
- 支持关键字搜索、发件人过滤、按接收时间全局倒序排列。
- 可设置取件数量上限、是否包含垃圾箱。

### 5. 邮件阅读与管理

- 动态获取邮箱文件夹目录（收件箱、垃圾箱、已发送等）与邮件总数。
- 邮件列表展示主题、发件人、所属账号、协议来源与正文摘要。
- 邮件详情抽屉：HTML 富文本 / 纯文本双视图切换、附件列表、协议来源标识。
- 响应式布局，桌面端与移动端各自适配（移动端侧边栏收进抽屉）。

---

## 目录结构

遵循 Go 官方标准项目布局（Standard Go Project Layout）与 React/TypeScript 官方推荐分层：

```text
mail/
├── go.mod                            # Go 模块（仓库根目录）
├── go.sum
├── .env                              # 本地配置（SERVER_HOST / SERVER_PORT）
├── .env.example                      # 环境变量模板
├── main.go                           # 服务入口、//go:embed 前端产物、优雅停机
├── Makefile                          # 构建编排：build / test / release / clean
├── Dockerfile                        # 基于 GitHub Release 产物的最小运行镜像
├── docker-compose.yml                # 一键部署配置
├── README.md
├── .github/workflows/release.yml     # 推送 v* 标签时自动构建并发布 Release
│
├── internal/                         # 核心领域与协议封装（Go 编译器私有保护）
│   ├── config/                       # 服务配置与环境变量加载
│   ├── apperror/                     # 统一错误体系（错误包装与状态码映射）
│   ├── model/                        # 领域模型（账号凭据、邮件实体、文件夹等）
│   ├── version/                      # 版本信息（Version 常量硬编码，唯一可信来源）
│   ├── service/                      # 业务逻辑与协议适配层
│   │   ├── imap/                     # IMAP TLS 连接、RFC822 解析、邮件操作
│   │   └── graph/                    # Microsoft Graph OAuth2 令牌刷新、REST 邮件操作
│   ├── handler/                      # HTTP 控制器与统一 JSON 响应封装
│   │   ├── router.go                 # RESTful 路由树与 CORS 中间件
│   │   ├── auth.go                   # Authorization 凭据头解析（无状态）
│   │   ├── request.go                # query 参数解析与校验
│   │   ├── response.go               # 统一错误体与集合空值处理
│   │   ├── account_handler.go        # 账号资源接口
│   │   └── mail_handler.go           # 文件夹、邮件与聚合取件接口
│   └── web/                          # 内嵌资源静态服务（SPA 回退、缓存策略）
│       ├── handler.go
│       └── handler_test.go
│
└── web/                              # 前端应用 (Vite + React 19 + TailwindCSS v4 + Base UI)
    ├── package.json
    ├── vite.config.ts                # 开发代理 (/api -> 127.0.0.1:3001)；产物输出到 web/dist
    ├── dist/                         # Vite 构建产物（不入库，仅保留 .gitkeep）
    ├── index.html
    ├── public/favicon.svg
    ├── tsconfig.app.json
    ├── tsconfig.test.json
    └── src/
        ├── main.tsx                  # React 挂载入口
        ├── App.tsx                   # 整体布局、全局账号状态
        ├── index.css                 # Tailwind 主题变量与全局样式
        ├── types/                    # TypeScript 类型定义
        ├── services/                 # 客户端服务层 (api.ts, storage.ts)
        ├── utils/                    # 纯函数工具 (selection.ts, verification.ts, clipboard.ts)
        ├── hooks/                    # 自定义 Hook (useIsMobile.ts, useAutoFetch.ts)
        └── components/               # 自研组件（Base UI 原语 + Framer Motion 动画）
```

---

## 快速开始

### 环境要求

| 依赖 | 版本 |
| --- | --- |
| Go | 1.26+（`go.mod` 声明 `go 1.26.5`） |
| Node.js | 22.18+ 或 24+（`npm test` 依赖 Node 原生 TypeScript 支持；开发环境实测 24.22） |
| npm / pnpm | 任选 |

### 开发模式（前后端分离）

**终端一 —— 启动后端：**

```bash
go run .
```

后端监听 `http://127.0.0.1:3001`。健康检查：`GET http://127.0.0.1:3001/api/health`

**终端二 —— 启动前端：**

```bash
cd web
npm install
npm run dev
```

前端监听 `http://localhost:5273`，通过 Vite Proxy 把 `/api` 请求转发到后端。

### 生产模式（单一可执行文件）

```bash
make build
```

产物在 `bin/server`（Windows 为 `bin/server.exe`）。启动后直接访问 `http://127.0.0.1:3001/` 即可打开完整界面，
前端资源由二进制内嵌提供，不需要额外的静态服务器。

### Docker 部署（直接使用 GitHub Release 产物）

Docker 镜像**不在容器里编译源码**，而是在构建阶段从 GitHub Release 下载
`astramail-<版本>-linux-amd64` 与 `checksums.txt`，校验 SHA-256 后放入镜像。
最终运行镜像基于 `scratch`，只包含静态二进制和 CA 证书，没有 shell、包管理器或多余依赖。

先确认对应版本的 Release 已发布（例如 `v0.1.0`），然后：

```bash
# 构建镜像（默认版本 0.1.0）
docker build -t astramail:0.1.0 .

# 启动
docker run -d \
  --name astramail \
  --restart unless-stopped \
  -p 3001:3001 \
  astramail:0.1.0
```

或者使用 Compose：

```bash
docker compose up -d --build
```

构建其它版本时覆盖构建参数：

```bash
docker build \
  --build-arg VERSION=0.1.0 \
  --build-arg REPO=Ceality/AstraMail \
  -t astramail:0.1.0 .
```

| 项目 | 说明 |
| --- | --- |
| Release 资产 | `astramail-<版本>-linux-amd64` + `checksums.txt` |
| 运行用户 | `65532:65532`（非 root） |
| 监听地址 | 容器内 `0.0.0.0:3001` |
| 可写层 | 不需要；Compose 已启用 `read_only` |
| CA 证书 | 来自 Alpine 的 `ca-certificates`，用于 IMAP / Graph TLS |

> 镜像构建依赖外网访问 GitHub Release。若网络受限，可先手动下载两个资产，
> 或用 `docker build --build-arg REPO=<你的镜像仓库>` 指向可访问的发布源。

### 环境变量

后端从仓库根目录的 `.env` 或系统环境变量读取配置：

```ini
SERVER_HOST=127.0.0.1
SERVER_PORT=3001
```

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `SERVER_HOST` | `127.0.0.1` | 监听地址 |
| `SERVER_PORT` | `3000` | 监听端口 |

---

## 构建与发布

### 发布构建参数

发布构建会自动加上两个参数，显著压缩二进制体积：

| 参数 | 作用 | 节省 |
| --- | --- | --- |
| `-ldflags="-w -s"` | 去掉 DWARF 调试信息与符号表 | ~9.4 MiB |
| `-trimpath` | 去掉构建机绝对路径，便于复现构建 | — |

```bash
make build          # 发布构建
make backend-dev    # 带调试符号，便于本地排查
```

### 体积构成

前端产物（gzip 后约 177 KiB）并不是体积主因：

| 组成 | 大小 | 是否可优化 |
| --- | --- | --- |
| DWARF 调试信息 + 符号表 | ~9.4 MiB | ✅ `-w -s` 去掉 |
| 前端构建产物 | ~1.3 MiB | ❌ 内嵌必需 |
| Go 运行时 + 依赖库 | ~8.5 MiB | ❌ `runtime.pclntab`、类型元数据等硬成本 |

> 从 Gin 迁移到 chi 后，二进制已不再包含 Gin 及其 msgpack 绑定器，
> 因此不需要 `nomsgpack` build tag。

### 版本管理

版本号**硬编码在 Go 源码中**，是唯一可信来源：

```go
// internal/version/version.go
const (
	Version = "0.1.0"
)
```

发布新版本只需改这一行。构建产物名、`/api/health` 接口与启动日志都会自动反映新版本。

| 字段 | 来源 | 说明 |
| --- | --- | --- |
| `version` | 源码常量 | 硬编码，唯一需要手工维护 |
| `commit` | 构建期注入 | `-ldflags -X`，自动探测 Git；无 Git 时为 `nogit` |
| `buildDate` | 构建期注入 | `-ldflags -X`，构建时的 UTC 时间 |

> `commit` 与 `buildDate` 必须是 `var` 而非 `const`：链接器的 `-X` 只能覆盖字符串变量，
> 对常量注入会被**静默忽略**。因此 `version` 不用链接器注入，改由发布工具从源码解析。

---

## 接口说明

所有接口位于 `/api` 下，资源化路径 + 标准 HTTP 动词，字段采用驼峰命名。

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/health` | 健康检查（返回版本、提交号、构建时间） |
| GET | `/api/account?checkProtocol=both` | 账号资源 + 各协议连通状态 |
| GET | `/api/folders?protocol=imap` | 文件夹集合 |
| GET | `/api/messages?folder=INBOX&page=1&limit=20&q=关键字&protocol=both` | 邮件集合（分页 / 搜索） |
| GET | `/api/messages/{id}?folder=INBOX` | 邮件详情 |
| POST | `/api/messageQueries` | 跨账号聚合取件 |

**查询参数**

| 参数 | 适用接口 | 说明 |
| --- | --- | --- |
| `protocol` | folders / messages | `imap` / `graph` / `both`，缺省用账号自身协议 |
| `checkProtocol` | account | 连通性测试用的协议，覆盖账号自身协议 |
| `folder` | messages | 文件夹名，如 `INBOX` |
| `page` | messages | 页码，从 1 开始 |
| `limit` | messages | 每页数量 |
| `q` | messages | 关键字搜索 |

**认证**：后端保持无状态，凭据通过请求头传递，不落库、不建会话：

```http
Authorization: AstraMail <base64(JSON 账号凭据)>
```

凭据 JSON 字段：`email`、`password`、`clientId`、`refreshToken`、`protocol`、`imapHost`、`imapPort`。
（`POST /api/messageQueries` 是跨账号查询，凭据随请求体的 `accounts` 数组携带。）

**响应**：成功直接返回资源 —— `GET`/`POST` 返回 JSON 资源，`DELETE` 返回 `204 No Content`；
失败返回标准 HTTP 状态码 + 统一错误体：

```json
{
  "code": "IMAP_ERROR",
  "message": "IMAP Error: Connection to imap.example.com:993 failed"
}
```

| 错误码 | HTTP 状态码 |
| --- | --- |
| `BAD_REQUEST` | 400 |
| `AUTH_FAILED` | 401 |
| `NOT_FOUND` | 404 |
| `METHOD_NOT_ALLOWED` | 405 |
| `PARSE_ERROR` | 422 |
| `INTERNAL_ERROR` | 500 |
| `IMAP_ERROR` / `GRAPH_ERROR` | 502 |

> 文件夹用 query 参数而不是 `/folders/{folder}/messages` 路径段，因为 IMAP 文件夹名本身可能包含 `/`
> （如 `INBOX/Work`），路径段化会在百分号编码解码头处匹配失败。

### 内嵌模式下的路由规则

内嵌前端后，后端同时承担 API 与前端页面的分发：

| 请求路径 | 处理方式 |
| --- | --- |
| `/api/*` | 走 API 路由；未命中时返回统一 JSON 404 错误体 |
| `/assets/*` | 返回带内容哈希的静态资源，`Cache-Control: max-age=31536000, immutable` |
| 其它路径（无扩展名） | 单页应用回退，返回 `index.html` |
| 其它路径（含扩展名） | 返回 404（缺失的静态资源不会误报为首页） |

入口 `index.html` 使用 `Cache-Control: no-cache`，确保前端发布后用户能立即拿到新的入口文件。

---

## 账号导入格式

在「账号管理」中点击「导入」，粘贴多行数据（每行一个账号）：

```text
user1@outlook.com----Password123----d1234567-89ab-cdef----0.ARAA_token_xxx
user2@gmail.com----AppPassword456--------
user3@hotmail.com--------client_id_here----refresh_token_here
```

字段用四个连续短横线 `----` 分隔，顺序固定为：

```text
账号----密码----Client ID----刷新令牌
```

| 场景 | 填写方式 | 自动推断的模式 |
| --- | --- | --- |
| 全量凭据 | 四项全填 | `both`（IMAP + Graph 双通道） |
| 纯 IMAP | 只填账号 + 密码 | `imap` |
| 纯 Graph | 填账号 + Client ID + 刷新令牌（密码留空） | `graph` |

> 未勾选任何账号时，「选中取件」会直接提示并中止，不会退化成「全部取件」；
> 「导出」同样如此，避免误操作导出全部凭据。

---

## 测试

```bash
# 后端 + 前端一起跑
make test

# 分别执行
go test ./...
cd web && npm test
```

后端使用 Go 内置测试框架，覆盖错误体系、领域模型、Graph 服务、路由与前端内嵌服务。
前端使用 **Node 内置 test runner**（无需额外依赖），覆盖账号选择逻辑等纯函数。

```bash
# 前端类型检查与构建
cd web
npx tsc -b --force
npx vite build
```
