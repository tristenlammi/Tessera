package handlers

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/bcrypt"

	"github.com/tessera/tessera/internal/middleware"
	"github.com/tessera/tessera/internal/services"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	authService *services.AuthService
	log         zerolog.Logger
	db          *pgxpool.Pool
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authService *services.AuthService, log zerolog.Logger, db *pgxpool.Pool) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		log:         log,
		db:          db,
	}
}

// RegisterRequest represents the registration payload
type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	Name     string `json:"name" validate:"required,min=2"`
}

// Register handles user registration
func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	user, err := h.authService.Register(c.Context(), services.RegisterInput{
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
	})

	if err != nil {
		if err == services.ErrEmailTaken {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "Email already registered",
			})
		}
		h.log.Error().Err(err).Msg("Registration failed")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create account",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Account created successfully",
		"user": fiber.Map{
			"id":    user.ID,
			"email": user.Email,
			"name":  user.Name,
		},
	})
}

// LoginRequest represents the login payload
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
	TOTPCode string `json:"totp_code"` // Optional - required if 2FA enabled
}

// Login handles user authentication
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	user, tokens, pendingAuth, err := h.authService.Login(c.Context(), services.LoginInput{
		Email:     req.Email,
		Password:  req.Password,
		TOTPCode:  req.TOTPCode,
		UserAgent: c.Get("User-Agent"),
		IPAddress: c.IP(),
	})

	if err != nil {
		if err == services.ErrInvalidCredentials {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid email or password",
			})
		}
		if err == services.ErrUserNotActive {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Account is not active",
			})
		}
		if err == services.ErrTOTPRequired {
			// Return pending auth token instead of requiring password re-send
			// Note: We intentionally don't return user ID or email to avoid leaking account existence
			return c.Status(fiber.StatusPreconditionRequired).JSON(fiber.Map{
				"error":              "2FA code required",
				"totp_required":      true,
				"pending_auth_token": pendingAuth.Token,
			})
		}
		if err == services.ErrInvalidTOTPCode {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid 2FA code",
			})
		}
		h.log.Error().Err(err).Msg("Login failed")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Login failed",
		})
	}

	return c.JSON(fiber.Map{
		"user": fiber.Map{
			"id":            user.ID,
			"email":         user.Email,
			"name":          user.Name,
			"role":          user.Role,
			"timezone":      user.Timezone,
			"storage_used":  user.StorageUsed,
			"storage_limit": user.StorageLimit,
			"totp_enabled":  user.TOTPEnabled,
		},
		"tokens": tokens,
	})
}

// CompleteTOTPLoginRequest represents the 2FA completion payload
type CompleteTOTPLoginRequest struct {
	PendingAuthToken string `json:"pending_auth_token" validate:"required"`
	TOTPCode         string `json:"totp_code" validate:"required"`
}

// CompleteTOTPLogin completes 2FA login using a pending auth token
func (h *AuthHandler) CompleteTOTPLogin(c *fiber.Ctx) error {
	var req CompleteTOTPLoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	user, tokens, err := h.authService.CompleteTOTPLogin(c.Context(), services.CompleteTOTPLoginInput{
		PendingToken: req.PendingAuthToken,
		TOTPCode:     req.TOTPCode,
		UserAgent:    c.Get("User-Agent"),
		IPAddress:    c.IP(),
	})

	if err != nil {
		if err == services.ErrInvalidToken {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid or expired pending auth token",
			})
		}
		if err == services.ErrInvalidTOTPCode {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid 2FA code",
			})
		}
		if err == services.ErrUserNotActive {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Account is not active",
			})
		}
		h.log.Error().Err(err).Msg("TOTP login completion failed")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Login failed",
		})
	}

	return c.JSON(fiber.Map{
		"user": fiber.Map{
			"id":            user.ID,
			"email":         user.Email,
			"name":          user.Name,
			"role":          user.Role,
			"timezone":      user.Timezone,
			"storage_used":  user.StorageUsed,
			"storage_limit": user.StorageLimit,
			"totp_enabled":  user.TOTPEnabled,
		},
		"tokens": tokens,
	})
}

// RefreshRequest represents the token refresh payload
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// RefreshToken handles token refresh
func (h *AuthHandler) RefreshToken(c *fiber.Ctx) error {
	var req RefreshRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	tokens, err := h.authService.RefreshTokens(c.Context(), req.RefreshToken)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid or expired refresh token",
		})
	}

	return c.JSON(fiber.Map{
		"tokens": tokens,
	})
}

// Logout handles user logout
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	sessionID := middleware.GetSessionID(c)

	if err := h.authService.Logout(c.Context(), sessionID); err != nil {
		h.log.Error().Err(err).Msg("Logout failed")
	}

	return c.JSON(fiber.Map{
		"message": "Logged out successfully",
	})
}

