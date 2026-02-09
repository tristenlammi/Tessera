package models

import (
	"time"

	"github.com/google/uuid"
)

// Contact represents a contact entry
type Contact struct {
	ID        uuid.UUID  `json:"id"`
	UserID    uuid.UUID  `json:"userId"`
	FirstName string     `json:"firstName"`
	LastName  string     `json:"lastName"`
	Email     string     `json:"email"`
	Phone     string     `json:"phone"`
	Company   string     `json:"company"`
	JobTitle  string     `json:"jobTitle"`
	Birthday  *time.Time `json:"birthday,omitempty"`
	Notes     string     `json:"notes"`
	Avatar    *string    `json:"avatar,omitempty"`
	Favorite  bool       `json:"favorite"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
}
