package handler

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"Astramail/internal/model"
	"Astramail/internal/service/graph"
	"Astramail/internal/service/imap"
)

// AccountHandler handles account verification and management requests.
type AccountHandler struct {
	imapSvc  *imap.Service
	graphSvc *graph.Service
}

// NewAccountHandler creates a new AccountHandler instance.
func NewAccountHandler(imapSvc *imap.Service, graphSvc *graph.Service) *AccountHandler {
	return &AccountHandler{
		imapSvc:  imapSvc,
		graphSvc: graphSvc,
	}
}

// CheckAccount handles GET /api/v1/account: it reports the account identified by the
// Authorization header together with the connectivity status of each protocol.
// The optional check_protocol query parameter overrides the account's own protocol.
func (h *AccountHandler) CheckAccount(w http.ResponseWriter, r *http.Request) {
	creds := Credentials(r.Context())

	checkProto := model.ProtocolType(resolveProtocol(creds, queryText(r, queryCheckProtocol)))

	var (
		imapStatus  *model.ProtocolStatus
		graphStatus *model.ProtocolStatus
		wg          sync.WaitGroup
	)

	if checkProto == model.ProtocolTypeIMAP || checkProto == model.ProtocolTypeBoth {
		wg.Go(func() {
			imapStatus = h.testIMAP(r.Context(), creds)
		})
	}

	if checkProto == model.ProtocolTypeGraph || checkProto == model.ProtocolTypeBoth {
		wg.Go(func() {
			graphStatus = h.testGraph(r.Context(), creds)
		})
	}

	wg.Wait()

	writeJSON(w, http.StatusOK, model.AccountStatus{
		Email:       creds.Email,
		IMAPStatus:  imapStatus,
		GraphStatus: graphStatus,
	})
}

// testIMAP 验证 IMAP 凭据并返回探测结果。
func (h *AccountHandler) testIMAP(ctx context.Context, creds model.AccountCredentials) *model.ProtocolStatus {
	if !creds.HasIMAPCreds() {
		return &model.ProtocolStatus{
			Success: false,
			Message: "No password provided for IMAP",
		}
	}

	total, err := h.imapSvc.TestConnection(ctx, creds)
	if err != nil {
		return &model.ProtocolStatus{
			Success: false,
			Message: err.Error(),
		}
	}

	return &model.ProtocolStatus{
		Success:      true,
		Message:      fmt.Sprintf("IMAP connection successful. Total messages: %d", total),
		MessageCount: &total,
	}
}

// testGraph 验证 Microsoft Graph 凭据并返回探测结果。
func (h *AccountHandler) testGraph(ctx context.Context, creds model.AccountCredentials) *model.ProtocolStatus {
	if !creds.HasGraphCreds() {
		return &model.ProtocolStatus{
			Success: false,
			Message: "client_id or refresh_token missing for Microsoft Graph",
		}
	}

	total, err := h.graphSvc.TestConnection(ctx, creds)
	if err != nil {
		return &model.ProtocolStatus{
			Success: false,
			Message: err.Error(),
		}
	}

	return &model.ProtocolStatus{
		Success:      true,
		Message:      fmt.Sprintf("Microsoft Graph API successful. Total messages: %d", total),
		MessageCount: &total,
	}
}
