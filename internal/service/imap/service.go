// Package imap provides IMAP protocol integration and RFC822 mail processing.
package imap

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	_ "github.com/emersion/go-message/charset" // registers GBK/Big5 MIME charsets
	"github.com/emersion/go-message/mail"

	"Astramail/internal/apperror"
	"Astramail/internal/model"
)

const (
	// inboxFolderName 是探测连接时使用的收件箱名称。
	inboxFolderName = "INBOX"
	// mailboxBufferSize 是接收 mailbox 列表的 channel 缓冲大小。
	mailboxBufferSize = 50
	// snippetReadLimit 是生成摘要时最多读取的正文字节数。
	snippetReadLimit = 1024
	// snippetMaxRunes 是摘要保留的最大字符数。
	snippetMaxRunes = 120
	// unknownSender 是无法解析发件人时的占位显示名。
	unknownSender = "Unknown"
	// noSubject 是主题为空时的占位主题。
	noSubject = "(No Subject)"
	// defaultAttachmentName 是附件名为空时的占位名。
	defaultAttachmentName = "attachment"
	// defaultContentType 是附件类型未知时的兜底 MIME 类型。
	defaultContentType = "application/octet-stream"
)

// Service provides IMAP operations.
type Service struct{}

// NewService creates a new IMAP Service instance.
func NewService() *Service {
	return &Service{}
}

// TestConnection verifies credentials and returns total message count in INBOX.
func (s *Service) TestConnection(ctx context.Context, creds model.AccountCredentials) (int, error) {
	c, err := s.connectSession(ctx, &creds)
	if err != nil {
		return 0, err
	}
	defer closeSession(c)

	err = ctx.Err()
	if err != nil {
		return 0, err
	}

	mbox, err := c.Select(inboxFolderName, true)
	if err != nil {
		return 0, apperror.NewIMAPError(fmt.Sprintf("Failed to select INBOX: %v", err), err)
	}

	return int(mbox.Messages), nil
}

// junkFolderCandidates 是各服务商垃圾箱的常见命名，按优先级排列。
// 不同 IMAP 服务商命名不一致（189.cn 用 "垃圾邮件"，Gmail 用 "[Gmail]/Spam"，
// Outlook 用 "Junk"），必须先 LIST 再匹配，不能硬编码。
var junkFolderCandidates = []string{
	"Junk",
	"Junk E-mail",
	"Junk Email",
	"Spam",
	"[Gmail]/Spam",
	"垃圾邮件",
	"垃圾箱",
	"Junk Mail",
}

// ResolveJunkFolder 通过 LIST 找出该账号真实的垃圾箱名。
// 找不到时返回空字符串与 nil —— 表示该账号没有垃圾箱，调用方应跳过而非报错。
func (s *Service) ResolveJunkFolder(ctx context.Context, creds model.AccountCredentials) (string, error) {
	folders, err := s.ListFolders(ctx, creds)
	if err != nil {
		return "", err
	}

	byName := make(map[string]string, len(folders))
	for _, f := range folders {
		byName[strings.ToLower(strings.Trim(f.Name, "\""))] = f.Name
		byName[strings.ToLower(f.DisplayName)] = f.Name
	}

	for _, candidate := range junkFolderCandidates {
		if name, ok := byName[strings.ToLower(candidate)]; ok {
			return name, nil
		}
	}

	// 名称里带 junk/spam/垃圾 的也认
	for _, f := range folders {
		lower := strings.ToLower(f.DisplayName)
		if strings.Contains(lower, "junk") || strings.Contains(lower, "spam") ||
			strings.Contains(f.DisplayName, "垃圾") {
			return f.Name, nil
		}
	}

	return "", nil
}

