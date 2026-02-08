package handlers

import (
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type ContactsHandler struct {
	log zerolog.Logger
}

func NewContactsHandler(log zerolog.Logger) *ContactsHandler {
	return &ContactsHandler{log: log}
}

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

// In-memory storage for contacts (per user)
var (
	contactsList = make(map[uuid.UUID][]Contact)
	contactsMu   sync.RWMutex
)

// RegisterRoutes registers contacts routes
func (h *ContactsHandler) RegisterRoutes(app *fiber.App, authMiddleware fiber.Handler) {
	contacts := app.Group("/api/contacts", authMiddleware)

	contacts.Get("/", h.ListContacts)
	contacts.Post("/", h.CreateContact)
	contacts.Get("/:id", h.GetContact)
	contacts.Put("/:id", h.UpdateContact)
	contacts.Patch("/:id/favorite", h.ToggleFavorite)
	contacts.Delete("/:id", h.DeleteContact)
}

// ListContacts returns all contacts for a user
func (h *ContactsHandler) ListContacts(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	contactsMu.RLock()
	userContacts := contactsList[userID]
	contactsMu.RUnlock()

	if userContacts == nil {
		userContacts = []Contact{}
	}

	return c.JSON(userContacts)
}

// CreateContact creates a new contact
func (h *ContactsHandler) CreateContact(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	var input struct {
		FirstName string     `json:"firstName"`
		LastName  string     `json:"lastName"`
		Email     string     `json:"email"`
		Phone     string     `json:"phone"`
		Company   string     `json:"company"`
		JobTitle  string     `json:"jobTitle"`
		Birthday  *time.Time `json:"birthday"`
		Notes     string     `json:"notes"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if input.FirstName == "" && input.LastName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "First name or last name is required"})
	}

	now := time.Now()
	contact := Contact{
		ID:        uuid.New(),
		UserID:    userID,
		FirstName: input.FirstName,
		LastName:  input.LastName,
		Email:     input.Email,
		Phone:     input.Phone,
		Company:   input.Company,
		JobTitle:  input.JobTitle,
		Birthday:  input.Birthday,
		Notes:     input.Notes,
		Favorite:  false,
		CreatedAt: now,
		UpdatedAt: now,
	}

	contactsMu.Lock()
	contactsList[userID] = append(contactsList[userID], contact)
	contactsMu.Unlock()

	h.log.Info().
		Str("contact_id", contact.ID.String()).
		Str("name", contact.FirstName+" "+contact.LastName).
		Msg("Contact created")

	return c.Status(fiber.StatusCreated).JSON(contact)
}

// GetContact returns a specific contact
func (h *ContactsHandler) GetContact(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	contactID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid contact ID"})
	}

	contactsMu.RLock()
	userContacts := contactsList[userID]
	contactsMu.RUnlock()

	for _, contact := range userContacts {
		if contact.ID == contactID {
			return c.JSON(contact)
		}
	}

	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Contact not found"})
}

// UpdateContact updates an existing contact
func (h *ContactsHandler) UpdateContact(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	contactID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid contact ID"})
	}

	var input struct {
		FirstName string     `json:"firstName"`
		LastName  string     `json:"lastName"`
		Email     string     `json:"email"`
		Phone     string     `json:"phone"`
		Company   string     `json:"company"`
		JobTitle  string     `json:"jobTitle"`
		Birthday  *time.Time `json:"birthday"`
		Notes     string     `json:"notes"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	contactsMu.Lock()
	defer contactsMu.Unlock()

	userContacts := contactsList[userID]
	for i, contact := range userContacts {
		if contact.ID == contactID {
			userContacts[i].FirstName = input.FirstName
			userContacts[i].LastName = input.LastName
			userContacts[i].Email = input.Email
			userContacts[i].Phone = input.Phone
			userContacts[i].Company = input.Company
			userContacts[i].JobTitle = input.JobTitle
			userContacts[i].Birthday = input.Birthday
			userContacts[i].Notes = input.Notes
			userContacts[i].UpdatedAt = time.Now()

			contactsList[userID] = userContacts

			h.log.Info().
				Str("contact_id", contactID.String()).
				Msg("Contact updated")

			return c.JSON(userContacts[i])
		}
	}

	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Contact not found"})
}

// ToggleFavorite toggles the favorite status of a contact
func (h *ContactsHandler) ToggleFavorite(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	contactID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid contact ID"})
	}

	var input struct {
		Favorite bool `json:"favorite"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	contactsMu.Lock()
	defer contactsMu.Unlock()

	userContacts := contactsList[userID]
	for i, contact := range userContacts {
		if contact.ID == contactID {
			userContacts[i].Favorite = input.Favorite
			userContacts[i].UpdatedAt = time.Now()

			contactsList[userID] = userContacts

			return c.JSON(userContacts[i])
		}
	}

	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Contact not found"})
}

// DeleteContact deletes a contact
func (h *ContactsHandler) DeleteContact(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	contactID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid contact ID"})
	}

	contactsMu.Lock()
	defer contactsMu.Unlock()

	userContacts := contactsList[userID]
	for i, contact := range userContacts {
		if contact.ID == contactID {
			contactsList[userID] = append(userContacts[:i], userContacts[i+1:]...)

			h.log.Info().
				Str("contact_id", contactID.String()).
				Msg("Contact deleted")

			return c.SendStatus(fiber.StatusNoContent)
		}
	}

	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Contact not found"})
}
