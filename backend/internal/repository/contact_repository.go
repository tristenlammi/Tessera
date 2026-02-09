package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tessera/tessera/internal/models"
)

// ContactRepository handles contact database operations
type ContactRepository struct {
	db *pgxpool.Pool
}

// NewContactRepository creates a new contact repository
func NewContactRepository(db *pgxpool.Pool) *ContactRepository {
	return &ContactRepository{db: db}
}

// ListByUser returns all contacts for a user
func (r *ContactRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]models.Contact, error) {
	query := `
		SELECT id, user_id, first_name, last_name, email, phone, company, job_title,
		       birthday, notes, avatar, favorite, created_at, updated_at
		FROM contacts
		WHERE user_id = $1
		ORDER BY first_name ASC, last_name ASC
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contacts []models.Contact
	for rows.Next() {
		var c models.Contact
		if err := rows.Scan(
			&c.ID, &c.UserID, &c.FirstName, &c.LastName, &c.Email, &c.Phone,
			&c.Company, &c.JobTitle, &c.Birthday, &c.Notes, &c.Avatar,
			&c.Favorite, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		contacts = append(contacts, c)
	}

	if contacts == nil {
		contacts = []models.Contact{}
	}
	return contacts, nil
}

// GetByID returns a single contact
func (r *ContactRepository) GetByID(ctx context.Context, contactID, userID uuid.UUID) (*models.Contact, error) {
	query := `
		SELECT id, user_id, first_name, last_name, email, phone, company, job_title,
		       birthday, notes, avatar, favorite, created_at, updated_at
		FROM contacts
		WHERE id = $1 AND user_id = $2
	`

	var c models.Contact
	err := r.db.QueryRow(ctx, query, contactID, userID).Scan(
		&c.ID, &c.UserID, &c.FirstName, &c.LastName, &c.Email, &c.Phone,
		&c.Company, &c.JobTitle, &c.Birthday, &c.Notes, &c.Avatar,
		&c.Favorite, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// Create inserts a new contact
func (r *ContactRepository) Create(ctx context.Context, contact *models.Contact) error {
	query := `
		INSERT INTO contacts (id, user_id, first_name, last_name, email, phone, company, job_title,
		                       birthday, notes, avatar, favorite, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	_, err := r.db.Exec(ctx, query,
		contact.ID, contact.UserID, contact.FirstName, contact.LastName, contact.Email,
		contact.Phone, contact.Company, contact.JobTitle, contact.Birthday, contact.Notes,
		contact.Avatar, contact.Favorite, contact.CreatedAt, contact.UpdatedAt,
	)
	return err
}

// Update updates a contact
func (r *ContactRepository) Update(ctx context.Context, contact *models.Contact) error {
	query := `
		UPDATE contacts SET
			first_name = $3, last_name = $4, email = $5, phone = $6, company = $7,
			job_title = $8, birthday = $9, notes = $10, avatar = $11, favorite = $12,
			updated_at = $13
		WHERE id = $1 AND user_id = $2
	`

	_, err := r.db.Exec(ctx, query,
		contact.ID, contact.UserID, contact.FirstName, contact.LastName, contact.Email,
		contact.Phone, contact.Company, contact.JobTitle, contact.Birthday, contact.Notes,
		contact.Avatar, contact.Favorite, contact.UpdatedAt,
	)
	return err
}

// ToggleFavorite toggles the favorite status
func (r *ContactRepository) ToggleFavorite(ctx context.Context, contactID, userID uuid.UUID, favorite bool) error {
	_, err := r.db.Exec(ctx,
		"UPDATE contacts SET favorite = $3, updated_at = $4 WHERE id = $1 AND user_id = $2",
		contactID, userID, favorite, time.Now(),
	)
	return err
}

// Delete deletes a contact
func (r *ContactRepository) Delete(ctx context.Context, contactID, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx, "DELETE FROM contacts WHERE id = $1 AND user_id = $2", contactID, userID)
	return err
}
