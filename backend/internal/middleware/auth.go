package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/tessera/tessera/internal/config"
	"github.com/tessera/tessera/internal/services"
)

// AuthMiddleware handles JWT authentication
type AuthMiddleware struct {
	authService          *services.AuthService
	jwtConfig            config.JWTConfig
	validateSessionCache bool // Set to true to check session validity on each request
}

// NewAuthMiddleware creates a new auth middleware
func NewAuthMiddleware(authService *services.AuthService, jwtConfig config.JWTConfig) *AuthMiddleware {
	return &AuthMiddleware{
		authService:          authService,
		jwtConfig:            jwtConfig,
		validateSessionCache: true, // Enable session validation by default
	}
}

// Authenticate verifies the JWT token and sets user context
func (m *AuthMiddleware) Authenticate(c *fiber.Ctx) error {
	// Get token from Authorization header
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Missing authorization header",
		})
	}

	// Check Bearer prefix
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid authorization header format",
		})
	}

	tokenString := parts[1]

	// Validate token
	claims, err := m.authService.ValidateToken(tokenString)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid or expired token",
		})
	}

	// Check if session is still valid (enables token revocation)
	// This prevents use of tokens from logged-out sessions
	if m.validateSessionCache && claims.SessionID != "" {
		if !m.authService.ValidateSession(c.Context(), claims.SessionID) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Session has been revoked",
			})
		}
	}

	// Set user info in context
	c.Locals("userID", claims.UserID)
	c.Locals("sessionID", claims.SessionID)
	c.Locals("userRole", claims.Role)

	return c.Next()
}

// RequireAdmin checks if the user has admin role
func (m *AuthMiddleware) RequireAdmin(c *fiber.Ctx) error {
	role, ok := c.Locals("userRole").(string)
	if !ok || role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Admin access required",
		})
	}
	return c.Next()
}

// GetUserID extracts the user ID from context
func GetUserID(c *fiber.Ctx) uuid.UUID {
	if userID, ok := c.Locals("userID").(uuid.UUID); ok {
		return userID
	}
	return uuid.Nil
}

// GetSessionID extracts the session ID from context
func GetSessionID(c *fiber.Ctx) string {
	if sessionID, ok := c.Locals("sessionID").(string); ok {
		return sessionID
	}
	return ""
}

// GetUserRole extracts the user role from context
func GetUserRole(c *fiber.Ctx) string {
	if role, ok := c.Locals("userRole").(string); ok {
		return role
	}
	return "user"
}
