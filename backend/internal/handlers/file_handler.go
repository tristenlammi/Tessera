package handlers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/tessera/tessera/internal/middleware"
	"github.com/tessera/tessera/internal/repository"
	"github.com/tessera/tessera/internal/services"
	"github.com/tessera/tessera/internal/storage"
	"github.com/tessera/tessera/internal/websocket"
)

// FileHandler handles file operation endpoints
type FileHandler struct {
	fileService *services.FileService
	log         zerolog.Logger
	hub         *websocket.Hub
	jwtSecret   string
	storage     storage.Storage
}

// NewFileHandler creates a new file handler
func NewFileHandler(fileService *services.FileService, log zerolog.Logger, hub *websocket.Hub, jwtSecret string, store storage.Storage) *FileHandler {
	return &FileHandler{
		fileService: fileService,
		log:         log,
		hub:         hub,
		jwtSecret:   jwtSecret,
		storage:     store,
	}
}

// List retrieves files in a folder
func (h *FileHandler) List(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var parentID *uuid.UUID
	if parentIDStr := c.Query("parent_id"); parentIDStr != "" {
		id, err := uuid.Parse(parentIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid parent_id",
			})
		}
		parentID = &id
	}

	files, err := h.fileService.List(c.Context(), services.ListInput{
		OwnerID:  userID,
		ParentID: parentID,
	})

	if err != nil {
		h.log.Error().Err(err).Msg("Failed to list files")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list files",
		})
	}

	return c.JSON(fiber.Map{
		"files": files,
	})
}

// Get retrieves a single file
func (h *FileHandler) Get(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	fileID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid file ID",
		})
	}

	file, err := h.fileService.Get(c.Context(), fileID, userID)
	if err != nil {
		if err == repository.ErrFileNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "File not found",
			})
		}
		h.log.Error().Err(err).Msg("Failed to get file")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get file",
		})
	}

	return c.JSON(file)
}

// CreateFolderRequest represents the folder creation payload
type CreateFolderRequest struct {
	Name     string  `json:"name" validate:"required"`
	ParentID *string `json:"parent_id"`
}

// CreateFolder creates a new folder
func (h *FileHandler) CreateFolder(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req CreateFolderRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	var parentID *uuid.UUID
	if req.ParentID != nil {
		id, err := uuid.Parse(*req.ParentID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid parent_id",
			})
		}
		parentID = &id
	}

	folder, err := h.fileService.CreateFolder(c.Context(), services.CreateFolderInput{
		OwnerID:  userID,
		ParentID: parentID,
		Name:     req.Name,
	})

	if err != nil {
		h.log.Error().Err(err).Msg("Failed to create folder")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create folder",
		})
	}

	// Broadcast file created event
	h.broadcastFileEvent(websocket.EventFileCreated, folder, userID, parentID)

	return c.Status(fiber.StatusCreated).JSON(folder)
}

// CreateDocumentRequest represents the document file creation payload
type CreateDocumentRequest struct {
	Name     string  `json:"name" validate:"required"`
	Title    string  `json:"title"`
	Content  string  `json:"content"`
	Format   string  `json:"format"`
	ParentID *string `json:"parentId"`
}

// CreateDocumentFile creates a new document file (.tdoc)
func (h *FileHandler) CreateDocumentFile(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req CreateDocumentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Ensure filename ends with .tdoc
	if !strings.HasSuffix(req.Name, ".tdoc") {
		req.Name = req.Name + ".tdoc"
	}

	// Set defaults
	if req.Title == "" {
		req.Title = strings.TrimSuffix(req.Name, ".tdoc")
	}
	if req.Format == "" {
		req.Format = "html"
	}
	if req.Content == "" {
		req.Content = "<p></p>"
	}

	var parentID *uuid.UUID
	if req.ParentID != nil && *req.ParentID != "" {
		id, err := uuid.Parse(*req.ParentID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid parentId",
			})
		}
		parentID = &id
	}

	// Create document content JSON
	docContent := map[string]interface{}{
		"title":   req.Title,
		"content": req.Content,
		"format":  req.Format,
	}
	contentBytes, err := json.Marshal(docContent)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create document",
		})
	}

	// Upload document as file
	file, err := h.fileService.UploadFile(c.Context(), services.UploadInput{
		OwnerID:  userID,
		ParentID: parentID,
		Name:     req.Name,
		Size:     int64(len(contentBytes)),
		Reader:   bytes.NewReader(contentBytes),
	})

	if err != nil {
		h.log.Error().Err(err).Msg("Failed to create document file")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create document file",
		})
	}

	// Broadcast file created event
	h.broadcastFileEvent(websocket.EventFileCreated, file, userID, parentID)

	return c.Status(fiber.StatusCreated).JSON(file)
}

