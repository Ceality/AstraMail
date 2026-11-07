// Package version 暴露 AstraMail 的版本信息。
//
// 版本号直接硬编码在本文件中，作为唯一可信来源（single source of truth）：
// 发布新版本时只需修改下面的 Version 常量，无需外部 VERSION 文件或构建参数。
package version

const (
	// Version 是语义化版本号，例如 "0.1.0"。
	//
	// 发布新版本时修改此常量即可，构建产物与 /api/health 会自动反映新版本。
	//
	// 这是唯一需要手工维护的版本信息，不参与构建期注入。
	Version = "0.1.0"
)

// 以下变量由构建时 -ldflags -X 覆盖。
//
// 必须是变量而非常量：链接器只能注入字符串变量，对常量注入会被静默忽略。
var (
	// Commit 是构建时的 Git 提交号；无 Git 环境时为 "unknown"。
	//
	// 由发布工具在构建时通过 -ldflags -X 注入；`go run` 与 `go test` 场景下为 "unknown"。
	Commit = "unknown"

	// BuildDate 是构建时间（RFC3339）；未注入时由调试信息推断。
	//
	// 由发布工具在构建时通过 -ldflags -X 注入；`go run` 与 `go test` 场景下为 "unknown"。
	BuildDate = "unknown"
)

// String 返回适合日志与健康检查接口展示的版本描述。
func String() string {
	return Version + "+" + Commit
}

// Long 返回包含构建时间的完整版本描述。
func Long() string {
	return Version + "+" + Commit + " (" + BuildDate + ")"
}
