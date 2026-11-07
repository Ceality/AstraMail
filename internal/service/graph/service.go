// Package graph provides Microsoft Graph API mail operations.
package graph

import (
	"Astramail/internal/apperror"
	"Astramail/internal/model"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// graphBaseURL 是 Microsoft Graph v1.0 的根地址。
	graphBaseURL = "https://graph.microsoft.com/v1.0"

	// inboxURL 用于读取收件箱的邮件总数。
	inboxURL = graphBaseURL + "/me/mailFolders/Inbox"
	// foldersURL 用于列出顶层邮件文件夹。
	foldersURL = graphBaseURL + "/me/mailFolders?$top=50"
	// messageSelect 是邮件列表接口需要返回的字段集合。
	messageSelect = "id,subject,from,toRecipients,receivedDateTime,hasAttachments,bodyPreview"
	// defaultLimit 是分页大小未指定时的默认值。
	defaultLimit = 20
	// unknownSender 是无法解析出发件人时的占位显示名。
	unknownSender = "Unknown"
	// noSubject 是主题为空时的占位主题。
	noSubject = "(No Subject)"
	// defaultAttachmentName 是附件名为空时的占位名。
	defaultAttachmentName = "attachment"
	// defaultContentType 是附件类型未知时的兜底 MIME 类型。
	defaultContentType = "application/octet-stream"
	// defaultHTTPTimeout 是单次 Graph HTTP 请求的最长等待时间。
	defaultHTTPTimeout = 30 * time.Second
)

// Service provides Microsoft Graph API mail operations.
type Service struct {
	httpClient *http.Client
}

// NewService creates a new Graph Service instance.
func NewService() *Service {
	return &Service{
		httpClient: &http.Client{Timeout: defaultHTTPTimeout},
	}
}

// tokenResponse 是 OAuth2 令牌端点返回的响应，字段名固定为 snake_case。
type tokenResponse struct {
	//nolint:tagliatelle // OAuth2 规范固定返回 access_token
	AccessToken string `json:"access_token"`
}

type graphFolderItem struct {
	ID             string `json:"id"`
	DisplayName    string `json:"displayName"`
	TotalItemCount uint32 `json:"totalItemCount"`
}

type graphFolderListResponse struct {
	Value []graphFolderItem `json:"value"`
}

type recipientAddress struct {
	Address *string `json:"address"`
	Name    *string `json:"name"`
}

type recipientWrapper struct {
	EmailAddress *recipientAddress `json:"emailAddress"`
}

type messageBody struct {
	ContentType *string `json:"contentType"`
	Content     *string `json:"content"`
}

type graphAttachmentItem struct {
	Name        *string `json:"name"`
	ContentType *string `json:"contentType"`
	Size        *int    `json:"size"`
}

type graphMessageItem struct {
	ID               string                `json:"id"`
	Subject          *string               `json:"subject"`
	From             *recipientWrapper     `json:"from"`
	ToRecipients     []recipientWrapper    `json:"toRecipients"`
	CcRecipients     []recipientWrapper    `json:"ccRecipients"`
	ReceivedDateTime *string               `json:"receivedDateTime"`
	HasAttachments   bool                  `json:"hasAttachments"`
	BodyPreview      *string               `json:"bodyPreview"`
	Body             *messageBody          `json:"body"`
	Attachments      []graphAttachmentItem `json:"attachments"`
}

type graphMessageListResponse struct {
	Value []graphMessageItem `json:"value"`
	//nolint:tagliatelle // OData 计数注解固定为 @odata.count
	Count *int `json:"@odata.count"`
}

// TestConnection tests connectivity to the Graph API and returns total message count.
func (s *Service) TestConnection(ctx context.Context, creds model.AccountCredentials) (int, error) {
	token, err := s.getAccessToken(ctx, &creds)
	if err != nil {
		return 0, err
	}

	bodyBytes, err := s.doRequest(ctx, http.MethodGet, inboxURL, token, nil)
	if err != nil {
		return 0, err
	}

	var folder graphFolderItem

	err = json.Unmarshal(bodyBytes, &folder)
	if err != nil {
		return 0, apperror.NewGraphError(fmt.Sprintf("Failed to parse Inbox response: %v", err), err)
	}

	return int(folder.TotalItemCount), nil
}

