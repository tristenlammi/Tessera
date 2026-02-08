package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/bcrypt"

	"github.com/tessera/tessera/internal/config"
	"github.com/tessera/tessera/internal/models"
	"github.com/tessera/tessera/internal/repository"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotActive      = errors.New("user account is not active")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrEmailTaken         = errors.New("email already registered")
	ErrTOTPRequired       = errors.New("2FA code required")
	ErrInvalidTOTPCode    = errors.New("invalid 2FA code")
	ErrTOTPAlreadyEnabled = errors.New("2FA is already enabled")
	ErrTOTPNotEnabled     = errors.New("2FA is not enabled")
)

// sessionCacheEntry holds cached session validation result
type sessionCacheEntry struct {
	valid     bool
	expiresAt time.Time
}

// AuthService handles authentication operations
type AuthService struct {
	userRepo    *repository.UserRepository
	sessionRepo *repository.SessionRepository
	jwtConfig   config.JWTConfig
	totpService *TOTPService
	log         zerolog.Logger
	// Session validation cache (reduces Redis hits)
	sessionCache   map[string]*sessionCacheEntry
	sessionCacheMu sync.RWMutex
	cacheTTL       time.Duration
}

// NewAuthService creates a new auth service
func NewAuthService(userRepo *repository.UserRepository, sessionRepo *repository.SessionRepository, jwtConfig config.JWTConfig) *AuthService {
	svc := &AuthService{
		userRepo:     userRepo,
		sessionRepo:  sessionRepo,
		jwtConfig:    jwtConfig,
		totpService:  NewTOTPService(),
		log:          zerolog.Nop(),
		sessionCache: make(map[string]*sessionCacheEntry),
		cacheTTL:     5 * time.Second, // Short TTL balances performance vs security
	}
	// Start cache cleanup goroutine
	go svc.cleanupSessionCache()
	return svc
}

// SetLogger sets the logger for the auth service
func (s *AuthService) SetLogger(log zerolog.Logger) {
	s.log = log
}

// cleanupSessionCache periodically removes expired cache entries
func (s *AuthService) cleanupSessionCache() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		s.sessionCacheMu.Lock()
		for key, entry := range s.sessionCache {
			if now.After(entry.expiresAt) {
				delete(s.sessionCache, key)
			}
		}
		s.sessionCacheMu.Unlock()
	}
}

// RegisterInput contains registration data
type RegisterInput struct {
	Email    string
	Password string
	Name     string
}

// NeedsSetup checks if initial setup is needed (no users exist)
func (s *AuthService) NeedsSetup(ctx context.Context) (bool, error) {
	count, err := s.userRepo.Count(ctx)
	if err != nil {
		return false, err
	}
	return count == 0, nil
}

