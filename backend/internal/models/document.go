package models

import (
	"time"

	"github.com/google/uuid"
)

// Document represents a rich text document
type Document struct {
	ID            uuid.UUID              `json:"id"`
	FileID        *uuid.UUID             `json:"fileId"`
	Title         string                 `json:"title"`
	Content       string                 `json:"content"`
	Format        string                 `json:"format"` // tiptap, markdown, html
	OwnerID       uuid.UUID              `json:"ownerId"`
	OwnerName     string                 `json:"ownerName"`
	Collaborators []DocumentCollaborator `json:"collaborators"`
	IsPublic      bool                   `json:"isPublic"`
	Version       int                    `json:"version"`
	CreatedAt     time.Time              `json:"createdAt"`
	UpdatedAt     time.Time              `json:"updatedAt"`
}

// DocumentCollaborator represents a document collaborator
type DocumentCollaborator struct {
	UserID     uuid.UUID `json:"userId"`
	UserName   string    `json:"userName"`
	UserEmail  string    `json:"userEmail"`
	Permission string    `json:"permission"` // view, edit
	Color      string    `json:"color"`
	Online     bool      `json:"online"`
}

// Collaborator colors for assignment
var CollaboratorColors = []string{
	"#f44336", "#e91e63", "#9c27b0", "#673ab7", "#3f51b5",
	"#2196f3", "#03a9f4", "#00bcd4", "#009688", "#4caf50",
	"#8bc34a", "#cddc39", "#ffeb3b", "#ffc107", "#ff9800",
}