// ListFolders retrieves all mailboxes/folders for the account.
func (s *Service) ListFolders(ctx context.Context, creds model.AccountCredentials) ([]model.FolderInfo, error) {
	c, err := s.connectSession(ctx, &creds)
	if err != nil {
		return nil, err
	}
	defer closeSession(c)

	mailboxes := make(chan *imap.MailboxInfo, mailboxBufferSize)
	done := make(chan error, 1)

	go func() {
		done <- c.List("", "*", mailboxes)
	}()

	var collectedNames []string

	for m := range mailboxes {
		if m != nil {
			collectedNames = append(collectedNames, m.Name)
		}
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case err := <-done:
		if err != nil {
			return nil, apperror.NewIMAPError(fmt.Sprintf("Failed to list mailboxes: %v", err), err)
		}
	}

	var folderInfos []model.FolderInfo

	for _, name := range collectedNames {
		err := ctx.Err()
		if err != nil {
			return nil, err
		}

		total := uint32(0)

		mb, err := c.Select(name, true)
		if err == nil && mb != nil {
			total = mb.Messages
		}

		displayName := strings.Trim(name, "\"")
		folderInfos = append(folderInfos, model.FolderInfo{
			Name:        name,
			DisplayName: displayName,
			TotalCount:  total,
		})
	}

	return folderInfos, nil
}

// FetchMailList retrieves paginated email summaries from a given folder.
func (s *Service) FetchMailList(
	ctx context.Context,
	creds model.AccountCredentials,
	folder string,
	page int,
	limit int,
	query *string,
) ([]model.EmailSummary, int, error) {
	c, err := s.connectSession(ctx, &creds)
	if err != nil {
		return nil, 0, err
	}
	defer closeSession(c)

	err = ctx.Err()
	if err != nil {
		return nil, 0, err
	}

	mbox, err := c.Select(folder, true)
	if err != nil {
		return nil, 0, apperror.NewIMAPError(fmt.Sprintf("Failed to select folder %q: %v", folder, err), err)
	}

	total := mbox.Messages

	seqSet, ok := computeSeqRange(total, page, limit)
	if !ok {
		return []model.EmailSummary{}, int(total), nil
	}

	summaries, err := collectIMAPMessages(ctx, c, seqSet, limit, query, creds.Email)
	if err != nil {
		return nil, 0, err
	}

	return summaries, int(total), nil
}

// FetchMailDetail retrieves full email details including bodies and attachments.
func (s *Service) FetchMailDetail(
	ctx context.Context,
	creds model.AccountCredentials,
	folder string,
	mailID string,
) (*model.EmailDetail, error) {
	c, err := s.connectSession(ctx, &creds)
	if err != nil {
		return nil, err
	}
	defer closeSession(c)

	err = ctx.Err()
	if err != nil {
		return nil, err
	}

	_, err = c.Select(folder, true)
	if err != nil {
		return nil, apperror.NewIMAPError(fmt.Sprintf("Failed to select folder %q: %v", folder, err), err)
	}

	uid, err := strconv.ParseUint(mailID, 10, 32)
	if err != nil {
		return nil, apperror.NewBadRequest("Invalid mail ID: "+mailID, nil)
	}

	section := &imap.BodySectionName{}

	targetMsg, err := fetchIMAPMessageByUID(ctx, c, uint32(uid), section)
	if err != nil {
		return nil, err
	}

	body := targetMsg.GetBody(section)
	if body == nil {
		return nil, apperror.NewParseError("Message body is empty", nil)
	}

	return parseIMAPMessageDetail(mailID, body)
}

// ExecuteAction performs actions such as deleting emails.
func (s *Service) ExecuteAction(
	ctx context.Context,
	creds model.AccountCredentials,
	folder string,
	mailIDs []string,
	_ model.MailAction,
) error {
	c, err := s.connectSession(ctx, &creds)
	if err != nil {
		return err
	}
	defer closeSession(c)

	err = ctx.Err()
	if err != nil {
		return err
	}

	_, err = c.Select(folder, false)
	if err != nil {
		return apperror.NewIMAPError(fmt.Sprintf("Failed to select folder %q: %v", folder, err), err)
	}

	for _, id := range mailIDs {
		err = ctx.Err()
		if err != nil {
			return err
		}

		uid, err := strconv.ParseUint(id, 10, 32)
		if err != nil {
			continue
		}

		err = deleteMessage(c, uint32(uid))
		if err != nil {
			return err
		}
	}

	return nil
}