// Register creates a new user account
func (s *AuthService) Register(ctx context.Context, input RegisterInput) (*models.User, error) {
	// Check if email exists
	exists, err := s.userRepo.EmailExists(ctx, input.Email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrEmailTaken
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// Check if this is the first user (make them admin)
	userCount, err := s.userRepo.Count(ctx)
	if err != nil {
		return nil, err
	}
	role := models.RoleUser
	if userCount == 0 {
		role = models.RoleAdmin
	}

	user := &models.User{
		Email:        input.Email,
		PasswordHash: string(hashedPassword),
		Name:         input.Name,
		Role:         role,
		StorageLimit: 10 * 1024 * 1024 * 1024, // 10GB default
		IsActive:     true,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// LoginInput contains login credentials
type LoginInput struct {
	Email     string
	Password  string
	TOTPCode  string // Optional - required if 2FA enabled
	UserAgent string
	IPAddress string
}

// TokenPair contains access and refresh tokens
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// PendingAuthToken is returned when 2FA is required but not provided
type PendingAuthToken struct {
	Token string `json:"pending_auth_token"`
}

// Login authenticates a user and returns tokens
// Returns ErrTOTPRequired if 2FA is enabled but no code provided
func (s *AuthService) Login(ctx context.Context, input LoginInput) (*models.User, *TokenPair, *PendingAuthToken, error) {
	user, err := s.userRepo.GetByEmail(ctx, input.Email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, nil, nil, ErrInvalidCredentials
		}
		return nil, nil, nil, err
	}

	if !user.IsActive {
		return nil, nil, nil, ErrUserNotActive
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return nil, nil, nil, ErrInvalidCredentials
	}

	// Check if 2FA is enabled
	if user.TOTPEnabled {
		if input.TOTPCode == "" {
			// Generate a pending auth token instead of returning password flow
			pendingToken, err := generateRefreshToken() // reuse secure token generation
			if err != nil {
				return nil, nil, nil, err
			}
			if err := s.sessionRepo.CreatePendingAuth(ctx, pendingToken, user.ID, user.Email); err != nil {
				return nil, nil, nil, err
			}
			return user, nil, &PendingAuthToken{Token: pendingToken}, ErrTOTPRequired
		}

		// Try TOTP code first
		if !s.totpService.ValidateCode(user.TOTPSecret, input.TOTPCode) {
			// Try backup code
			codeIndex := s.totpService.ValidateBackupCode(input.TOTPCode, user.BackupCodes)
			if codeIndex == -1 {
				return nil, nil, nil, ErrInvalidTOTPCode
			}

			// Audit log: backup code used
			s.log.Warn().
				Str("user_id", user.ID.String()).
				Str("email", user.Email).
				Str("ip", input.IPAddress).
				Int("backup_code_index", codeIndex).
				Msg("Backup code used for login - consider prompting user to regenerate codes")

			// Remove used backup code
			user.BackupCodes[codeIndex] = ""
			if err := s.userRepo.UpdateBackupCodes(ctx, user.ID, user.BackupCodes); err != nil {
				return nil, nil, nil, err
			}
		}
	}

	// Update last login
	now := time.Now()
	user.LastLoginAt = &now
	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, nil, nil, err
	}

	// Generate tokens
	tokens, err := s.createSession(ctx, user.ID, user.Role, input.UserAgent, input.IPAddress)
	if err != nil {
		return nil, nil, nil, err
	}

	return user, tokens, nil, nil
}

// CompleteTOTPLoginInput contains data for completing 2FA login
type CompleteTOTPLoginInput struct {
	PendingToken string
	TOTPCode     string
	UserAgent    string
	IPAddress    string
}

// CompleteTOTPLogin completes login using a pending auth token and TOTP code
func (s *AuthService) CompleteTOTPLogin(ctx context.Context, input CompleteTOTPLoginInput) (*models.User, *TokenPair, error) {
	// Retrieve and consume the pending auth token
	pending, err := s.sessionRepo.GetPendingAuth(ctx, input.PendingToken)
	if err != nil {
		return nil, nil, ErrInvalidToken
	}

	user, err := s.userRepo.GetByID(ctx, pending.UserID)
	if err != nil {
		return nil, nil, ErrInvalidCredentials
	}

	if !user.IsActive {
		return nil, nil, ErrUserNotActive
	}

	// Validate TOTP code
	if !s.totpService.ValidateCode(user.TOTPSecret, input.TOTPCode) {
		// Try backup code
		codeIndex := s.totpService.ValidateBackupCode(input.TOTPCode, user.BackupCodes)
		if codeIndex == -1 {
			return nil, nil, ErrInvalidTOTPCode
		}

		// Audit log: backup code used
		s.log.Warn().
			Str("user_id", user.ID.String()).
			Str("email", user.Email).
			Str("ip", input.IPAddress).
			Int("backup_code_index", codeIndex).
			Msg("Backup code used for TOTP login - consider prompting user to regenerate codes")

		// Remove used backup code
		user.BackupCodes[codeIndex] = ""
		if err := s.userRepo.UpdateBackupCodes(ctx, user.ID, user.BackupCodes); err != nil {
			return nil, nil, err
		}
	}

	// Update last login
	now := time.Now()
	user.LastLoginAt = &now
	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, nil, err
	}

	// Generate tokens
	tokens, err := s.createSession(ctx, user.ID, user.Role, input.UserAgent, input.IPAddress)
	if err != nil {
		return nil, nil, err
	}

	return user, tokens, nil
}

// RefreshTokens generates new tokens from a refresh token
func (s *AuthService) RefreshTokens(ctx context.Context, refreshToken string) (*TokenPair, error) {
	session, err := s.sessionRepo.GetByRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, ErrInvalidToken
	}

	// Get user to obtain role
	user, err := s.userRepo.GetByID(ctx, session.UserID)
	if err != nil {
		return nil, ErrInvalidToken
	}

	// Delete old session
	if err := s.sessionRepo.Delete(ctx, session); err != nil {
		return nil, err
	}

	// Create new session
	return s.createSession(ctx, session.UserID, user.Role, session.UserAgent, session.IPAddress)
}