// GetDocumentContent retrieves the content of a document file
func (h *FileHandler) GetDocumentContent(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	fileID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid file ID",
		})
	}

	// Get file metadata
	file, err := h.fileService.Get(c.Context(), fileID, userID)
	if err != nil {
		if err == repository.ErrFileNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "File not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get file",
		})
	}

	// Verify it's a document file
	if !strings.HasSuffix(file.Name, ".tdoc") {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Not a document file",
		})
	}

	// Get file content
	reader, _, err := h.fileService.Download(c.Context(), fileID, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to read document",
		})
	}
	defer reader.Close()

	// Parse document content
	var docContent map[string]interface{}
	if err := json.NewDecoder(reader).Decode(&docContent); err != nil {
		// If parsing fails, return empty content (legacy file)
		return c.JSON(fiber.Map{
			"title":   strings.TrimSuffix(file.Name, ".tdoc"),
			"content": "",
			"format":  "html",
		})
	}

	return c.JSON(docContent)
}

// UpdateDocumentContentRequest represents document content update payload
type UpdateDocumentContentRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	Format  string `json:"format"`
}

// UpdateDocumentContent updates the content of a document file
func (h *FileHandler) UpdateDocumentContent(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	fileID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid file ID",
		})
	}

	var req UpdateDocumentContentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Get file metadata to verify ownership and type
	file, err := h.fileService.Get(c.Context(), fileID, userID)
	if err != nil {
		if err == repository.ErrFileNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "File not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get file",
		})
	}

	// Verify it's a document file
	if !strings.HasSuffix(file.Name, ".tdoc") {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Not a document file",
		})
	}

	// Create updated content
	if req.Format == "" {
		req.Format = "html"
	}
	docContent := map[string]interface{}{
		"title":   req.Title,
		"content": req.Content,
		"format":  req.Format,
	}
	contentBytes, err := json.Marshal(docContent)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update document",
		})
	}

	// Update file content using versioning
	updatedFile, err := h.fileService.UpdateFileContentWithInput(c.Context(), services.UpdateContentInput{
		FileID:  fileID,
		OwnerID: userID,
		Reader:  bytes.NewReader(contentBytes),
		Size:    int64(len(contentBytes)),
	})

	if err != nil {
		h.log.Error().Err(err).Msg("Failed to update document content")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update document",
		})
	}

	return c.JSON(updatedFile)
}

// UpdateRequest represents the file update payload
type UpdateRequest struct {
	Name      *string `json:"name"`
	ParentID  *string `json:"parent_id"`
	IsStarred *bool   `json:"is_starred"`
}

// Update modifies a file's metadata
func (h *FileHandler) Update(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	fileID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid file ID",
		})
	}

	var req UpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	input := services.UpdateInput{
		FileID:    fileID,
		OwnerID:   userID,
		Name:      req.Name,
		IsStarred: req.IsStarred,
	}

	if req.ParentID != nil {
		id, err := uuid.Parse(*req.ParentID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid parent_id",
			})
		}
		input.ParentID = &id
	}

	file, err := h.fileService.Update(c.Context(), input)
	if err != nil {
		if err == repository.ErrFileNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "File not found",
			})
		}
		h.log.Error().Err(err).Msg("Failed to update file")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update file",
		})
	}

	// Broadcast update event
	if req.ParentID != nil {
		// File was moved
		h.broadcastFileEvent(websocket.EventFileMoved, file, userID, file.ParentID)
	} else {
		h.broadcastFileEvent(websocket.EventFileUpdated, file, userID, file.ParentID)
	}

	return c.JSON(file)
}

