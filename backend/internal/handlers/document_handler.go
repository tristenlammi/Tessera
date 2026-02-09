package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/tessera/tessera/internal/middleware"
	"github.com/tessera/tessera/internal/models"
	"github.com/tessera/tessera/internal/repository"
)

// DocumentHandler handles document editing endpoints
type DocumentHandler struct {
	log      zerolog.Logger
	docRepo  *repository.DocumentRepository
	userRepo *repository.UserRepository
}

// NewDocumentHandler creates a new document handler
func NewDocumentHandler(log zerolog.Logger, docRepo *repository.DocumentRepository, userRepo *repository.UserRepository) *DocumentHandler {
	return &DocumentHandler{
		log:      log,
		docRepo:  docRepo,
		userRepo: userRepo,
	}
}

// ListDocuments returns all documents for a user
func (h *DocumentHandler) ListDocuments(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	docs, err := h.docRepo.ListByOwner(c.Context(), userID)
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to list documents")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch documents",
		})
	}

	return c.JSON(fiber.Map{
		"documents": docs,
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

	doc, err := h.docRepo.GetByID(c.Context(), docID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Document not found",
		})
	}

	// Check access: must be owner or collaborator
	if doc.OwnerID != userID {
		isCollab, err := h.docRepo.IsCollaborator(c.Context(), docID, userID)
		if err != nil || !isCollab {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Document not found",
			})
		}
	}

	return c.JSON(doc)
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

	// Get owner name
	ownerName := "User"
	user, err := h.userRepo.GetByID(c.Context(), userID)
	if err == nil {
		ownerName = user.Name
	}

	now := time.Now()
	doc := &models.Document{
		ID:            uuid.New(),
		FileID:        req.FileID,
		Title:         req.Title,
		Content:       req.Content,
		Format:        req.Format,
		OwnerID:       userID,
		OwnerName:     ownerName,
		Collaborators: []models.DocumentCollaborator{},
		IsPublic:      false,
		Version:       1,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := h.docRepo.Create(c.Context(), doc); err != nil {
		h.log.Error().Err(err).Msg("Failed to create document")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create document",
		})
	}

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

	doc, err := h.docRepo.GetByID(c.Context(), docID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Document not found",
		})
	}

	// Check access: must be owner or collaborator with edit permission
	if doc.OwnerID != userID {
		isCollab, _ := h.docRepo.IsCollaborator(c.Context(), docID, userID)
		if !isCollab {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Document not found",
			})
		}
	}

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

	if err := h.docRepo.Update(c.Context(), doc); err != nil {
		h.log.Error().Err(err).Msg("Failed to update document")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update document",
		})
	}

	return c.JSON(doc)
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

	if err := h.docRepo.Delete(c.Context(), docID, userID); err != nil {
		h.log.Error().Err(err).Msg("Failed to delete document")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete document",
		})
	}

	h.log.Info().
		Str("doc_id", docID.String()).
		Str("user_id", userID.String()).
		Msg("Document deleted")

	return c.SendStatus(fiber.StatusNoContent)
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

	// Verify document exists and user is owner
	doc, err := h.docRepo.GetByID(c.Context(), docID)
	if err != nil || doc.OwnerID != userID {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Document not found",
		})
	}

	// Look up user by email
	targetUser, err := h.userRepo.GetByEmail(c.Context(), req.Email)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// Check if already shared
	isCollab, _ := h.docRepo.IsCollaborator(c.Context(), docID, targetUser.ID)
	if isCollab {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "Already shared with this user",
		})
	}

	colorIndex := len(doc.Collaborators) % len(models.CollaboratorColors)
	color := models.CollaboratorColors[colorIndex]

	if err := h.docRepo.AddCollaborator(c.Context(), docID, targetUser.ID, req.Permission, color); err != nil {
		h.log.Error().Err(err).Msg("Failed to share document")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to share document",
		})
	}

	// Re-fetch to get updated collaborators
	doc, _ = h.docRepo.GetByID(c.Context(), docID)

	return c.JSON(fiber.Map{
		"success":       true,
		"collaborators": doc.Collaborators,
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

	// Verify document exists and user is owner
	doc, err := h.docRepo.GetByID(c.Context(), docID)
	if err != nil || doc.OwnerID != userID {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Document not found",
		})
	}

	if err := h.docRepo.RemoveCollaborator(c.Context(), docID, collabID); err != nil {
		h.log.Error().Err(err).Msg("Failed to remove collaborator")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to remove collaborator",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}
