package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tessera/tessera/internal/models"
)

// DocumentRepository handles document database operations
type DocumentRepository struct {
	db *pgxpool.Pool
}

// NewDocumentRepository creates a new document repository
func NewDocumentRepository(db *pgxpool.Pool) *DocumentRepository {
	return &DocumentRepository{db: db}
}

// ListByOwner returns all documents owned by a user
func (r *DocumentRepository) ListByOwner(ctx context.Context, ownerID uuid.UUID) ([]models.Document, error) {
	query := `
		SELECT d.id, d.file_id, d.owner_id, d.title, d.content, d.format, d.is_public,
		       d.version, d.created_at, d.updated_at, COALESCE(u.name, 'User') as owner_name
		FROM documents d
		LEFT JOIN users u ON d.owner_id = u.id
		WHERE d.owner_id = $1
		ORDER BY d.updated_at DESC
	`

	rows, err := r.db.Query(ctx, query, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []models.Document
	for rows.Next() {
		var d models.Document
		if err := rows.Scan(
			&d.ID, &d.FileID, &d.OwnerID, &d.Title, &d.Content, &d.Format,
			&d.IsPublic, &d.Version, &d.CreatedAt, &d.UpdatedAt, &d.OwnerName,
		); err != nil {
			return nil, err
		}
		// Load collaborators
		collabs, _ := r.getCollaborators(ctx, d.ID)
		d.Collaborators = collabs
		docs = append(docs, d)
	}

	if docs == nil {
		docs = []models.Document{}
	}
	return docs, nil
}

// GetByID returns a document by ID (regardless of owner, for collaborator access)
func (r *DocumentRepository) GetByID(ctx context.Context, docID uuid.UUID) (*models.Document, error) {
	query := `
		SELECT d.id, d.file_id, d.owner_id, d.title, d.content, d.format, d.is_public,
		       d.version, d.created_at, d.updated_at, COALESCE(u.name, 'User') as owner_name
		FROM documents d
		LEFT JOIN users u ON d.owner_id = u.id
		WHERE d.id = $1
	`

	var d models.Document
	err := r.db.QueryRow(ctx, query, docID).Scan(
		&d.ID, &d.FileID, &d.OwnerID, &d.Title, &d.Content, &d.Format,
		&d.IsPublic, &d.Version, &d.CreatedAt, &d.UpdatedAt, &d.OwnerName,
	)
	if err != nil {
		return nil, err
	}

	collabs, _ := r.getCollaborators(ctx, d.ID)
	d.Collaborators = collabs
	return &d, nil
}

// Create inserts a new document
func (r *DocumentRepository) Create(ctx context.Context, doc *models.Document) error {
	query := `
		INSERT INTO documents (id, file_id, owner_id, title, content, format, is_public, version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := r.db.Exec(ctx, query,
		doc.ID, doc.FileID, doc.OwnerID, doc.Title, doc.Content, doc.Format,
		doc.IsPublic, doc.Version, doc.CreatedAt, doc.UpdatedAt,
	)
	return err
}

// Update updates a document
func (r *DocumentRepository) Update(ctx context.Context, doc *models.Document) error {
	query := `
		UPDATE documents SET
			title = $2, content = $3, format = $4, is_public = $5, version = $6, updated_at = $7
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query,
		doc.ID, doc.Title, doc.Content, doc.Format, doc.IsPublic, doc.Version, doc.UpdatedAt,
	)
	return err
}

// Delete deletes a document owned by a user
func (r *DocumentRepository) Delete(ctx context.Context, docID, ownerID uuid.UUID) error {
	_, err := r.db.Exec(ctx, "DELETE FROM documents WHERE id = $1 AND owner_id = $2", docID, ownerID)
	return err
}

// AddCollaborator adds a collaborator to a document
func (r *DocumentRepository) AddCollaborator(ctx context.Context, docID, userID uuid.UUID, permission, color string) error {
	query := `
		INSERT INTO document_collaborators (id, document_id, user_id, permission, color, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (document_id, user_id) DO UPDATE SET permission = $4
	`
	_, err := r.db.Exec(ctx, query, uuid.New(), docID, userID, permission, color)
	return err
}

// RemoveCollaborator removes a collaborator from a document
func (r *DocumentRepository) RemoveCollaborator(ctx context.Context, docID, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		"DELETE FROM document_collaborators WHERE document_id = $1 AND user_id = $2",
		docID, userID,
	)
	return err
}

// IsCollaborator checks if a user is a collaborator on a document
func (r *DocumentRepository) IsCollaborator(ctx context.Context, docID, userID uuid.UUID) (bool, error) {
	var count int
	err := r.db.QueryRow(ctx,
		"SELECT COUNT(*) FROM document_collaborators WHERE document_id = $1 AND user_id = $2",
		docID, userID,
	).Scan(&count)
	return count > 0, err
}

// getCollaborators loads all collaborators for a document
func (r *DocumentRepository) getCollaborators(ctx context.Context, docID uuid.UUID) ([]models.DocumentCollaborator, error) {
	query := `
		SELECT dc.user_id, COALESCE(u.name, '') as user_name, COALESCE(u.email, '') as user_email,
		       dc.permission, dc.color
		FROM document_collaborators dc
		LEFT JOIN users u ON dc.user_id = u.id
		WHERE dc.document_id = $1
	`

	rows, err := r.db.Query(ctx, query, docID)
	if err != nil {
		return []models.DocumentCollaborator{}, nil
	}
	defer rows.Close()

	var collabs []models.DocumentCollaborator
	for rows.Next() {
		var c models.DocumentCollaborator
		if err := rows.Scan(&c.UserID, &c.UserName, &c.UserEmail, &c.Permission, &c.Color); err != nil {
			continue
		}
		c.Online = false // Real-time status managed by WebSocket
		collabs = append(collabs, c)
	}

	if collabs == nil {
		collabs = []models.DocumentCollaborator{}
	}
	return collabs, nil
}
