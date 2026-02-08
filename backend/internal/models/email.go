package models

import (
	"encoding/json"
	"time"
)

type EmailAccount struct {
	ID           string `json:"id" db:"id"`
	UserID       string `json:"user_id" db:"user_id"`
	Name         string `json:"name" db:"name"`
	EmailAddress string `json:"email_address" db:"email_address"`

	// IMAP settings
	IMAPHost     string `json:"imap_host" db:"imap_host"`
	IMAPPort     int    `json:"imap_port" db:"imap_port"`
	IMAPUsername string `json:"imap_username" db:"imap_username"`
	IMAPPassword string `json:"-" db:"imap_password"` // Never expose in JSON
	IMAPUseTLS   bool   `json:"imap_use_tls" db:"imap_use_tls"`

	// SMTP settings
	SMTPHost     string `json:"smtp_host" db:"smtp_host"`
	SMTPPort     int    `json:"smtp_port" db:"smtp_port"`
	SMTPUsername string `json:"smtp_username" db:"smtp_username"`
	SMTPPassword string `json:"-" db:"smtp_password"` // Never expose in JSON
	SMTPUseTLS   bool   `json:"smtp_use_tls" db:"smtp_use_tls"`

	// Signature
	Signature string `json:"signature" db:"signature"` // HTML signature appended to emails

	// Sync state
	LastSyncAt *time.Time `json:"last_sync_at" db:"last_sync_at"`
	SyncError  *string    `json:"sync_error" db:"sync_error"`
	IsDefault  bool       `json:"is_default" db:"is_default"`

	// Send delay for undo-send (seconds, 0 = immediate)
	SendDelay int `json:"send_delay" db:"send_delay"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type EmailFolder struct {
	ID          string  `json:"id" db:"id"`
	AccountID   string  `json:"account_id" db:"account_id"`
	ParentID    *string `json:"parent_id,omitempty" db:"parent_id"` // For nested folders
	Name        string  `json:"name" db:"name"`
	RemoteName  string  `json:"remote_name" db:"remote_name"`
	FolderType  *string `json:"folder_type" db:"folder_type"`       // inbox, sent, drafts, trash, spam, archive, custom
	Delimiter   *string `json:"delimiter,omitempty" db:"delimiter"` // IMAP hierarchy delimiter
	SortOrder   int     `json:"sort_order" db:"sort_order"`         // Custom sort order
	UnreadCount int     `json:"unread_count" db:"unread_count"`
	TotalCount  int     `json:"total_count" db:"total_count"`
	UIDValidity *int64  `json:"uidvalidity" db:"uidvalidity"`
	UIDNext     *int64  `json:"uidnext" db:"uidnext"`

	// Not stored in DB - populated at runtime
	Children []*EmailFolder `json:"children,omitempty" db:"-"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type EmailAddress struct {
	Name    string `json:"name,omitempty"`
	Address string `json:"address"`
}

type Email struct {
	ID        string `json:"id" db:"id"`
	AccountID string `json:"account_id" db:"account_id"`
	FolderID  string `json:"folder_id" db:"folder_id"`
	MessageID string `json:"message_id" db:"message_id"`
	UID       int64  `json:"uid" db:"uid"`

	// Threading
	ThreadID         string `json:"thread_id" db:"thread_id"`
	ReferencesHeader string `json:"-" db:"references_header"` // Full References header

	// Headers
	Subject      string `json:"subject" db:"subject"`
	FromAddress  string `json:"from_address" db:"from_address"`
	FromName     string `json:"from_name" db:"from_name"`
	ToAddresses  string `json:"-" db:"to_addresses"` // JSONB stored as string
	CCAddresses  string `json:"-" db:"cc_addresses"`
	BCCAddresses string `json:"-" db:"bcc_addresses"`
	ReplyTo      string `json:"reply_to" db:"reply_to"`
	InReplyTo    string `json:"in_reply_to" db:"in_reply_to"`

	// Parsed addresses for JSON
	To  []EmailAddress `json:"to" db:"-"`
	CC  []EmailAddress `json:"cc" db:"-"`
	BCC []EmailAddress `json:"bcc" db:"-"`

	// Content
	TextBody string `json:"text_body,omitempty" db:"text_body"`
	HTMLBody string `json:"html_body,omitempty" db:"html_body"`
	Snippet  string `json:"snippet" db:"snippet"`

	// Flags
	IsRead         bool `json:"is_read" db:"is_read"`
	IsStarred      bool `json:"is_starred" db:"is_starred"`
	IsAnswered     bool `json:"is_answered" db:"is_answered"`
	IsDraft        bool `json:"is_draft" db:"is_draft"`
	HasAttachments bool `json:"has_attachments" db:"has_attachments"`

	// Dates
	Date       time.Time `json:"date" db:"date"`
	ReceivedAt time.Time `json:"received_at" db:"received_at"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`

	// Associations
	Attachments []EmailAttachment `json:"attachments,omitempty" db:"-"`
	Labels      []EmailLabel      `json:"labels,omitempty" db:"-"`
}

// ParseAddresses parses the JSONB address fields into structs
func (e *Email) ParseAddresses() {
	if e.ToAddresses != "" && e.ToAddresses != "[]" {
		json.Unmarshal([]byte(e.ToAddresses), &e.To)
	}
	if e.CCAddresses != "" && e.CCAddresses != "[]" {
		json.Unmarshal([]byte(e.CCAddresses), &e.CC)
	}
	if e.BCCAddresses != "" && e.BCCAddresses != "[]" {
		json.Unmarshal([]byte(e.BCCAddresses), &e.BCC)
	}
}

type EmailAttachment struct {
	ID          string    `json:"id" db:"id"`
	EmailID     string    `json:"email_id" db:"email_id"`
	Filename    string    `json:"filename" db:"filename"`
	ContentType string    `json:"content_type" db:"content_type"`
	Size        int64     `json:"size" db:"size"`
	ContentID   string    `json:"content_id" db:"content_id"`
	IsInline    bool      `json:"is_inline" db:"is_inline"`
	StorageKey  string    `json:"-" db:"storage_key"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// EmailListItem is a lightweight email for list views
type EmailListItem struct {
	ID             string    `json:"id"`
	ThreadID       string    `json:"thread_id"`
	Subject        string    `json:"subject"`
	FromAddress    string    `json:"from_address"`
	FromName       string    `json:"from_name"`
	Snippet        string    `json:"snippet"`
	Date           time.Time `json:"date"`
	IsRead         bool      `json:"is_read"`
	IsStarred      bool      `json:"is_starred"`
	HasAttachments bool      `json:"has_attachments"`
	ThreadCount    int       `json:"thread_count,omitempty"` // Number of emails in thread
}

// ComposeEmail represents an email being composed/sent
type ComposeEmail struct {
	AccountID       string           `json:"account_id"`
	To              []EmailAddress   `json:"to"`
	CC              []EmailAddress   `json:"cc,omitempty"`
	BCC             []EmailAddress   `json:"bcc,omitempty"`
	Subject         string           `json:"subject"`
	Body            string           `json:"body"`
	IsHTML          bool             `json:"is_html"`
	ReplyToID       string           `json:"reply_to_id,omitempty"`
	Attachments     []string         `json:"attachments,omitempty"` // File IDs from Tessera storage
	DraftID         string           `json:"draft_id,omitempty"`    // If editing a saved draft
	FileAttachments []FileAttachment `json:"-"`                     // Raw uploaded file attachments
}

// FileAttachment represents an uploaded file to be attached to an email
type FileAttachment struct {
	Filename    string
	ContentType string
	Data        []byte
}

// EmailDraft represents a saved draft
type EmailDraft struct {
	ID        string         `json:"id" db:"id"`
	AccountID string         `json:"account_id" db:"account_id"`
	To        []EmailAddress `json:"to" db:"-"`
	CC        []EmailAddress `json:"cc" db:"-"`
	BCC       []EmailAddress `json:"bcc" db:"-"`
	Subject   string         `json:"subject" db:"subject"`
	Body      string         `json:"body" db:"body"`
	IsHTML    bool           `json:"is_html" db:"is_html"`
	ReplyToID string         `json:"reply_to_id" db:"reply_to_id"`
	CreatedAt time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt time.Time      `json:"updated_at" db:"updated_at"`
}

// PendingSend represents a queued email waiting to be sent (for undo-send)
type PendingSend struct {
	ID        string       `json:"id"`
	AccountID string       `json:"account_id"`
	Compose   ComposeEmail `json:"compose"`
	SendAt    time.Time    `json:"send_at"`
	Cancelled bool         `json:"cancelled"`
}

// EmailLabel represents a custom label (like Gmail labels)
type EmailLabel struct {
	ID        string    `json:"id" db:"id"`
	AccountID string    `json:"account_id" db:"account_id"`
	Name      string    `json:"name" db:"name"`
	Color     string    `json:"color" db:"color"`
	IsSystem  bool      `json:"is_system" db:"is_system"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`

	// Computed fields
	EmailCount int `json:"email_count,omitempty" db:"-"`
}

// EmailLabelAssignment represents a label applied to an email
type EmailLabelAssignment struct {
	ID        string    `json:"id" db:"id"`
	EmailID   string    `json:"email_id" db:"email_id"`
	LabelID   string    `json:"label_id" db:"label_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// RuleCondition represents a single condition in an email rule
type RuleCondition struct {
	Field    string `json:"field"`    // from, to, subject, body
	Operator string `json:"operator"` // contains, equals, startswith, endswith, regex
	Value    string `json:"value"`
}

// RuleAction represents an action to take when a rule matches
type RuleAction struct {
	Type  string `json:"type"`  // label, move, star, mark_read, archive, delete
	Value string `json:"value"` // label_id, folder_id, or empty for star/mark_read/delete
}

// EmailRule represents an automatic email filtering rule
type EmailRule struct {
	ID             string          `json:"id" db:"id"`
	AccountID      string          `json:"account_id" db:"account_id"`
	Name           string          `json:"name" db:"name"`
	IsEnabled      bool            `json:"is_enabled" db:"is_enabled"`
	Priority       int             `json:"priority" db:"priority"`
	MatchType      string          `json:"match_type" db:"match_type"` // any, all
	ConditionsJSON string          `json:"-" db:"conditions"`
	ActionsJSON    string          `json:"-" db:"actions"`
	Conditions     []RuleCondition `json:"conditions" db:"-"`
	Actions        []RuleAction    `json:"actions" db:"-"`
	StopProcessing bool            `json:"stop_processing" db:"stop_processing"`
	CreatedAt      time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at" db:"updated_at"`
}

// ParseRuleJSON parses the JSONB fields into structs
func (r *EmailRule) ParseRuleJSON() {
	if r.ConditionsJSON != "" && r.ConditionsJSON != "[]" {
		json.Unmarshal([]byte(r.ConditionsJSON), &r.Conditions)
	}
	if r.ActionsJSON != "" && r.ActionsJSON != "[]" {
		json.Unmarshal([]byte(r.ActionsJSON), &r.Actions)
	}
}

// SerializeRuleJSON converts the struct fields to JSON strings
func (r *EmailRule) SerializeRuleJSON() error {
	condBytes, err := json.Marshal(r.Conditions)
	if err != nil {
		return err
	}
	r.ConditionsJSON = string(condBytes)

	actBytes, err := json.Marshal(r.Actions)
	if err != nil {
		return err
	}
	r.ActionsJSON = string(actBytes)
	return nil
}

// EmailThread represents a conversation thread containing multiple emails
type EmailThread struct {
	ThreadID       string          `json:"thread_id"`
	Subject        string          `json:"subject"`      // Subject from latest email
	Snippet        string          `json:"snippet"`      // Snippet from latest email
	LatestDate     time.Time       `json:"latest_date"`  // Most recent email date
	EmailCount     int             `json:"email_count"`  // Total emails in thread
	UnreadCount    int             `json:"unread_count"` // Unread emails in thread
	HasAttachments bool            `json:"has_attachments"`
	IsStarred      bool            `json:"is_starred"`       // Any email starred
	Participants   []EmailAddress  `json:"participants"`     // All participants
	LatestEmail    *EmailListItem  `json:"latest_email"`     // The most recent email
	Emails         []EmailListItem `json:"emails,omitempty"` // All emails when expanded
}
