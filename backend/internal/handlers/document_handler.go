package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/tessera/tessera/internal/middleware"
)

// DocumentHandler handles document editing endpoints
type DocumentHandler struct {
	log zerolog.Logger
}

// NewDocumentHandler creates a new document handler
func NewDocumentHandler(log zerolog.Logger) *DocumentHandler {
	return &DocumentHandler{
		log: log,
	}
}

// Document represents a rich text document
type Document struct {
	ID            uuid.UUID              `json:"id"`
	FileID        *uuid.UUID             `json:"fileId"`
	Title         string                 `json:"title"`
	Content       string                 `json:"content"` // JSON (Tiptap format) or HTML
	Format        string                 `json:"format"`  // tiptap, markdown, html
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

// In-memory storage (in production, use database)
var documents = make(map[uuid.UUID][]Document)

// Collaborator colors
var collaboratorColors = []string{
	"#f44336", "#e91e63", "#9c27b0", "#673ab7", "#3f51b5",
	"#2196f3", "#03a9f4", "#00bcd4", "#009688", "#4caf50",
	"#8bc34a", "#cddc39", "#ffeb3b", "#ffc107", "#ff9800",
}

// ListDocuments returns all documents for a user
func (h *DocumentHandler) ListDocuments(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	userDocs, exists := documents[userID]
	if !exists {
		userDocs = []Document{}
	}

	return c.JSON(fiber.Map{
		"documents": userDocs,
	})
}

// GetDocument returns a single document
func (h *DocumentHandler) GetDocument(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	docID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid document ID",
		})
	}

	// Check user's own documents
	userDocs := documents[userID]
	for _, doc := range userDocs {
		if doc.ID == docID {
			return c.JSON(doc)
		}
	}

	// Check shared documents
	for ownerID, ownerDocs := range documents {
		if ownerID == userID {
			continue
		}
		for _, doc := range ownerDocs {
			if doc.ID == docID {
				for _, collab := range doc.Collaborators {
					if collab.UserID == userID {
						return c.JSON(doc)
					}
				}
			}
		}
	}

	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
		"error": "Document not found",
	})
}

// CreateDocument creates a new document
func (h *DocumentHandler) CreateDocument(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req struct {
		Title   string     `json:"title"`
		Content string     `json:"content"`
		Format  string     `json:"format"`
		FileID  *uuid.UUID `json:"fileId"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Title == "" {
		req.Title = "Untitled Document"
	}
	if req.Format == "" {
		req.Format = "tiptap"
	}
	if req.Content == "" {
		req.Content = `{"type":"doc","content":[{"type":"paragraph"}]}`
	}

	now := time.Now()
	doc := Document{
		ID:            uuid.New(),
		FileID:        req.FileID,
		Title:         req.Title,
		Content:       req.Content,
		Format:        req.Format,
		OwnerID:       userID,
		OwnerName:     "User", // Would get from user service
		Collaborators: []DocumentCollaborator{},
		IsPublic:      false,
		Version:       1,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	documents[userID] = append(documents[userID], doc)

	h.log.Info().
		Str("doc_id", doc.ID.String()).
		Str("user_id", userID.String()).
		Msg("Document created")

	return c.Status(fiber.StatusCreated).JSON(doc)
}

// UpdateDocument updates a document
func (h *DocumentHandler) UpdateDocument(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	docID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid document ID",
		})
	}

	var req struct {
		Title   *string `json:"title"`
		Content *string `json:"content"`
		Version *int    `json:"version"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	userDocs := documents[userID]
	for i, doc := range userDocs {
		if doc.ID == docID {
			// Optimistic locking
			if req.Version != nil && *req.Version != doc.Version {
				return c.Status(fiber.StatusConflict).JSON(fiber.Map{
					"error":   "Document has been modified",
					"version": doc.Version,
				})
			}

			if req.Title != nil {
				doc.Title = *req.Title
			}
			if req.Content != nil {
				doc.Content = *req.Content
			}

			doc.Version++
			doc.UpdatedAt = time.Now()
			documents[userID][i] = doc

			return c.JSON(doc)
		}
	}

	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
		"error": "Document not found",
	})
}

// DeleteDocument deletes a document
func (h *DocumentHandler) DeleteDocument(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	docID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid document ID",
		})
	}

	userDocs := documents[userID]
	for i, doc := range userDocs {
		if doc.ID == docID {
			documents[userID] = append(userDocs[:i], userDocs[i+1:]...)

			h.log.Info().
				Str("doc_id", docID.String()).
				Str("user_id", userID.String()).
				Msg("Document deleted")

			return c.SendStatus(fiber.StatusNoContent)
		}
	}

	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
		"error": "Document not found",
	})
}

// ShareDocument shares a document with another user
func (h *DocumentHandler) ShareDocument(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	docID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid document ID",
		})
	}

	var req struct {
		Email      string `json:"email"`
		Permission string `json:"permission"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Email == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Email is required",
		})
	}

	if req.Permission == "" {
		req.Permission = "view"
	}

	userDocs := documents[userID]
	for i, doc := range userDocs {
		if doc.ID == docID {
			// Check if already shared
			for _, collab := range doc.Collaborators {
				if collab.UserEmail == req.Email {
					return c.Status(fiber.StatusConflict).JSON(fiber.Map{
						"error": "Already shared with this user",
					})
				}
			}

			// Add collaborator (in production, look up user by email)
			colorIndex := len(doc.Collaborators) % len(collaboratorColors)
			collab := DocumentCollaborator{
				UserID:     uuid.New(), // Would be actual user ID
				UserName:   req.Email,  // Would be actual name
				UserEmail:  req.Email,
				Permission: req.Permission,
				Color:      collaboratorColors[colorIndex],
				Online:     false,
			}

			doc.Collaborators = append(doc.Collaborators, collab)
			doc.UpdatedAt = time.Now()
			documents[userID][i] = doc

			return c.JSON(fiber.Map{
				"success":       true,
				"collaborators": doc.Collaborators,
			})
		}
	}

	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
		"error": "Document not found",
	})
}

// RemoveCollaborator removes a collaborator from a document
func (h *DocumentHandler) RemoveCollaborator(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	docID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid document ID",
		})
	}

	collabID, err := uuid.Parse(c.Params("userId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	userDocs := documents[userID]
	for i, doc := range userDocs {
		if doc.ID == docID {
			// Remove collaborator
			newCollabs := []DocumentCollaborator{}
			for _, collab := range doc.Collaborators {
				if collab.UserID != collabID {
					newCollabs = append(newCollabs, collab)
				}
			}

			doc.Collaborators = newCollabs
			doc.UpdatedAt = time.Now()
			documents[userID][i] = doc

			return c.SendStatus(fiber.StatusNoContent)
		}
	}

	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
		"error": "Document not found",
	})
}