// ListFolders lists all top-level mail folders.
func (s *Service) ListFolders(ctx context.Context, creds model.AccountCredentials) ([]model.FolderInfo, error) {
	token, err := s.getAccessToken(ctx, &creds)
	if err != nil {
		return nil, err
	}

	bodyBytes, err := s.doRequest(ctx, http.MethodGet, foldersURL, token, nil)
	if err != nil {
		return nil, err
	}

	var listRes graphFolderListResponse

	err = json.Unmarshal(bodyBytes, &listRes)
	if err != nil {
		return nil, apperror.NewGraphError(fmt.Sprintf("Failed to parse folders response: %v", err), err)
	}

	var folders []model.FolderInfo

	for _, f := range listRes.Value {
		folders = append(folders, model.FolderInfo{
			Name:        f.ID,
			DisplayName: f.DisplayName,
			TotalCount:  f.TotalItemCount,
		})
	}

	return folders, nil
}

// FetchMailList fetches a page of emails from a Microsoft Graph folder.
func (s *Service) FetchMailList(
	ctx context.Context,
	creds model.AccountCredentials,
	folder string,
	page int,
	limit int,
	query *string,
) ([]model.EmailSummary, int, error) {
	token, err := s.getAccessToken(ctx, &creds)
	if err != nil {
		return nil, 0, err
	}

	if page <= 0 {
		page = 1
	}

	if limit <= 0 {
		limit = defaultLimit
	}

	skip := (page - 1) * limit

	baseURL := BuildGraphListURL(folder, limit, skip, query)
	headers := map[string]string{
		"ConsistencyLevel": "eventual",
	}

	bodyBytes, err := s.doRequest(ctx, http.MethodGet, baseURL, token, headers)
	if err != nil {
		return nil, 0, err
	}

	var listRes graphMessageListResponse

	err = json.Unmarshal(bodyBytes, &listRes)
	if err != nil {
		return nil, 0, apperror.NewGraphError(fmt.Sprintf("Failed to parse messages response: %v", err), err)
	}

	total := len(listRes.Value)
	if listRes.Count != nil {
		total = *listRes.Count
	}

	var summaries []model.EmailSummary

	for _, m := range listRes.Value {
		summaries = append(summaries, convertGraphMessage(&m, creds.Email))
	}

	return summaries, total, nil
}

// FetchMailDetail fetches message detail including HTML/text body and attachments via Graph.
func (s *Service) FetchMailDetail(
	ctx context.Context,
	creds model.AccountCredentials,
	mailID string,
) (*model.EmailDetail, error) {
	token, err := s.getAccessToken(ctx, &creds)
	if err != nil {
		return nil, err
	}

	targetURL := fmt.Sprintf("%s/me/messages/%s?$expand=attachments", graphBaseURL, mailID)

	bodyBytes, err := s.doRequest(ctx, http.MethodGet, targetURL, token, nil)
	if err != nil {
		return nil, err
	}

	var m graphMessageItem

	err = json.Unmarshal(bodyBytes, &m)
	if err != nil {
		return nil, apperror.NewGraphError(fmt.Sprintf("Failed to parse message detail: %v", err), err)
	}

	return convertGraphDetail(&m), nil
}

// convertGraphDetail 将 Graph 消息转换为邮件详情。
func convertGraphDetail(m *graphMessageItem) *model.EmailDetail {
	fromStr := recipientToString(m.From)
	toList := recipientsToString(m.ToRecipients)
	ccList := recipientsToString(m.CcRecipients)

	subject := noSubject
	if m.Subject != nil && *m.Subject != "" {
		subject = *m.Subject
	}

	dateStr := ""
	if m.ReceivedDateTime != nil {
		dateStr = *m.ReceivedDateTime
	}

	body := extractGraphBody(m.Body)

	return &model.EmailDetail{
		ID:          m.ID,
		Subject:     subject,
		From:        fromStr,
		To:          toList,
		Cc:          ccList,
		Date:        dateStr,
		BodyText:    body.text,
		BodyHTML:    body.html,
		Attachments: convertGraphAttachments(m.Attachments),
		Protocol:    "graph",
	}
}