// Delete moves a file to trash
func (h *FileHandler) Delete(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	fileID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid file ID",
		})
	}

	// Check if permanent delete is requested
	permanent := c.Query("permanent") == "true"

	// Get file info before deleting for the event
	file, _ := h.fileService.Get(c.Context(), fileID, userID)

	if permanent {
		if err := h.fileService.PermanentDelete(c.Context(), fileID, userID); err != nil {
			if err == repository.ErrFileNotFound {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"error": "File not found",
				})
			}
			h.log.Error().Err(err).Msg("Failed to permanently delete file")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to delete file",
			})
		}
	} else {
		if err := h.fileService.Delete(c.Context(), fileID, userID); err != nil {
			if err == repository.ErrFileNotFound {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"error": "File not found",
				})
			}
			h.log.Error().Err(err).Msg("Failed to trash file")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to delete file",
			})
		}
	}

	// Broadcast delete event
	if file != nil {
		h.broadcastFileEvent(websocket.EventFileDeleted, map[string]interface{}{
			"id":        fileID,
			"name":      file.Name,
			"permanent": permanent,
		}, userID, file.ParentID)
	}

	return c.JSON(fiber.Map{
		"message": "File deleted successfully",
	})
}

// Restore recovers a file from trash
func (h *FileHandler) Restore(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	fileID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid file ID",
		})
	}

	file, err := h.fileService.Restore(c.Context(), fileID, userID)
	if err != nil {
		if err == repository.ErrFileNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "File not found",
			})
		}
		h.log.Error().Err(err).Msg("Failed to restore file")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to restore file",
		})
	}

	// Broadcast restored event
	h.broadcastFileEvent(websocket.EventFileRestored, file, userID, file.ParentID)

	return c.JSON(file)
}

// CopyRequest represents the copy payload
type CopyRequest struct {
	DestinationID *string `json:"destination_id"`
}

// Copy duplicates a file
func (h *FileHandler) Copy(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	fileID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid file ID",
		})
	}

	var req CopyRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	var destID *uuid.UUID
	if req.DestinationID != nil {
		id, err := uuid.Parse(*req.DestinationID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid destination_id",
			})
		}
		destID = &id
	}

	file, err := h.fileService.CopyFile(c.Context(), fileID, userID, destID, "")
	if err != nil {
		if err == repository.ErrFileNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "File not found",
			})
		}
		h.log.Error().Err(err).Msg("Failed to copy file")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to copy file",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(file)
}

// Download streams a file's content
func (h *FileHandler) Download(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	fileID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid file ID",
		})
	}

	reader, file, err := h.fileService.Download(c.Context(), fileID, userID)
	if err != nil {
		if err == repository.ErrFileNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "File not found",
			})
		}
		h.log.Error().Err(err).Msg("Failed to download file")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to download file",
		})
	}

	// Read the entire file into memory before the handler returns.
	// We cannot use defer reader.Close() + c.SendStream(reader) because
	// fasthttp reads the body stream AFTER the handler returns, but defer
	// would close the reader BEFORE that — resulting in truncated responses.
	data, readErr := io.ReadAll(reader)
	reader.Close()
	if readErr != nil {
		h.log.Error().Err(readErr).Msg("Failed to read file data")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to read file data",
		})
	}

	c.Set("Content-Type", file.MimeType)

	// Use inline disposition for previewable types, attachment for others
	disposition := "attachment"
	if isPreviewable(file.MimeType) {
		disposition = "inline"
	}
	// Sanitize filename to prevent header injection
	safeName := sanitizeFilename(file.Name)
	c.Set("Content-Disposition", disposition+"; filename=\""+safeName+"\"; filename*=UTF-8''"+url.PathEscape(file.Name))

	return c.Send(data)
}

// sanitizeFilename removes characters that could be used for header injection
func sanitizeFilename(name string) string {
	// Remove quotes, newlines, and carriage returns that could break headers
	name = strings.ReplaceAll(name, "\"", "")
	name = strings.ReplaceAll(name, "\n", "")
	name = strings.ReplaceAll(name, "\r", "")
	name = strings.ReplaceAll(name, "\x00", "")
	return name
}

// isPreviewable returns true if the mime type can be displayed inline in browsers
func isPreviewable(mimeType string) bool {
	switch {
	case len(mimeType) >= 6 && mimeType[:6] == "image/":
		return true
	case len(mimeType) >= 6 && mimeType[:6] == "video/":
		return true
	case len(mimeType) >= 6 && mimeType[:6] == "audio/":
		return true
	case mimeType == "application/pdf":
		return true
	case len(mimeType) >= 5 && mimeType[:5] == "text/":
		return true
	default:
		return false
	}
}

