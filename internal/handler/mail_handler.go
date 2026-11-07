package handler

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"slices"
	"strings"

	"Astramail/internal/model"
	"Astramail/internal/service/graph"
	"Astramail/internal/service/imap"
)

// MailHandler handles email inspection and operational requests.
type MailHandler struct {
	imapSvc  *imap.Service
	graphSvc *graph.Service
}

// NewMailHandler creates a new MailHandler instance.
func NewMailHandler(imapSvc *imap.Service, graphSvc *graph.Service) *MailHandler {
	return &MailHandler{
		imapSvc:  imapSvc,
		graphSvc: graphSvc,
	}
}

// ListFolders handles GET /api/v1/folders.
func (h *MailHandler) ListFolders(w http.ResponseWriter, r *http.Request) {
	creds := Credentials(r.Context())

	ctx := r.Context()
	proto := resolveProtocol(creds, queryText(r, queryProtocol))

	var (
		folders []model.FolderInfo
		err     error
	)

	if useGraph(proto, creds) {
		folders, err = h.graphSvc.ListFolders(ctx, creds)
	} else {
		folders, err = h.imapSvc.ListFolders(ctx, creds)
	}

	if err != nil {
		HandleError(w, err)

		return
	}

	writeJSON(w, http.StatusOK, nonNil(folders))
}

// ListMessages handles GET /api/v1/messages.
func (h *MailHandler) ListMessages(w http.ResponseWriter, r *http.Request) {
	creds := Credentials(r.Context())

	page, err := queryInt(r, queryPage, 1)
	if err != nil {
		HandleError(w, err)

		return
	}

	limit, err := queryInt(r, queryLimit, defaultPageSize)
	if err != nil {
		HandleError(w, err)

		return
	}

	ctx := r.Context()
	folder := folderParam(r)
	proto := resolveProtocol(creds, queryText(r, queryProtocol))
	search := queryText(r, querySearch)

	var (
		messages []model.EmailSummary
		total    int
	)

	if useGraph(proto, creds) {
		messages, total, err = h.graphSvc.FetchMailList(ctx, creds, folder, page, limit, search)
	} else {
		messages, total, err = h.imapSvc.FetchMailList(ctx, creds, folder, page, limit, search)
	}

	if err != nil {
		HandleError(w, err)

		return
	}

	writeJSON(w, http.StatusOK, model.MessageListResponse{
		Items: nonNil(messages),
		Total: total,
		Page:  page,
		Limit: limit,
	})
}

// FetchMessageDetail handles GET /api/v1/messages/{id}.
func (h *MailHandler) FetchMessageDetail(w http.ResponseWriter, r *http.Request) {
	creds := Credentials(r.Context())

	ctx := r.Context()
	proto := resolveProtocol(creds, queryText(r, queryProtocol))
	mailID := pathParam(r, paramMailID)

	var (
		detail *model.EmailDetail
		err    error
	)

	if useGraph(proto, creds) {
		detail, err = h.graphSvc.FetchMailDetail(ctx, creds, mailID)
	} else {
		detail, err = h.imapSvc.FetchMailDetail(ctx, creds, folderParam(r), mailID)
	}

	if err != nil {
		HandleError(w, err)

		return
	}

	writeJSON(w, http.StatusOK, detail)
}

// QueryMessages handles POST /api/messageQueries: it aggregates messages from
// every account listed in the request body, then de-duplicates and sorts them.
func (h *MailHandler) QueryMessages(w http.ResponseWriter, r *http.Request) {
	var req model.MessageQueryRequest
	if !bindJSON(w, r, &req) {
		return
	}

	limit := req.Limit
	if limit <= 0 {
		limit = defaultPageSize
	}

	ctx := r.Context()

	var (
		allMessages []model.EmailSummary
		allErrors   []string
	)

	for _, creds := range req.Accounts {
		msgs, errs := h.fetchAccountMessages(ctx, creds, &req, limit)
		allMessages = append(allMessages, msgs...)
		allErrors = append(allErrors, errs...)
	}

	deduped := deduplicateAndSort(allMessages)

	writeJSON(w, http.StatusOK, model.MessageQueryResponse{
		Items:  nonNil(deduped),
		Total:  len(deduped),
		Errors: allErrors,
	})
}

type accountProtocols struct {
	// 请求意图：用户/前端明确要求使用的协议
	reqGraph bool
	reqIMAP  bool
	// 实际可执行：账号具备对应凭据
	graph bool
	imap  bool
}

func determineProtocols(creds model.AccountCredentials, reqProto *string) accountProtocols {
	proto := string(creds.Protocol)
	if reqProto != nil && *reqProto != "" {
		proto = *reqProto
	}

	isBoth := strings.EqualFold(proto, "both")
	reqGraph := isBoth || strings.EqualFold(proto, "graph")
	reqIMAP := isBoth || strings.EqualFold(proto, "imap")

	return accountProtocols{
		reqGraph: reqGraph,
		reqIMAP:  reqIMAP,
		graph:    reqGraph && creds.HasGraphCreds(),
		imap:     reqIMAP && creds.HasIMAPCreds(),
	}
}

