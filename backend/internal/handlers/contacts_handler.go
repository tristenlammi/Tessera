package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/tessera/tessera/internal/models"
	"github.com/tessera/tessera/internal/repository"
)

type ContactsHandler struct {
	log         zerolog.Logger
	contactRepo *repository.ContactRepository
}

func NewContactsHandler(log zerolog.Logger, contactRepo *repository.ContactRepository) *ContactsHandler {
	return &ContactsHandler{
		log:         log,
		contactRepo: contactRepo,
	}
}

// ListContacts returns all contacts for a user
func (h *ContactsHandler) ListContacts(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	contacts, err := h.contactRepo.ListByUser(c.Context(), userID)
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to list contacts")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch contacts"})
	}

	return c.JSON(contacts)
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
	contact := &models.Contact{
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

	if err := h.contactRepo.Create(c.Context(), contact); err != nil {
		h.log.Error().Err(err).Msg("Failed to create contact")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create contact"})
	}

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

	contact, err := h.contactRepo.GetByID(c.Context(), contactID, userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Contact not found"})
	}

	return c.JSON(contact)
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

	contact, err := h.contactRepo.GetByID(c.Context(), contactID, userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Contact not found"})
	}

	contact.FirstName = input.FirstName
	contact.LastName = input.LastName
	contact.Email = input.Email
	contact.Phone = input.Phone
	contact.Company = input.Company
	contact.JobTitle = input.JobTitle
	contact.Birthday = input.Birthday
	contact.Notes = input.Notes
	contact.UpdatedAt = time.Now()

	if err := h.contactRepo.Update(c.Context(), contact); err != nil {
		h.log.Error().Err(err).Msg("Failed to update contact")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update contact"})
	}

	h.log.Info().
		Str("contact_id", contactID.String()).
		Msg("Contact updated")

	return c.JSON(contact)
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

	if err := h.contactRepo.ToggleFavorite(c.Context(), contactID, userID, input.Favorite); err != nil {
		h.log.Error().Err(err).Msg("Failed to toggle favorite")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update contact"})
	}

	contact, err := h.contactRepo.GetByID(c.Context(), contactID, userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Contact not found"})
	}

	return c.JSON(contact)
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

	if err := h.contactRepo.Delete(c.Context(), contactID, userID); err != nil {
		h.log.Error().Err(err).Msg("Failed to delete contact")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete contact"})
	}

	h.log.Info().
		Str("contact_id", contactID.String()).
		Msg("Contact deleted")

	return c.SendStatus(fiber.StatusNoContent)
}