// GetVersions retrieves version history for a file
func (h *FileHandler) GetVersions(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	fileID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid file ID",
		})
	}

	versions, err := h.fileService.GetVersions(c.Context(), fileID, userID)
	if err != nil {
		if err == repository.ErrFileNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "File not found",
			})
		}
		h.log.Error().Err(err).Msg("Failed to get file versions")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get file versions",
		})
	}

	return c.JSON(fiber.Map{
		"versions": versions,
	})
}

// RestoreVersion restores a file to a specific version
func (h *FileHandler) RestoreVersion(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	fileID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid file ID",
		})
	}

	version, err := strconv.Atoi(c.Params("version"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid version number",
		})
	}

	file, err := h.fileService.RestoreVersion(c.Context(), fileID, userID, version)
	if err != nil {
		if err == repository.ErrFileNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "File or version not found",
			})
		}
		h.log.Error().Err(err).Msg("Failed to restore version")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to restore version",
		})
	}

	return c.JSON(file)
}

// Search finds files matching a query
func (h *FileHandler) Search(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	query := c.Query("q")
	if query == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Search query is required",
		})
	}

	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	files, err := h.fileService.Search(c.Context(), userID, query, limit)
	if err != nil {
		h.log.Error().Err(err).Msg("Search failed")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Search failed",
		})
	}

	return c.JSON(fiber.Map{
		"files": files,
		"query": query,
	})
}

// ListTrash retrieves all trashed files
func (h *FileHandler) ListTrash(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	files, err := h.fileService.ListTrash(c.Context(), userID)
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to list trash")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list trash",
		})
	}

	return c.JSON(fiber.Map{
		"files": files,
	})
}

// ListStarred retrieves all starred files
func (h *FileHandler) ListStarred(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	files, err := h.fileService.ListStarred(c.Context(), userID)
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to list starred files")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list starred files",
		})
	}

	return c.JSON(fiber.Map{
		"files": files,
	})
}

// EmptyTrash permanently deletes all trashed files
func (h *FileHandler) EmptyTrash(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	if err := h.fileService.EmptyTrash(c.Context(), userID); err != nil {
		h.log.Error().Err(err).Msg("Failed to empty trash")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to empty trash",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Trash emptied successfully",
	})
}

// StorageStats returns storage usage statistics
func (h *FileHandler) StorageStats(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	// TODO: Get actual storage limit from user
	storageLimit := int64(10 * 1024 * 1024 * 1024) // 10GB default

	stats, err := h.fileService.GetStorageStats(c.Context(), userID, storageLimit)
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to get storage stats")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get storage stats",
		})
	}

	return c.JSON(stats)
}

// Upload-related handlers (Tus protocol simplified implementation)

// InitiateUpload starts a new upload session
func (h *FileHandler) InitiateUpload(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	// Get upload metadata from headers (Tus protocol)
	fileName := c.Get("Upload-Metadata") // Simplified - would need base64 decode
	fileSize := c.Get("Upload-Length")

	size, err := strconv.ParseInt(fileSize, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid Upload-Length",
		})
	}

	// Generate upload ID
	uploadID := uuid.New().String()

	// TODO: Store upload session in Redis with metadata
	_ = userID
	_ = fileName
	_ = size

	c.Set("Location", "/api/upload/"+uploadID)
	c.Set("Tus-Resumable", "1.0.0")

	return c.SendStatus(fiber.StatusCreated)
}

// ChunkUpload handles file chunk uploads
func (h *FileHandler) ChunkUpload(c *fiber.Ctx) error {
	uploadID := c.Params("uploadId")
	_ = uploadID

	// TODO: Implement chunk handling
	// 1. Get upload session from Redis
	// 2. Validate offset
	// 3. Write chunk to temporary storage
	// 4. Update offset
	// 5. If complete, finalize upload

	c.Set("Tus-Resumable", "1.0.0")
	c.Set("Upload-Offset", "0") // Would be actual offset

	return c.SendStatus(fiber.StatusNoContent)
}

// UploadStatus returns the current upload offset
func (h *FileHandler) UploadStatus(c *fiber.Ctx) error {
	uploadID := c.Params("uploadId")
	_ = uploadID

	// TODO: Get upload session from Redis and return offset

	c.Set("Tus-Resumable", "1.0.0")
	c.Set("Upload-Offset", "0")
	c.Set("Upload-Length", "0")
	c.Set("Cache-Control", "no-store")

	return c.SendStatus(fiber.StatusOK)
}

