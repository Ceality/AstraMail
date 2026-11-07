package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"Astramail/internal/apperror"
)

// errorBody 是统一错误响应体：HTTP 状态码表达结果，code 供程序判断，message 供人阅读。
type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// 路由级错误码：未匹配到资源或方法。
const (
	errorCodeNotFound         = "NOT_FOUND"
	errorCodeMethodNotAllowed = "METHOD_NOT_ALLOWED"
)

// writeJSON 写出 JSON 响应；序列化失败时降级为 500 纯文本。
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(v); err != nil {
		// 头已写出，只能尽力补一条日志级错误
		http.Error(w, `{"code":"INTERNAL_ERROR","message":"failed to encode response"}`, http.StatusInternalServerError)
	}
}

// NotFound 写出未匹配路由的统一 404 响应。
func NotFound(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusNotFound, errorBody{
		Code:    errorCodeNotFound,
		Message: "No resource at " + r.Method + " " + r.URL.Path,
	})
}

// MethodNotAllowed 写出方法不被支持的统一 405 响应。
func MethodNotAllowed(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusMethodNotAllowed, errorBody{
		Code:    errorCodeMethodNotAllowed,
		Message: r.Method + " is not allowed on " + r.URL.Path,
	})
}

// HandleError 按 API 契约写出错误响应。
// 调用方负责在其后立即 return，避免继续写出第二份响应。
func HandleError(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}

	appErr, ok := errors.AsType[*apperror.AppError](err)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, errorBody{
			Code:    string(apperror.CodeInternal),
			Message: err.Error(),
		})
		return
	}

	writeJSON(w, appErr.StatusCode, errorBody{
		Code:    string(appErr.Code),
		Message: appErr.Message,
	})
}

// nonNil 保证集合类资源序列化为 [] 而不是 null。
func nonNil[T any](items []T) []T {
	if items == nil {
		return []T{}
	}

	return items
}