// Logout invalidates a session
func (s *AuthService) Logout(ctx context.Context, sessionID string) error {
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return err
	}

	// Invalidate cache entry
	s.sessionCacheMu.Lock()
	delete(s.sessionCache, sessionID)
	s.sessionCacheMu.Unlock()

	return s.sessionRepo.Delete(ctx, session)
}

// LogoutAll invalidates all sessions for a user
func (s *AuthService) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	// Clear all cache entries for this user's sessions
	sessions, _ := s.sessionRepo.GetUserSessions(ctx, userID)
	s.sessionCacheMu.Lock()
	for _, sess := range sessions {
		delete(s.sessionCache, sess.ID)
	}
	s.sessionCacheMu.Unlock()

	return s.sessionRepo.DeleteAllForUser(ctx, userID)
}

// ValidateSession checks if a session is still valid (not revoked)
// Uses a short-lived cache to reduce Redis hits for high-traffic scenarios
func (s *AuthService) ValidateSession(ctx context.Context, sessionID string) bool {
	// Check cache first
	s.sessionCacheMu.RLock()
	if entry, ok := s.sessionCache[sessionID]; ok && time.Now().Before(entry.expiresAt) {
		s.sessionCacheMu.RUnlock()
		return entry.valid
	}
	s.sessionCacheMu.RUnlock()

	// Cache miss - check Redis
	_, err := s.sessionRepo.GetByID(ctx, sessionID)
	valid := err == nil

	// Update cache
	s.sessionCacheMu.Lock()
	s.sessionCache[sessionID] = &sessionCacheEntry{
		valid:     valid,
		expiresAt: time.Now().Add(s.cacheTTL),
	}
	s.sessionCacheMu.Unlock()

	return valid
}

// CreateWebSocketTicket creates a short-lived ticket for WebSocket connections
func (s *AuthService) CreateWebSocketTicket(ctx context.Context, userID uuid.UUID) (string, error) {
	ticket, err := generateRefreshToken() // reuse secure token generation
	if err != nil {
		return "", err
	}
	if err := s.sessionRepo.CreateWSTicket(ctx, ticket, userID); err != nil {
		return "", err
	}
	return ticket, nil
}

// ValidateWebSocketTicket validates and consumes a WebSocket ticket
func (s *AuthService) ValidateWebSocketTicket(ctx context.Context, ticket string) (uuid.UUID, error) {
	wsTicket, err := s.sessionRepo.GetWSTicket(ctx, ticket)
	if err != nil {
		return uuid.Nil, ErrInvalidToken
	}
	return wsTicket.UserID, nil
}

// ValidateToken verifies a JWT and returns the claims
func (s *AuthService) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.jwtConfig.Secret), nil
	})

	if err != nil {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// GetUserByID retrieves a user by ID
func (s *AuthService) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	return s.userRepo.GetByID(ctx, id)
}

// ChangePassword updates the user's password
func (s *AuthService) ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	// Verify old password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPassword)); err != nil {
		return ErrInvalidCredentials
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// Update password
	if err := s.userRepo.UpdatePassword(ctx, userID, string(hashedPassword)); err != nil {
		return err
	}

	// Invalidate all sessions (force re-login)
	return s.sessionRepo.DeleteAllForUser(ctx, userID)
}

// UpdateTimezone updates the user's timezone preference
func (s *AuthService) UpdateTimezone(ctx context.Context, userID uuid.UUID, timezone string) error {
	return s.userRepo.UpdateTimezone(ctx, userID, timezone)
}

// createSession generates tokens and stores a session
func (s *AuthService) createSession(ctx context.Context, userID uuid.UUID, role, userAgent, ipAddress string) (*TokenPair, error) {
	sessionID := uuid.New().String()
	refreshToken, err := generateRefreshToken()
	if err != nil {
		return nil, err
	}
	expiresAt := time.Now().Add(s.jwtConfig.RefreshExpiry)

	// Create JWT
	accessToken, err := s.generateJWT(userID, sessionID, role)
	if err != nil {
		return nil, err
	}

	// Store session
	session := &models.Session{
		ID:           sessionID,
		UserID:       userID,
		RefreshToken: refreshToken,
		UserAgent:    userAgent,
		IPAddress:    ipAddress,
		ExpiresAt:    expiresAt,
		CreatedAt:    time.Now(),
	}

	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(s.jwtConfig.Expiry),
	}, nil
}

