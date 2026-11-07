package model

// FolderInfo describes a mail folder and its message counts.
type FolderInfo struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	TotalCount  uint32 `json:"totalCount"`
}

// AttachmentInfo describes metadata of an email attachment.
type AttachmentInfo struct {
	Filename string `json:"filename"`
	MimeType string `json:"mimeType"`
	Size     int    `json:"size"`
}

// EmailSummary provides summary metadata for an email in list view.
type EmailSummary struct {
	ID             string   `json:"id"`
	Subject        string   `json:"subject"`
	From           string   `json:"from"`
	To             []string `json:"to"`
	Date           string   `json:"date"`
	HasAttachments bool     `json:"hasAttachments"`
	Snippet        string   `json:"snippet"`
	Protocol       string   `json:"protocol"`
	AccountEmail   *string  `json:"accountEmail,omitempty"`
}

// MessageQueryRequest is the request body of POST /api/messageQueries,
// which aggregates messages across several accounts at once.
type MessageQueryRequest struct {
	Accounts     []AccountCredentials `json:"accounts"`
	Limit        int                  `json:"limit"`
	Query        *string              `json:"query,omitempty"`
	SenderFilter *string              `json:"senderFilter,omitempty"`
	Protocol     *string              `json:"protocol,omitempty"`
}

// EmailDetail provides full content and attachments for a single email.
type EmailDetail struct {
	ID          string           `json:"id"`
	Subject     string           `json:"subject"`
	From        string           `json:"from"`
	To          []string         `json:"to"`
	Cc          []string         `json:"cc"`
	Date        string           `json:"date"`
	BodyText    *string          `json:"bodyText"`
	BodyHTML    *string          `json:"bodyHtml"`
	Attachments []AttachmentInfo `json:"attachments"`
	Protocol    string           `json:"protocol"`
}

// MailAction defines message operations.
type MailAction string

// Supported mail action constants.
const (
	// MailActionDelete deletes the given messages.
	MailActionDelete MailAction = "delete"
)

// MessageListResponse is the paginated message collection returned by GET /api/messages.
type MessageListResponse struct {
	Items []EmailSummary `json:"items"`
	Total int            `json:"total"`
	Page  int            `json:"page"`
	Limit int            `json:"limit"`
}

// MessageQueryResponse is the aggregated message collection returned by
// POST /api/messageQueries.
type MessageQueryResponse struct {
	Items  []EmailSummary `json:"items"`
	Total  int            `json:"total"`
	Errors []string       `json:"errors,omitempty"`
}