// Me returns the current user's profile
func (h *AuthHandler) Me(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	user, err := h.authService.GetUserByID(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	return c.JSON(fiber.Map{
		"id":            user.ID,
		"email":         user.Email,
		"name":          user.Name,
		"role":          user.Role,
		"timezone":      user.Timezone,
		"storage_used":  user.StorageUsed,
		"storage_limit": user.StorageLimit,
		"created_at":    user.CreatedAt,
	})
}

// UpdateSettingsRequest represents the user settings update payload
type UpdateSettingsRequest struct {
	Timezone string `json:"timezone"`
}

// UpdateSettings handles user settings updates
func (h *AuthHandler) UpdateSettings(c *fiber.Ctx) error {
	var req UpdateSettingsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	userID := middleware.GetUserID(c)

	// Validate timezone by trying to load it
	if req.Timezone != "" {
		if _, err := time.LoadLocation(req.Timezone); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid timezone",
			})
		}

		if err := h.authService.UpdateTimezone(c.Context(), userID, req.Timezone); err != nil {
			h.log.Error().Err(err).Msg("Failed to update timezone")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to update settings",
			})
		}
	}

	// Fetch updated user to return
	user, err := h.authService.GetUserByID(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	return c.JSON(fiber.Map{
		"id":            user.ID,
		"email":         user.Email,
		"name":          user.Name,
		"role":          user.Role,
		"timezone":      user.Timezone,
		"storage_used":  user.StorageUsed,
		"storage_limit": user.StorageLimit,
		"created_at":    user.CreatedAt,
	})
}

// ChangePasswordRequest represents the password change payload
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8"`
}

// ChangePassword handles password changes
func (h *AuthHandler) ChangePassword(c *fiber.Ctx) error {
	var req ChangePasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	userID := middleware.GetUserID(c)

	if err := h.authService.ChangePassword(c.Context(), userID, req.OldPassword, req.NewPassword); err != nil {
		if err == services.ErrInvalidCredentials {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Current password is incorrect",
			})
		}
		h.log.Error().Err(err).Msg("Password change failed")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to change password",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Password changed successfully. Please login again.",
	})
}

// GetWebSocketTicket creates a short-lived ticket for WebSocket authentication
// This avoids exposing the JWT in the WebSocket URL
func (h *AuthHandler) GetWebSocketTicket(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	ticket, err := h.authService.CreateWebSocketTicket(c.Context(), userID)
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to create WebSocket ticket")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create ticket",
		})
	}

	return c.JSON(fiber.Map{
		"ticket": ticket,
	})
}

// ForgotPasswordRequest represents the forgot password payload
type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// ForgotPassword directs user to contact admin (homelab apps don't need email-based reset)
func (h *AuthHandler) ForgotPassword(c *fiber.Ctx) error {
	var req ForgotPasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// In a homelab deployment, password reset is admin-generated.
	// Return a helpful message without leaking user existence.
	return c.JSON(fiber.Map{
		"message": "Please contact your administrator for a password reset link.",
	})
}

// ResetPasswordRequest represents the password reset payload
type ResetPasswordRequest struct {
	Token       string `json:"token" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8"`
}

// ResetPassword completes password reset using an admin-generated token
func (h *AuthHandler) ResetPassword(c *fiber.Ctx) error {
	var req ResetPasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Token == "" || req.NewPassword == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Token and new password are required",
		})
	}

	if len(req.NewPassword) < 8 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Password must be at least 8 characters",
		})
	}

	ctx, cancel := context.WithTimeout(c.Context(), 10*time.Second)
	defer cancel()

	// Look up the token
	var tokenID, userID uuid.UUID
	var expiresAt time.Time
	var usedAt *time.Time
	err := h.db.QueryRow(ctx,
		`SELECT id, user_id, expires_at, used_at FROM password_reset_tokens WHERE token = $1`,
		req.Token,
	).Scan(&tokenID, &userID, &expiresAt, &usedAt)

	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid or expired reset link",
		})
	}

	// Check if already used
	if usedAt != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "This reset link has already been used",
		})
	}

	// Check if expired
	if time.Now().After(expiresAt) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "This reset link has expired. Please contact your administrator for a new one.",
		})
	}

	// Hash the new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to hash password")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to reset password",
		})
	}

	// Update the password
	_, err = h.db.Exec(ctx,
		`UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2`,
		string(hashedPassword), userID,
	)
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to update password")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to reset password",
		})
	}

	// Mark the token as used
	now := time.Now()
	h.db.Exec(ctx,
		`UPDATE password_reset_tokens SET used_at = $1 WHERE id = $2`,
		now, tokenID,
	)

	h.log.Info().
		Str("user_id", userID.String()).
		Msg("Password reset completed via admin link")

	return c.JSON(fiber.Map{
		"message": "Password reset successfully. You can now log in with your new password.",
	})
}