// fetchAccountMessages 按协议拉取单个账号的邮件摘要。
func (h *MailHandler) fetchAccountMessages(
	ctx context.Context,
	creds model.AccountCredentials,
	req *model.MessageQueryRequest,
	limit int,
) ([]model.EmailSummary, []string) {
	protos := determineProtocols(creds, req.Protocol)

	var errMsgs []string

	if !protos.graph && !protos.imap {
		errStr := fmt.Sprintf("%s: No valid credentials configured (IMAP=%v, Graph=%v)",
			creds.Email, creds.HasIMAPCreds(), creds.HasGraphCreds())
		log.Printf("[QueryMessages] %s", errStr)
		errMsgs = append(errMsgs, errStr)
	}

	// 请求了 Graph 但账号缺少 Graph 凭据：明确告知而非静默跳过
	if protos.reqGraph && !protos.graph {
		errMsgs = append(errMsgs, fmt.Sprintf(
			"%s: Graph credentials missing (client ID / refresh token not set), Graph skipped",
			creds.Email,
		))
	}

	// 请求了 IMAP 但账号缺少 IMAP 凭据
	if protos.reqIMAP && !protos.imap {
		errMsgs = append(errMsgs, fmt.Sprintf(
			"%s: IMAP credentials missing (password not set), IMAP skipped",
			creds.Email,
		))
	}

	var messages []model.EmailSummary

	if protos.graph {
		// Graph 的 Junk 由 API 层映射到 junkemail，名称固定
		m, errs := h.fetchGraphFolders(ctx, creds, []string{folderInbox, folderJunk}, limit, req)
		messages = append(messages, m...)
		errMsgs = append(errMsgs, errs...)
	}

	if protos.imap {
		// IMAP 各服务商垃圾箱命名不同，先解析真实名称；
		// 解析不到就只取收件箱，不报错。
		imapFolders := []string{folderInbox}

		junkName, junkErr := h.imapSvc.ResolveJunkFolder(ctx, creds)
		if junkErr == nil && junkName != "" {
			imapFolders = append(imapFolders, junkName)
		}

		m, errs := h.fetchIMAPFolders(ctx, creds, imapFolders, limit, req)
		messages = append(messages, m...)
		errMsgs = append(errMsgs, errs...)
	}

	return messages, errMsgs
}

func (h *MailHandler) fetchGraphFolders(
	ctx context.Context,
	creds model.AccountCredentials,
	folders []string,
	limit int,
	req *model.MessageQueryRequest,
) ([]model.EmailSummary, []string) {
	var (
		messages []model.EmailSummary
		errMsgs  []string
	)

	for _, folder := range folders {
		m, err := h.fetchGraphFolder(ctx, creds, folder, limit, req)
		if err != nil {
			errMsgs = append(errMsgs, fmt.Sprintf("%s (Graph/%s): %v", creds.Email, folder, err))

			continue
		}

		messages = append(messages, m...)
	}

	return messages, errMsgs
}

func (h *MailHandler) fetchIMAPFolders(
	ctx context.Context,
	creds model.AccountCredentials,
	folders []string,
	limit int,
	req *model.MessageQueryRequest,
) ([]model.EmailSummary, []string) {
	var (
		messages []model.EmailSummary
		errMsgs  []string
	)

	for _, folder := range folders {
		m, err := h.fetchIMAPFolder(ctx, creds, folder, limit, req)
		if err != nil {
			errMsgs = append(errMsgs, fmt.Sprintf("%s (IMAP/%s): %v", creds.Email, folder, err))

			continue
		}

		messages = append(messages, m...)
	}

	return messages, errMsgs
}

func (h *MailHandler) fetchGraphFolder(
	ctx context.Context,
	creds model.AccountCredentials,
	folder string,
	limit int,
	req *model.MessageQueryRequest,
) ([]model.EmailSummary, error) {
	msgs, _, err := h.graphSvc.FetchMailList(ctx, creds, folder, 1, limit, req.Query)
	if err != nil {
		log.Printf("[QueryMessages] Graph error for %s (%s): %v", creds.Email, folder, err)

		return nil, err
	}

	return filterAndAppend(nil, msgs, req.SenderFilter), nil
}

func (h *MailHandler) fetchIMAPFolder(
	ctx context.Context,
	creds model.AccountCredentials,
	folder string,
	limit int,
	req *model.MessageQueryRequest,
) ([]model.EmailSummary, error) {
	msgs, _, err := h.imapSvc.FetchMailList(ctx, creds, folder, 1, limit, req.Query)
	if err != nil {
		log.Printf("[QueryMessages] IMAP error for %s (%s): %v", creds.Email, folder, err)

		return nil, err
	}

	return filterAndAppend(nil, msgs, req.SenderFilter), nil
}

// deduplicateAndSort 按 账号|主题|时间 去重，并按时间倒序排列。
func deduplicateAndSort(messages []model.EmailSummary) []model.EmailSummary {
	seen := make(map[string]bool)

	var deduped []model.EmailSummary

	for _, m := range messages {
		acc := ""
		if m.AccountEmail != nil {
			acc = *m.AccountEmail
		}

		key := fmt.Sprintf("%s|%s|%s", acc, m.Subject, m.Date)
		if !seen[key] {
			seen[key] = true

			deduped = append(deduped, m)
		}
	}

	slices.SortFunc(deduped, func(a, b model.EmailSummary) int {
		return strings.Compare(b.Date, a.Date)
	})

	return deduped
}

// filterAndAppend 按发件人关键字过滤后追加到结果集。
func filterAndAppend(all, msgs []model.EmailSummary, senderFilter *string) []model.EmailSummary {
	var sf string
	if senderFilter != nil {
		sf = strings.ToLower(strings.TrimSpace(*senderFilter))
	}

	for _, m := range msgs {
		if sf != "" && !strings.Contains(strings.ToLower(m.From), sf) {
			continue
		}

		all = append(all, m)
	}

	return all
}