// JWTClaims represents the JWT payload
type JWTClaims struct {
	UserID    uuid.UUID `json:"user_id"`
	SessionID string    `json:"session_id"`
	Role      string    `json:"role"`
	jwt.RegisteredClaims
}

// generateJWT creates a new JWT token
func (s *AuthService) generateJWT(userID uuid.UUID, sessionID, role string) (string, error) {
	claims := JWTClaims{
		UserID:    userID,
		SessionID: sessionID,
		Role:      role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.jwtConfig.Expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "tessera",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtConfig.Secret))
}

// generateRefreshToken creates a random refresh token
func generateRefreshToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate secure random bytes: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// TOTPSetupResponse contains the data needed for 2FA setup
type TOTPSetupResponse struct {
	Secret    string `json:"secret"`     // Base32 secret for manual entry
	QRCodeURL string `json:"qrcode_url"` // otpauth:// URL for QR code generation
}

// InitiateTOTPSetup generates a new TOTP secret for the user
// The user should then verify the code before 2FA is enabled
func (s *AuthService) InitiateTOTPSetup(ctx context.Context, userID uuid.UUID) (*TOTPSetupResponse, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if user.TOTPEnabled {
		return nil, ErrTOTPAlreadyEnabled
	}

	// Generate new secret
	secret, err := s.totpService.GenerateSecret()
	if err != nil {
		return nil, err
	}

	// Store the secret temporarily (not yet enabled)
	if err := s.userRepo.UpdateTOTPSecret(ctx, userID, secret); err != nil {
		return nil, err
	}

	return &TOTPSetupResponse{
		Secret:    s.totpService.FormatSecretForDisplay(secret),
		QRCodeURL: s.totpService.GenerateOTPAuthURL(secret, user.Email),
	}, nil
}

// ConfirmTOTPSetup verifies the TOTP code and enables 2FA
// Returns the backup codes that the user should save
func (s *AuthService) ConfirmTOTPSetup(ctx context.Context, userID uuid.UUID, code string) ([]string, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if user.TOTPEnabled {
		return nil, ErrTOTPAlreadyEnabled
	}

	if user.TOTPSecret == "" {
		return nil, errors.New("2FA setup not initiated")
	}

	// Validate the code
	if !s.totpService.ValidateCode(user.TOTPSecret, code) {
		return nil, ErrInvalidTOTPCode
	}

	// Generate backup codes
	plainCodes, hashedCodes, err := s.totpService.GenerateBackupCodes(10)
	if err != nil {
		return nil, err
	}

	// Enable 2FA
	if err := s.userRepo.EnableTOTP(ctx, userID, hashedCodes); err != nil {
		return nil, err
	}

	return plainCodes, nil
}

// DisableTOTP disables 2FA for the user after password verification
func (s *AuthService) DisableTOTP(ctx context.Context, userID uuid.UUID, password string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	if !user.TOTPEnabled {
		return ErrTOTPNotEnabled
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return ErrInvalidCredentials
	}

	return s.userRepo.DisableTOTP(ctx, userID)
}

// RegenerateBackupCodes generates new backup codes (invalidates old ones)
func (s *AuthService) RegenerateBackupCodes(ctx context.Context, userID uuid.UUID, password string) ([]string, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if !user.TOTPEnabled {
		return nil, ErrTOTPNotEnabled
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	// Generate new backup codes
	plainCodes, hashedCodes, err := s.totpService.GenerateBackupCodes(10)
	if err != nil {
		return nil, err
	}

	if err := s.userRepo.UpdateBackupCodes(ctx, userID, hashedCodes); err != nil {
		return nil, err
	}

	return plainCodes, nil
}

// GetTOTPStatus returns the 2FA status for a user
func (s *AuthService) GetTOTPStatus(ctx context.Context, userID uuid.UUID) (enabled bool, backupCodesRemaining int, err error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return false, 0, err
	}

	remaining := 0
	for _, code := range user.BackupCodes {
		if code != "" {
			remaining++
		}
	}

	return user.TOTPEnabled, remaining, nil
}
