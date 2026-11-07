// Package model defines core domain and data transfer models for AstraMail.
package model

import (
	"encoding/json"
	"strings"
)

// ProtocolType defines the supported mail protocol types.
type ProtocolType string

// Supported email protocol constants.
const (
	// ProtocolTypeIMAP specifies standard IMAP protocol.
	ProtocolTypeIMAP ProtocolType = "imap"
	// ProtocolTypeGraph specifies Microsoft Graph REST API protocol.
	ProtocolTypeGraph ProtocolType = "graph"
	// ProtocolTypeBoth specifies attempting both IMAP and Graph protocols.
	ProtocolTypeBoth ProtocolType = "both"
)

// IMAP 连接默认值。
const (
	// defaultIMAPHost 是无法从邮箱域名推断服务器时使用的默认主机。
	defaultIMAPHost = "outlook.office365.com"
	// defaultIMAPPort 是 IMAP over TLS 的标准端口。
	defaultIMAPPort uint16 = 993
	// emailDomainParts 是 "local@domain" 按 @ 拆分后的期望片段数。
	emailDomainParts = 2
	// emailDomainIndex 是拆分邮箱地址后域名所在的下标。
	emailDomainIndex = 1
)

// AccountCredentials represents credentials and server configuration for an email account.
type AccountCredentials struct {
	Email        string       `json:"email"`
	Password     string       `json:"password,omitempty"`
	ClientID     string       `json:"clientId,omitempty"`
	RefreshToken string       `json:"refreshToken,omitempty"`
	Protocol     ProtocolType `json:"protocol"`
	IMAPHost     string       `json:"imapHost,omitempty"`
	IMAPPort     uint16       `json:"imapPort,omitempty"`
}

// UnmarshalJSON implements custom JSON unmarshaling to support both camelCase and snake_case keys.
func (a *AccountCredentials) UnmarshalJSON(data []byte) error {
	type Alias AccountCredentials

	var aux struct {
		Alias

		//nolint:tagliatelle // compatibility for snake_case
		AltClientID string `json:"client_id"`
		//nolint:tagliatelle // compatibility for snake_case
		AltRefreshToken string `json:"refresh_token"`
		//nolint:tagliatelle // compatibility for snake_case
		AltIMAPHost string `json:"imap_host"`
		//nolint:tagliatelle // compatibility for snake_case
		AltIMAPPort uint16 `json:"imap_port"`
	}

	err := json.Unmarshal(data, &aux)
	if err != nil {
		return err
	}

	*a = AccountCredentials(aux.Alias)
	if a.ClientID == "" && aux.AltClientID != "" {
		a.ClientID = aux.AltClientID
	}

	if a.RefreshToken == "" && aux.AltRefreshToken != "" {
		a.RefreshToken = aux.AltRefreshToken
	}

	if a.IMAPHost == "" && aux.AltIMAPHost != "" {
		a.IMAPHost = aux.AltIMAPHost
	}

	if a.IMAPPort == 0 && aux.AltIMAPPort != 0 {
		a.IMAPPort = aux.AltIMAPPort
	}

	return nil
}

// ResolvedIMAPHost resolves the IMAP host from custom config or email domain.
func (a *AccountCredentials) ResolvedIMAPHost() string {
	if strings.TrimSpace(a.IMAPHost) != "" {
		return strings.TrimSpace(a.IMAPHost)
	}

	parts := strings.Split(a.Email, "@")
	if len(parts) < emailDomainParts {
		return defaultIMAPHost
	}

	domain := strings.ToLower(parts[emailDomainIndex])
	switch domain {
	case "outlook.com", "hotmail.com", "live.com", "office365.com":
		return defaultIMAPHost
	case "gmail.com":
		return "imap.gmail.com"
	case "qq.com", "foxmail.com":
		return "imap.qq.com"
	case "163.com":
		return "imap.163.com"
	case "126.com":
		return "imap.126.com"
	case "sina.com":
		return "imap.sina.com"
	case "yahoo.com":
		return "imap.mail.yahoo.com"
	default:
		if domain != "" {
			return "imap." + domain
		}

		return defaultIMAPHost
	}
}

// ResolvedIMAPPort resolves the IMAP port, defaulting to 993 for TLS.
func (a *AccountCredentials) ResolvedIMAPPort() uint16 {
	if a.IMAPPort > 0 {
		return a.IMAPPort
	}

	return defaultIMAPPort
}

// HasIMAPCreds checks whether password credentials are provided.
func (a *AccountCredentials) HasIMAPCreds() bool {
	return strings.TrimSpace(a.Password) != ""
}

// HasGraphCreds checks whether client_id and refresh_token are provided.
func (a *AccountCredentials) HasGraphCreds() bool {
	return strings.TrimSpace(a.ClientID) != "" && strings.TrimSpace(a.RefreshToken) != ""
}

// ProtocolStatus holds the connectivity test result for a protocol.
type ProtocolStatus struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	FolderCount  *int   `json:"folderCount"`
	MessageCount *int   `json:"messageCount"`
}

// AccountStatus is the account resource returned by GET /api/account.
type AccountStatus struct {
	Email       string          `json:"email"`
	IMAPStatus  *ProtocolStatus `json:"imapStatus"`
	GraphStatus *ProtocolStatus `json:"graphStatus"`
}