// ExecuteAction performs mail actions such as deleting messages via Graph.
func (s *Service) ExecuteAction(
	ctx context.Context,
	creds model.AccountCredentials,
	mailIDs []string,
	_ model.MailAction,
) error {
	token, err := s.getAccessToken(ctx, &creds)
	if err != nil {
		return err
	}

	for _, id := range mailIDs {
		targetURL := graphBaseURL + "/me/messages/" + id

		_, err = s.doRequest(ctx, http.MethodDelete, targetURL, token, nil)
		if err != nil {
			return err
		}
	}

	return nil
}

// doRequest performs an authenticated HTTP request to Microsoft Graph and returns response bytes.
func (s *Service) doRequest(
	ctx context.Context,
	method, targetURL, token string,
	headers map[string]string,
) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, targetURL, http.NoBody)
	if err != nil {
		return nil, apperror.NewGraphError(err.Error(), err)
	}

	req.Header.Set("Authorization", "Bearer "+token)

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, apperror.NewGraphError(fmt.Sprintf("Graph request failed: %v", err), err)
	}

	defer closeBody(resp.Body)

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, apperror.NewGraphError(fmt.Sprintf("Failed to read response body: %v", err), err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		statusErr := fmt.Sprintf("Graph operation returned status %d: %s", resp.StatusCode, bodyBytes)

		return nil, apperror.NewGraphError(statusErr, nil)
	}

	return bodyBytes, nil
}

// getAccessToken exchanges the refresh token for a Microsoft Graph access token.
func (s *Service) getAccessToken(ctx context.Context, creds *model.AccountCredentials) (string, error) {
	if strings.TrimSpace(creds.ClientID) == "" {
		return "", apperror.NewBadRequest("client_id is required for Microsoft Graph", nil)
	}

	if strings.TrimSpace(creds.RefreshToken) == "" {
		return "", apperror.NewBadRequest("refresh_token is required for Microsoft Graph", nil)
	}

	endpoints := []string{
		"https://login.microsoftonline.com/common/oauth2/v2.0/token",
		"https://login.microsoftonline.com/consumers/oauth2/v2.0/token",
	}

	var lastErr string

	for _, endpoint := range endpoints {
		token, err := s.fetchToken(ctx, endpoint, creds)
		if err == nil {
			return token, nil
		}

		lastErr = err.Error()
	}

	return "", apperror.NewAuthenticationFailed("Microsoft OAuth refresh token error: "+lastErr, nil)
}

// fetchToken 向单个 OAuth2 令牌端点换取 access_token。
func (s *Service) fetchToken(
	ctx context.Context,
	endpoint string,
	creds *model.AccountCredentials,
) (string, error) {
	formData := url.Values{
		"client_id":     {creds.ClientID},
		"grant_type":    {"refresh_token"},
		"refresh_token": {creds.RefreshToken},
		"scope":         {"https://graph.microsoft.com/.default offline_access"},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(formData.Encode()))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}

	defer closeBody(resp.Body)

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode == http.StatusOK {
		accessToken, ok := parseAccessToken(bodyBytes)
		if ok {
			return accessToken, nil
		}
	}

	return "", fmt.Errorf("status %d: %s", resp.StatusCode, bodyBytes)
}

// parseAccessToken 从 OAuth2 响应体中解析 access_token。
func parseAccessToken(bodyBytes []byte) (string, bool) {
	var tokenRes tokenResponse

	err := json.Unmarshal(bodyBytes, &tokenRes)
	if err != nil {
		return "", false
	}

	return tokenRes.AccessToken, tokenRes.AccessToken != ""
}

// closeBody 关闭响应体并记录失败：响应体已读完，关闭失败不影响业务结果。
func closeBody(body io.Closer) {
	err := body.Close()
	if err != nil {
		log.Printf("graph: failed to close response body: %v", err)
	}
}

func recipientToString(r *recipientWrapper) string {
	if r == nil || r.EmailAddress == nil {
		return unknownSender
	}

	ea := r.EmailAddress
	name := ""
	addr := ""

	if ea.Name != nil {
		name = *ea.Name
	}

	if ea.Address != nil {
		addr = *ea.Address
	}

	switch {
	case name != "" && addr != "":
		return fmt.Sprintf("%s <%s>", name, addr)
	case addr != "":
		return addr
	case name != "":
		return name
	default:
		return unknownSender
	}
}