// SimpleUpload handles simple file uploads (non-Tus)
func (h *FileHandler) SimpleUpload(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No file provided",
		})
	}

	var parentID *uuid.UUID
	if parentIDStr := c.FormValue("parent_id"); parentIDStr != "" {
		id, err := uuid.Parse(parentIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid parent_id",
			})
		}
		parentID = &id
	}

	// Open the file
	src, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to read file",
		})
	}
	defer src.Close()

	// Upload
	uploadedFile, err := h.fileService.UploadFile(c.Context(), services.UploadInput{
		OwnerID:  userID,
		ParentID: parentID,
		Name:     file.Filename,
		Size:     file.Size,
		Reader:   src,
	})

	if err != nil {
		if err == services.ErrQuotaExceeded {
			return c.Status(fiber.StatusPaymentRequired).JSON(fiber.Map{
				"error": "Storage quota exceeded",
			})
		}
		h.log.Error().Err(err).Msg("Upload failed")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Upload failed",
		})
	}

	// Broadcast file created event
	h.broadcastFileEvent(websocket.EventFileCreated, uploadedFile, userID, parentID)

	return c.Status(fiber.StatusCreated).JSON(uploadedFile)
}

// CreateShareRequest represents the share creation payload
type CreateShareRequest struct {
	ExpiresInDays *int    `json:"expires_in_days"`
	Password      *string `json:"password"`
	AllowDownload *bool   `json:"allow_download"`
	MaxDownloads  *int    `json:"max_downloads"`
}

// CreateShare creates a share link for a file
func (h *FileHandler) CreateShare(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	fileID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid file ID",
		})
	}

	var req CreateShareRequest
	if err := c.BodyParser(&req); err != nil {
		// Default values
		req = CreateShareRequest{}
	}

	share, err := h.fileService.CreateShare(c.Context(), services.CreateShareInput{
		FileID:        fileID,
		OwnerID:       userID,
		ExpiresInDays: req.ExpiresInDays,
		Password:      req.Password,
		AllowDownload: req.AllowDownload,
		MaxDownloads:  req.MaxDownloads,
	})

	if err != nil {
		h.log.Error().Err(err).Msg("Failed to create share")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create share",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(share)
}

// GetShareAnalytics returns analytics for a file's share link
func (h *FileHandler) GetShareAnalytics(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	fileID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid file ID",
		})
	}

	analytics, err := h.fileService.GetShareAnalytics(c.Context(), fileID, userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Share not found",
		})
	}

	return c.JSON(analytics)
}

// ShareWithUserRequest represents user sharing payload
type ShareWithUserRequest struct {
	Email      string `json:"email"`
	Permission string `json:"permission"`
}

// ShareWithUser shares a file with another user by email
func (h *FileHandler) ShareWithUser(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	fileID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid file ID",
		})
	}

	var req ShareWithUserRequest
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

	// Look up user by email (we need to add this to the service)
	targetUser, err := h.fileService.GetUserByEmail(c.Context(), req.Email)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	if targetUser.ID == userID {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot share with yourself",
		})
	}

	share, err := h.fileService.ShareWithUser(c.Context(), services.ShareWithUserInput{
		FileID:     fileID,
		OwnerID:    userID,
		SharedWith: targetUser.ID,
		Permission: req.Permission,
	})

	if err != nil {
		h.log.Error().Err(err).Msg("Failed to share with user")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to share",
		})
	}

	// Add user info to response
	share.SharedName = targetUser.Name
	share.SharedEmail = targetUser.Email

	// Broadcast share created event
	h.hub.BroadcastToUser(targetUser.ID, &websocket.Event{
		Type:      websocket.EventShareCreated,
		Payload:   share,
		UserID:    userID,
		Timestamp: time.Now().UnixMilli(),
	})

	return c.Status(fiber.StatusCreated).JSON(share)
}

// GetFileShares retrieves all shares for a file
func (h *FileHandler) GetFileShares(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	fileID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid file ID",
		})
	}

	shares, err := h.fileService.GetFileShares(c.Context(), fileID, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get shares",
		})
	}

	return c.JSON(fiber.Map{
		"shares": shares,
	})
}

// GetSharedWithMe retrieves files shared with the current user
func (h *FileHandler) GetSharedWithMe(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	files, err := h.fileService.GetSharedWithMe(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get shared files",
		})
	}

	return c.JSON(fiber.Map{
		"files": files,
	})
}