// deleteMessage 标记指定 UID 的邮件为删除并立即执行 expunge。
func deleteMessage(c *client.Client, uid uint32) error {
	seqSet := new(imap.SeqSet)
	seqSet.AddNum(uid)

	item := imap.FormatFlagsOp(imap.AddFlags, false)
	flags := []any{imap.DeletedFlag}

	err := c.UidStore(seqSet, item, flags, nil)
	if err != nil {
		return apperror.NewIMAPError(fmt.Sprintf("Failed to mark message %d as deleted: %v", uid, err), err)
	}

	err = c.Expunge(nil)
	if err != nil {
		return apperror.NewIMAPError(fmt.Sprintf("Failed to expunge deleted message %d: %v", uid, err), err)
	}

	return nil
}

// closeSession safely logs out and closes the IMAP client.
func closeSession(c *client.Client) {
	if c == nil {
		return
	}

	err := c.Logout()
	if err != nil {
		log.Printf("imap: logout failed: %v", err)
	}

	closeClient(c)
}

// closeClient 关闭 IMAP 客户端并记录失败：清理阶段的错误不应掩盖主流程结果。
func closeClient(c *client.Client) {
	err := c.Close()
	if err != nil {
		log.Printf("imap: failed to close client: %v", err)
	}
}

// closeConn 关闭底层连接并记录失败。
func closeConn(conn io.Closer) {
	err := conn.Close()
	if err != nil {
		log.Printf("imap: failed to close connection: %v", err)
	}
}

// parseSender extracts a display name and email address from the From header.
func parseSender(header *mail.Header) string {
	fromList, err := header.AddressList("From")
	if err == nil && len(fromList) > 0 && fromList[0] != nil {
		if fromList[0].Name != "" {
			return fmt.Sprintf("%s <%s>", fromList[0].Name, fromList[0].Address)
		}

		return fromList[0].Address
	}

	return unknownSender
}

// parseAddressList extracts clean email addresses from a header field.
// 返回空切片而不是 nil，保证 JSON 序列化为 []（与 Rust 端 Vec<String> 契约一致）。
func parseAddressList(header *mail.Header, key string) []string {
	list, err := header.AddressList(key)
	if err != nil || len(list) == 0 {
		return []string{}
	}

	addrs := make([]string, 0, len(list))

	for _, a := range list {
		if a != nil && a.Address != "" {
			addrs = append(addrs, a.Address)
		}
	}

	return addrs
}

// parseDate extracts RFC3339 formatted date from mail header.
func parseDate(header *mail.Header) string {
	t, err := header.Date()
	if err == nil && !t.IsZero() {
		return t.Format(time.RFC3339)
	}

	return ""
}

func collectIMAPMessages(
	ctx context.Context,
	c *client.Client,
	seqSet *imap.SeqSet,
	limit int,
	query *string,
	email string,
) ([]model.EmailSummary, error) {
	section := &imap.BodySectionName{}
	items := []imap.FetchItem{
		imap.FetchUid,
		section.FetchItem(),
	}

	messages := make(chan *imap.Message, limit)
	done := make(chan error, 1)

	go func() {
		done <- c.Fetch(seqSet, items, messages)
	}()

	summaries := make([]model.EmailSummary, 0, limit)

	var queryLower string
	if query != nil && strings.TrimSpace(*query) != "" {
		queryLower = strings.ToLower(strings.TrimSpace(*query))
	}

	for msg := range messages {
		if summary, ok := parseIMAPSummary(msg, section, queryLower, email); ok {
			summaries = append(summaries, *summary)
		}
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case err := <-done:
		if err != nil {
			return nil, apperror.NewIMAPError(fmt.Sprintf("Failed to fetch messages: %v", err), err)
		}
	}

	for i, j := 0, len(summaries)-1; i < j; i, j = i+1, j-1 {
		summaries[i], summaries[j] = summaries[j], summaries[i]
	}

	return summaries, nil
}