// SetupStatus checks if initial setup is needed (no users exist)
func (h *AuthHandler) SetupStatus(c *fiber.Ctx) error {
	needsSetup, err := h.authService.NeedsSetup(c.Context())
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to check setup status")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to check setup status",
		})
	}

	return c.JSON(fiber.Map{
		"needs_setup": needsSetup,
	})
}

// ============ Two-Factor Authentication Handlers ============

// GetTOTPStatus returns the current 2FA status for the user
func (h *AuthHandler) GetTOTPStatus(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	enabled, backupCodesRemaining, err := h.authService.GetTOTPStatus(c.Context(), userID)
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to get 2FA status")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get 2FA status",
		})
	}

	return c.JSON(fiber.Map{
		"enabled":                enabled,
		"backup_codes_remaining": backupCodesRemaining,
	})
}

// InitiateTOTPSetup starts the 2FA setup process
// Returns the QR code URL for scanning with authenticator app
func (h *AuthHandler) InitiateTOTPSetup(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	setup, err := h.authService.InitiateTOTPSetup(c.Context(), userID)
	if err != nil {
		if err == services.ErrTOTPAlreadyEnabled {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "2FA is already enabled",
			})
		}
		h.log.Error().Err(err).Msg("Failed to initiate 2FA setup")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to initiate 2FA setup",
		})
	}

	// Only return QR code URL - don't expose the raw secret
	return c.JSON(fiber.Map{
		"qrcode_url": setup.QRCodeURL, // otpauth:// URL for QR code generation
	})
}

// ConfirmTOTPSetupRequest represents the 2FA confirmation payload
type ConfirmTOTPSetupRequest struct {
	Code string `json:"code" validate:"required,len=6"`
}

// ConfirmTOTPSetup verifies the TOTP code and enables 2FA
// Returns backup codes that must be saved by the user
func (h *AuthHandler) ConfirmTOTPSetup(c *fiber.Ctx) error {
	var req ConfirmTOTPSetupRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	userID := middleware.GetUserID(c)

	backupCodes, err := h.authService.ConfirmTOTPSetup(c.Context(), userID, req.Code)
	if err != nil {
		if err == services.ErrTOTPAlreadyEnabled {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "2FA is already enabled",
			})
		}
		if err == services.ErrInvalidTOTPCode {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid verification code",
			})
		}
		h.log.Error().Err(err).Msg("Failed to confirm 2FA setup")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to enable 2FA",
		})
	}

	return c.JSON(fiber.Map{
		"message":      "2FA enabled successfully",
		"backup_codes": backupCodes,
	})
}

// DisableTOTPRequest represents the 2FA disable payload
type DisableTOTPRequest struct {
	Password string `json:"password" validate:"required"`
}

// DisableTOTP disables 2FA for the user (requires password confirmation)
func (h *AuthHandler) DisableTOTP(c *fiber.Ctx) error {
	var req DisableTOTPRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	userID := middleware.GetUserID(c)

	err := h.authService.DisableTOTP(c.Context(), userID, req.Password)
	if err != nil {
		if err == services.ErrTOTPNotEnabled {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "2FA is not enabled",
			})
		}
		if err == services.ErrInvalidCredentials {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid password",
			})
		}
		h.log.Error().Err(err).Msg("Failed to disable 2FA")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to disable 2FA",
		})
	}

	return c.JSON(fiber.Map{
		"message": "2FA disabled successfully",
	})
}

// RegenerateBackupCodesRequest represents the backup code regeneration payload
type RegenerateBackupCodesRequest struct {
	Password string `json:"password" validate:"required"`
}

// RegenerateBackupCodes generates new backup codes (invalidates old ones)
func (h *AuthHandler) RegenerateBackupCodes(c *fiber.Ctx) error {
	var req RegenerateBackupCodesRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	userID := middleware.GetUserID(c)

	backupCodes, err := h.authService.RegenerateBackupCodes(c.Context(), userID, req.Password)
	if err != nil {
		if err == services.ErrTOTPNotEnabled {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "2FA is not enabled",
			})
		}
		if err == services.ErrInvalidCredentials {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid password",
			})
		}
		h.log.Error().Err(err).Msg("Failed to regenerate backup codes")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to regenerate backup codes",
		})
	}

	return c.JSON(fiber.Map{
		"message":      "Backup codes regenerated successfully",
		"backup_codes": backupCodes,
	})
}