// RevokeShare removes a share
func (h *FileHandler) RevokeShare(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	shareID, err := uuid.Parse(c.Params("shareId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid share ID",
		})
	}

	if err := h.fileService.RevokeUserShare(c.Context(), shareID, userID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to revoke share",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Share revoked",
	})
}

// GetShare retrieves share info by token (public)
func (h *FileHandler) GetShare(c *fiber.Ctx) error {
	token := c.Params("token")

	share, err := h.fileService.GetShare(c.Context(), token)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Share not found or expired",
		})
	}

	return c.JSON(share)
}

// DownloadShare downloads a shared file (public)
func (h *FileHandler) DownloadShare(c *fiber.Ctx) error {
	token := c.Params("token")
	password := c.Query("password")

	reader, fileInfo, err := h.fileService.DownloadShare(c.Context(), token, password)
	if err != nil {
		if err.Error() == "password required" || err.Error() == "invalid password" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Share not found or expired",
		})
	}

	data, readErr := io.ReadAll(reader)
	reader.Close()
	if readErr != nil {
		h.log.Error().Err(readErr).Msg("Failed to read shared file data")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to read file data",
		})
	}

	safeName := sanitizeFilename(fileInfo.Name)
	c.Set("Content-Disposition", "attachment; filename=\""+safeName+"\"; filename*=UTF-8''"+url.PathEscape(fileInfo.Name))
	c.Set("Content-Type", fileInfo.MimeType)

	return c.Send(data)
}

// streamTokenClaims represents a short-lived token for file streaming
type streamTokenClaims struct {
	FileID string `json:"file_id"`
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

// StreamToken generates a short-lived token for streaming a file
// The token is passed as a query parameter so <video>/<audio> elements can access the file
func (h *FileHandler) StreamToken(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	fileID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid file ID",
		})
	}

	// Verify user has access to the file
	_, err = h.fileService.Get(c.Context(), fileID, userID)
	if err != nil {
		if err == repository.ErrFileNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "File not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get file",
		})
	}

	claims := streamTokenClaims{
		FileID: fileID.String(),
		UserID: userID.String(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "tessera-stream",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(h.jwtSecret))
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to generate stream token")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate stream token",
		})
	}

	return c.JSON(fiber.Map{
		"token": tokenString,
	})
}

