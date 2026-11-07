package handler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"

	"Astramail/internal/apperror"
	"Astramail/internal/model"
)

const (
	// authScheme 是凭据头使用的认证方案：Authorization: AstraMail <base64(JSON)>。
	authScheme = "AstraMail"
)

// ctxAccountKey 是账号信息在请求上下文中的键类型。
// 用自定义类型避免与其它包的字符串键冲突。
type ctxAccountKey struct{}

// RequireCredentials 解析 Authorization 头中的账号凭据并写入请求上下文。
// 后端保持无状态：凭据只在本请求内使用，不落库、不建会话。
func RequireCredentials(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		creds, err := parseCredentials(r.Header.Get("Authorization"))
		if err != nil {
			HandleError(w, err)

			return
		}

		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxAccountKey{}, creds)))
	})
}

// Credentials 返回当前请求携带的账号凭据。
func Credentials(ctx context.Context) model.AccountCredentials {
	value, ok := ctx.Value(ctxAccountKey{}).(model.AccountCredentials)
	if !ok {
		return model.AccountCredentials{}
	}

	return value
}

// parseCredentials 解析认证头里的 base64 编码账号 JSON。
func parseCredentials(header string) (model.AccountCredentials, error) {
	scheme, encoded, found := strings.Cut(header, " ")
	if !found || !strings.EqualFold(strings.TrimSpace(scheme), authScheme) {
		return model.AccountCredentials{}, apperror.NewAuthenticationFailed(
			"Missing credentials, expected header: Authorization: "+authScheme+" <base64(json)>", nil)
	}

	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
	if err != nil {
		return model.AccountCredentials{}, apperror.NewBadRequest("Authorization header is not valid base64", nil)
	}

	var creds model.AccountCredentials

	err = json.Unmarshal(raw, &creds)
	if err != nil {
		return model.AccountCredentials{}, apperror.NewBadRequest("Authorization header is not valid account JSON", nil)
	}

	if strings.TrimSpace(creds.Email) == "" {
		return model.AccountCredentials{}, apperror.NewBadRequest("Credential field email is required", nil)
	}

	return creds, nil
}
