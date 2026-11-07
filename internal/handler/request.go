package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"Astramail/internal/apperror"
	"Astramail/internal/model"
)

// 查询参数与路径参数名。
const (
	queryFolder        = "folder"
	queryPage          = "page"
	queryLimit         = "limit"
	querySearch        = "q"
	queryProtocol      = "protocol"
	queryCheckProtocol = "checkProtocol"
	paramMailID        = "id"
)

// 请求默认值。
const (
	// folderInbox 是未指定 folder 时使用的默认收件箱。
	folderInbox = "INBOX"
	// folderJunk 是 Graph 通道的垃圾箱标识（API 层映射到 junkemail）。
	folderJunk = "Junk"
	// defaultPageSize 是未指定 limit 时每页返回的邮件数。
	defaultPageSize = 20
)

// queryInt 解析正整数查询参数；缺省时返回 fallback，非法值返回 400 错误。
func queryInt(r *http.Request, key string, fallback int) (int, error) {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return fallback, nil
	}

	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 0, apperror.NewBadRequest(
			fmt.Sprintf("Query parameter %s must be a positive integer", key), nil)
	}

	return value, nil
}

// queryText 返回去除首尾空白后的查询参数，空串视为未提供。
func queryText(r *http.Request, key string) *string {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return nil
	}

	return &raw
}

// folderParam 返回请求指定的文件夹，缺省为收件箱。
func folderParam(r *http.Request) string {
	folder := strings.TrimSpace(r.URL.Query().Get(queryFolder))
	if folder == "" {
		return folderInbox
	}

	return folder
}

// pathParam 返回 URL 路径参数。
func pathParam(r *http.Request, key string) string {
	return chi.URLParam(r, key)
}

// resolveProtocol 解析本次请求使用的协议：query 参数优先于账号自身配置。
func resolveProtocol(creds model.AccountCredentials, override *string) string {
	if override != nil {
		return *override
	}

	return string(creds.Protocol)
}

// useGraph 判断本次请求是否走 Microsoft Graph 通道。
func useGraph(proto string, creds model.AccountCredentials) bool {
	return strings.EqualFold(proto, "graph") && creds.HasGraphCreds()
}

// bindJSON 解析请求体并填充 req，失败时写出 400 响应并返回 false。
func bindJSON(w http.ResponseWriter, r *http.Request, req any) bool {
	err := json.NewDecoder(r.Body).Decode(req)
	if err != nil {
		HandleError(w, apperror.NewBadRequest("Invalid request payload: "+err.Error(), err))

		return false
	}

	return true
}