// StreamFile serves a file with HTTP Range request support for streaming playback.
// Auth is via a short-lived token in the query string (so <video src="..."> works).
//
// IMPORTANT: We use c.Context().SetBodyStreamWriter() instead of c.SendStream()
// because fasthttp reads the body stream AFTER the handler returns. If we used
// defer reader.Close() + SendStream(), the reader would be closed before fasthttp
// reads from it, producing empty/truncated responses. SetBodyStreamWriter gives us
// a callback that runs during response writing, so the reader stays open.
func (h *FileHandler) StreamFile(c *fiber.Ctx) error {
	tokenString := c.Query("token")
	if tokenString == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Missing stream token",
		})
	}

	// Validate stream token
	claims := &streamTokenClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(h.jwtSecret), nil
	})
	if err != nil || !token.Valid {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid or expired stream token",
		})
	}

	fileID, err := uuid.Parse(c.Params("id"))
	if err != nil || fileID.String() != claims.FileID {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid file ID",
		})
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid token",
		})
	}

	// Get file metadata
	file, err := h.fileService.Get(c.Context(), fileID, userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "File not found",
		})
	}

	// Get object info for Content-Length
	objInfo, err := h.storage.Stat(c.Context(), file.StorageKey)
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to stat file object")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to read file",
		})
	}
	totalSize := objInfo.Size

	// Set common headers
	safeName := sanitizeFilename(file.Name)
	c.Set("Content-Type", file.MimeType)
	c.Set("Accept-Ranges", "bytes")
	c.Set("Content-Disposition", "inline; filename=\""+safeName+"\"; filename*=UTF-8''"+url.PathEscape(file.Name))
	c.Set("Cache-Control", "private, max-age=3600")

	// Parse Range header
	rangeHeader := c.Get("Range")
	if rangeHeader == "" {
		// No range — serve the entire file using stream writer
		c.Set("Content-Length", strconv.FormatInt(totalSize, 10))
		c.Status(fiber.StatusOK)

		storageKey := file.StorageKey
		c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
			reader, err := h.storage.Download(context.Background(), storageKey)
			if err != nil {
				h.log.Error().Err(err).Msg("Stream: failed to download file")
				return
			}
			defer reader.Close()

			if _, err := io.Copy(w, reader); err != nil {
				h.log.Error().Err(err).Msg("Stream: failed to write file to response")
			}
			w.Flush()
		})
		return nil
	}

	// Parse "bytes=start-end"
	rangeHeader = strings.TrimPrefix(rangeHeader, "bytes=")
	parts := strings.SplitN(rangeHeader, "-", 2)
	if len(parts) != 2 {
		c.Set("Content-Range", fmt.Sprintf("bytes */%d", totalSize))
		return c.Status(fiber.StatusRequestedRangeNotSatisfiable).SendString("Invalid range")
	}

	var start, end int64
	if parts[0] == "" {
		// Suffix range: bytes=-500 → last 500 bytes
		suffixLen, parseErr := strconv.ParseInt(parts[1], 10, 64)
		if parseErr != nil || suffixLen <= 0 {
			c.Set("Content-Range", fmt.Sprintf("bytes */%d", totalSize))
			return c.Status(fiber.StatusRequestedRangeNotSatisfiable).SendString("Invalid range")
		}
		start = totalSize - suffixLen
		if start < 0 {
			start = 0
		}
		end = totalSize - 1
	} else {
		var parseErr error
		start, parseErr = strconv.ParseInt(parts[0], 10, 64)
		if parseErr != nil || start < 0 {
			c.Set("Content-Range", fmt.Sprintf("bytes */%d", totalSize))
			return c.Status(fiber.StatusRequestedRangeNotSatisfiable).SendString("Invalid range")
		}
		if parts[1] == "" {
			end = totalSize - 1
		} else {
			end, parseErr = strconv.ParseInt(parts[1], 10, 64)
			if parseErr != nil || end < start {
				c.Set("Content-Range", fmt.Sprintf("bytes */%d", totalSize))
				return c.Status(fiber.StatusRequestedRangeNotSatisfiable).SendString("Invalid range")
			}
		}
	}

	if start >= totalSize {
		c.Set("Content-Range", fmt.Sprintf("bytes */%d", totalSize))
		return c.Status(fiber.StatusRequestedRangeNotSatisfiable).SendString("Range not satisfiable")
	}
	if end >= totalSize {
		end = totalSize - 1
	}

	contentLen := end - start + 1

	c.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, totalSize))
	c.Set("Content-Length", strconv.FormatInt(contentLen, 10))
	c.Status(fiber.StatusPartialContent)

	storageKey := file.StorageKey
	rangeStart := start
	rangeLen := contentLen
	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		reader, err := h.storage.Download(context.Background(), storageKey)
		if err != nil {
			h.log.Error().Err(err).Msg("Stream range: failed to download file")
			return
		}
		defer reader.Close()

		// Seek to start position
		if rangeStart > 0 {
			if seeker, ok := reader.(io.Seeker); ok {
				if _, err := seeker.Seek(rangeStart, io.SeekStart); err != nil {
					h.log.Error().Err(err).Msg("Stream range: failed to seek")
					return
				}
			} else {
				// Fallback: discard bytes to reach start
				if _, err := io.CopyN(io.Discard, reader, rangeStart); err != nil {
					h.log.Error().Err(err).Msg("Stream range: failed to skip to start")
					return
				}
			}
		}

		// Copy only the requested range
		if _, err := io.CopyN(w, reader, rangeLen); err != nil && err != io.EOF {
			h.log.Error().Err(err).Msg("Stream range: failed to write range to response")
		}
		w.Flush()
	})
	return nil
}

// broadcastFileEvent sends a WebSocket event to subscribers
func (h *FileHandler) broadcastFileEvent(eventType websocket.EventType, payload interface{}, userID uuid.UUID, folderID *uuid.UUID) {
	if h.hub == nil {
		return
	}

	event := &websocket.Event{
		Type:      eventType,
		Payload:   payload,
		UserID:    userID,
		Timestamp: time.Now().UnixMilli(),
	}

	if folderID != nil {
		event.FolderID = folderID
		h.hub.BroadcastToFolder(*folderID, event, nil)
	} else {
		// Root folder - use nil UUID
		rootID := uuid.Nil
		h.hub.BroadcastToFolder(rootID, event, nil)
	}

	// Also broadcast to the user for their activity feed
	h.hub.BroadcastToUser(userID, event)
}
