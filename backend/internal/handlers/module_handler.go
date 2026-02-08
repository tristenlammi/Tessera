package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"github.com/tessera/tessera/internal/repository"
)

// ModuleHandler handles module configuration endpoints
type ModuleHandler struct {
	log          zerolog.Logger
	settingsRepo *repository.SettingsRepository
}

// NewModuleHandler creates a new module handler
func NewModuleHandler(log zerolog.Logger, settingsRepo *repository.SettingsRepository) *ModuleHandler {
	return &ModuleHandler{
		log:          log,
		settingsRepo: settingsRepo,
	}
}

// GetModules returns the current module settings (for regular users)
func (h *ModuleHandler) GetModules(c *fiber.Ctx) error {
	modules, err := h.settingsRepo.GetModules(c.Context())
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to get module settings")
		// Return defaults on error
		modules = map[string]bool{
			"documents": false,
			"pdf":       false,
			"tasks":     false,
			"calendar":  false,
			"contacts":  false,
			"email":     false,
		}
	}

	return c.JSON(fiber.Map{
		"modules": modules,
	})
}

// GetAdminModules returns module settings for admin
func (h *ModuleHandler) GetAdminModules(c *fiber.Ctx) error {
	modules, err := h.settingsRepo.GetModules(c.Context())
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to get module settings")
		modules = map[string]bool{
			"documents": false,
			"pdf":       false,
			"tasks":     false,
			"calendar":  false,
			"contacts":  false,
			"email":     false,
		}
	}

	return c.JSON(fiber.Map{
		"modules": modules,
	})
}

// UpdateModule updates a single module setting
func (h *ModuleHandler) UpdateModule(c *fiber.Ctx) error {
	moduleID := c.Params("id")

	var req struct {
		Enabled bool `json:"enabled"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Get current modules
	modules, err := h.settingsRepo.GetModules(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get module settings",
		})
	}

	// Check if module exists
	if _, exists := modules[moduleID]; !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Module not found",
		})
	}

	modules[moduleID] = req.Enabled

	// Save to database
	if err := h.settingsRepo.SetModules(c.Context(), modules); err != nil {
		h.log.Error().Err(err).Msg("Failed to save module settings")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save module settings",
		})
	}

	h.log.Info().
		Str("module", moduleID).
		Bool("enabled", req.Enabled).
		Msg("Module setting updated")

	return c.JSON(fiber.Map{
		"success": true,
		"module":  moduleID,
		"enabled": req.Enabled,
	})
}

// UpdateAllModules updates all module settings at once
func (h *ModuleHandler) UpdateAllModules(c *fiber.Ctx) error {
	var req struct {
		Modules map[string]bool `json:"modules"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Get current modules to preserve structure
	modules, _ := h.settingsRepo.GetModules(c.Context())

	for moduleID, enabled := range req.Modules {
		if _, exists := modules[moduleID]; exists {
			modules[moduleID] = enabled
		}
	}

	// Save to database
	if err := h.settingsRepo.SetModules(c.Context(), modules); err != nil {
		h.log.Error().Err(err).Msg("Failed to save module settings")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save module settings",
		})
	}

	h.log.Info().Msg("All module settings updated")

	return c.JSON(fiber.Map{
		"success": true,
		"modules": modules,
	})
}