func computeSeqRange(total uint32, page, limit int) (*imap.SeqSet, bool) {
	if total == 0 {
		return nil, false
	}

	if page <= 0 {
		page = 1
	}

	if limit <= 0 {
		limit = 20
	}

	uPage := uint32(page)
	uLimit := uint32(limit)

	skip := (uPage - 1) * uLimit
	if skip >= total {
		return nil, false
	}

	endSeq := total - skip

	var startSeq uint32 = 1
	if endSeq > uLimit {
		startSeq = endSeq - uLimit + 1
	}

	seqSet := new(imap.SeqSet)
	seqSet.AddRange(startSeq, endSeq)

	return seqSet, true
}

func parseIMAPSummary(
	msg *imap.Message,
	section *imap.BodySectionName,
	queryLower string,
	accountEmail string,
) (*model.EmailSummary, bool) {
	if msg == nil {
		return nil, false
	}

	id := strconv.FormatUint(uint64(msg.Uid), 10)
	if msg.Uid == 0 {
		id = strconv.FormatUint(uint64(msg.SeqNum), 10)
	}

	body := msg.GetBody(section)
	if body == nil {
		return nil, false
	}

	mr, err := mail.CreateReader(body)
	if err != nil {
		return nil, false
	}

	header := mr.Header

	subject := header.Get("Subject")
	if subject == "" {
		subject = noSubject
	}

	if queryLower != "" && !strings.Contains(strings.ToLower(subject), queryLower) {
		return nil, false
	}

	fromStr := parseSender(&header)
	toList := parseAddressList(&header, "To")
	dateStr := parseDate(&header)
	snippet, hasAttachments := extractSnippetAndAttachments(mr)

	return &model.EmailSummary{
		ID:             id,
		Subject:        subject,
		From:           fromStr,
		To:             toList,
		Date:           dateStr,
		HasAttachments: hasAttachments,
		Snippet:        snippet,
		Protocol:       "imap",
		AccountEmail:   &accountEmail,
	}, true
}

func extractSnippetAndAttachments(mr *mail.Reader) (string, bool) {
	snippet := ""
	hasAttachments := false

	for {
		p, err := mr.NextPart()
		if err != nil {
			break
		}

		switch h := p.Header.(type) {
		case *mail.InlineHeader:
			contentType := h.Get("Content-Type")
			if strings.HasPrefix(strings.ToLower(contentType), "text/plain") && snippet == "" {
				snippet = readSnippet(p.Body)
			}
		case *mail.AttachmentHeader:
			hasAttachments = true
		default:
			// 其他部分类型（如内嵌 multipart 容器）无需处理
		}
	}

	return snippet, hasAttachments
}

// readSnippet 读取正文片段并裁剪为单行摘要。
func readSnippet(body io.Reader) string {
	b, err := io.ReadAll(io.LimitReader(body, snippetReadLimit))
	if err != nil {
		return ""
	}

	clean := strings.TrimSpace(strings.ReplaceAll(string(b), "\n", " "))

	runes := []rune(clean)
	if len(runes) > snippetMaxRunes {
		return string(runes[:snippetMaxRunes]) + "..."
	}

	return clean
}

func parseIMAPMessageDetail(mailID string, body io.Reader) (*model.EmailDetail, error) {
	mr, err := mail.CreateReader(body)
	if err != nil {
		return nil, apperror.NewParseError(fmt.Sprintf("Failed to parse RFC822 message: %v", err), err)
	}

	header := mr.Header

	subject := header.Get("Subject")
	if subject == "" {
		subject = noSubject
	}

	fromStr := parseSender(&header)
	toAddrs := parseAddressList(&header, "To")
	ccAddrs := parseAddressList(&header, "Cc")
	dateStr := parseDate(&header)

	parsed, attachments := parseIMAPBodyAndAttachments(mr)

	return &model.EmailDetail{
		ID:          mailID,
		Subject:     subject,
		From:        fromStr,
		To:          toAddrs,
		Cc:          ccAddrs,
		Date:        dateStr,
		BodyText:    parsed.text,
		BodyHTML:    parsed.html,
		Attachments: attachments,
		Protocol:    "imap",
	}, nil
}