// recipientsToString 将 Graph 收件人列表转换为显示字符串列表。
// 始终返回非 nil 切片，保证 JSON 序列化为 [] 而不是 null。
func recipientsToString(list []recipientWrapper) []string {
	result := make([]string, 0, len(list))

	for i := range list {
		result = append(result, recipientToString(&list[i]))
	}

	return result
}

// ResolveGraphFolderPath maps user-friendly folder names or standard aliases to Microsoft Graph paths.
func ResolveGraphFolderPath(folder string) string {
	switch strings.ToLower(strings.TrimSpace(folder)) {
	case "", "inbox":
		return "mailFolders/Inbox"
	case "junk", "junk email", "junkemail", "spam":
		return "mailFolders/junkemail"
	case "drafts":
		return "mailFolders/drafts"
	case "sent", "sent items", "sentitems":
		return "mailFolders/sentitems"
	case "trash", "deleted items", "deleteditems":
		return "mailFolders/deleteditems"
	default:
		return "mailFolders/" + folder
	}
}

// BuildGraphListURL generates an RFC-compliant Microsoft Graph messages URL without raw spaces.
func BuildGraphListURL(folder string, limit, skip int, query *string) string {
	folderPath := ResolveGraphFolderPath(folder)

	if query != nil && strings.TrimSpace(*query) != "" {
		escapedSearch := url.QueryEscape("\"" + strings.TrimSpace(*query) + "\"")

		return fmt.Sprintf(
			"%s/me/%s/messages?$top=%d&$skip=%d&$count=true&$select=%s&$search=%s",
			graphBaseURL, folderPath, limit, skip, messageSelect, escapedSearch,
		)
	}

	return fmt.Sprintf(
		"%s/me/%s/messages?$top=%d&$skip=%d&$count=true&$orderby=receivedDateTime%%20desc&$select=%s",
		graphBaseURL, folderPath, limit, skip, messageSelect,
	)
}

func convertGraphMessage(m *graphMessageItem, accountEmail string) model.EmailSummary {
	fromStr := recipientToString(m.From)
	toList := recipientsToString(m.ToRecipients)

	subject := noSubject
	if m.Subject != nil && *m.Subject != "" {
		subject = *m.Subject
	}

	dateStr := ""
	if m.ReceivedDateTime != nil {
		dateStr = *m.ReceivedDateTime
	}

	snippet := ""
	if m.BodyPreview != nil {
		snippet = *m.BodyPreview
	}

	return model.EmailSummary{
		ID:             m.ID,
		Subject:        subject,
		From:           fromStr,
		To:             toList,
		Date:           dateStr,
		HasAttachments: m.HasAttachments,
		Snippet:        snippet,
		Protocol:       "graph",
		AccountEmail:   &accountEmail,
	}
}

// graphBody 是 Graph 邮件正文的解析结果，text 与 html 至多只有一个非空。
type graphBody struct {
	text *string
	html *string
}

// extractGraphBody 按 contentType 判断正文是 HTML 还是纯文本。
func extractGraphBody(b *messageBody) graphBody {
	if b == nil || b.Content == nil {
		return graphBody{}
	}

	if b.ContentType != nil && strings.EqualFold(*b.ContentType, "html") {
		return graphBody{html: b.Content}
	}

	return graphBody{text: b.Content}
}

func convertGraphAttachments(items []graphAttachmentItem) []model.AttachmentInfo {
	attachments := make([]model.AttachmentInfo, 0, len(items))

	for _, a := range items {
		name := defaultAttachmentName
		if a.Name != nil && *a.Name != "" {
			name = *a.Name
		}

		ct := defaultContentType
		if a.ContentType != nil && *a.ContentType != "" {
			ct = *a.ContentType
		}

		size := 0
		if a.Size != nil {
			size = *a.Size
		}

		attachments = append(attachments, model.AttachmentInfo{
			Filename: name,
			MimeType: ct,
			Size:     size,
		})
	}

	return attachments
}