// mailBody 是解析出的邮件正文，text 与 html 至多只有一个非空。
type mailBody struct {
	text *string
	html *string
}

func parseIMAPBodyAndAttachments(mr *mail.Reader) (mailBody, []model.AttachmentInfo) {
	var (
		body        mailBody
		attachments []model.AttachmentInfo
	)

	for {
		p, err := mr.NextPart()
		if err != nil {
			break
		}

		switch h := p.Header.(type) {
		case *mail.InlineHeader:
			contentType := strings.ToLower(h.Get("Content-Type"))

			b, readErr := io.ReadAll(p.Body)
			if readErr != nil {
				continue
			}

			contentStr := string(b)
			if strings.HasPrefix(contentType, "text/plain") && body.text == nil {
				body.text = &contentStr
			} else if strings.HasPrefix(contentType, "text/html") && body.html == nil {
				body.html = &contentStr
			}
		case *mail.AttachmentHeader:
			attachment, ok := readAttachment(h, p.Body)
			if ok {
				attachments = append(attachments, attachment)
			}
		default:
			// 其他部分类型（如内嵌 multipart 容器）无需提取内容
		}
	}

	if attachments == nil {
		attachments = []model.AttachmentInfo{}
	}

	return body, attachments
}

// readAttachment 读取附件部分的元数据，读取失败时返回 false。
func readAttachment(h *mail.AttachmentHeader, body io.Reader) (model.AttachmentInfo, bool) {
	filename, err := h.Filename()
	if err != nil || filename == "" {
		filename = defaultAttachmentName
	}

	contentType := h.Get("Content-Type")
	if contentType == "" {
		contentType = defaultContentType
	}

	data, err := io.ReadAll(body)
	if err != nil {
		return model.AttachmentInfo{}, false
	}

	return model.AttachmentInfo{
		Filename: filename,
		MimeType: contentType,
		Size:     len(data),
	}, true
}

func fetchIMAPMessageByUID(
	ctx context.Context,
	c *client.Client,
	uid uint32,
	section *imap.BodySectionName,
) (*imap.Message, error) {
	seqSet := new(imap.SeqSet)
	seqSet.AddNum(uid)

	items := []imap.FetchItem{
		imap.FetchUid,
		section.FetchItem(),
	}

	messages := make(chan *imap.Message, 1)
	done := make(chan error, 1)

	go func() {
		done <- c.UidFetch(seqSet, items, messages)
	}()

	var targetMsg *imap.Message

	for msg := range messages {
		targetMsg = msg
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case err := <-done:
		if err != nil {
			return nil, apperror.NewIMAPError(fmt.Sprintf("Failed to fetch message %d: %v", uid, err), err)
		}
	}

	if targetMsg == nil {
		return nil, apperror.NewIMAPError(fmt.Sprintf("Message %d not found", uid), nil)
	}

	return targetMsg, nil
}

// connectSession establishes a TLS connection and authenticates with the IMAP server using context.
func (*Service) connectSession(ctx context.Context, creds *model.AccountCredentials) (*client.Client, error) {
	err := ctx.Err()
	if err != nil {
		return nil, err
	}

	if !creds.HasIMAPCreds() {
		return nil, apperror.NewBadRequest("Password is required for IMAP", nil)
	}

	host := creds.ResolvedIMAPHost()
	port := creds.ResolvedIMAPPort()
	addr := fmt.Sprintf("%s:%d", host, port)

	dialer := &tls.Dialer{
		Config: &tls.Config{
			ServerName: host,
		},
	}

	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, apperror.NewIMAPError(fmt.Sprintf("Connection to %s failed: %v", addr, err), err)
	}

	c, err := client.New(conn)
	if err != nil {
		closeConn(conn)

		return nil, apperror.NewIMAPError(fmt.Sprintf("IMAP client init failed: %v", err), err)
	}

	err = c.Login(creds.Email, creds.Password)
	if err != nil {
		closeClient(c)

		return nil, apperror.NewAuthenticationFailed(fmt.Sprintf("IMAP login failed for %s: %v", creds.Email, err), err)
	}

	return c, nil
}
