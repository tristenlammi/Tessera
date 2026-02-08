package services

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	// TOTP settings
	totpDigits   = 6
	totpPeriod   = 30
	totpIssuer   = "Tessera"
	secretLength = 20 // 160 bits as recommended
)

// TOTPService handles two-factor authentication operations
type TOTPService struct{}

// NewTOTPService creates a new TOTP service
func NewTOTPService() *TOTPService {
	return &TOTPService{}
}

// GenerateSecret generates a new random TOTP secret
// Returns the secret in base32 format (for storage and authenticator apps)
func (s *TOTPService) GenerateSecret() (string, error) {
	secret := make([]byte, secretLength)
	if _, err := rand.Read(secret); err != nil {
		return "", fmt.Errorf("failed to generate random secret: %w", err)
	}

	// Encode to base32 without padding for compatibility with authenticator apps
	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret)
	return encoded, nil
}

// GenerateOTPAuthURL generates the otpauth:// URL for QR codes and manual entry
// This URL is compatible with Google Authenticator, Authy, Bitwarden, etc.
func (s *TOTPService) GenerateOTPAuthURL(secret, email string) string {
	// Format: otpauth://totp/Issuer:account?secret=SECRET&issuer=Issuer&algorithm=SHA1&digits=6&period=30
	return fmt.Sprintf(
		"otpauth://totp/%s:%s?secret=%s&issuer=%s&algorithm=SHA1&digits=%d&period=%d",
		totpIssuer,
		email,
		secret,
		totpIssuer,
		totpDigits,
		totpPeriod,
	)
}

// ValidateCode validates a TOTP code against the secret
// Returns true if the code is valid for the current or adjacent time windows
func (s *TOTPService) ValidateCode(secret, code string) bool {
	if len(code) != totpDigits {
		return false
	}

	// Check current time window and one before/after for clock drift tolerance
	now := time.Now().Unix()
	for i := int64(-1); i <= 1; i++ {
		counter := (now / totpPeriod) + i
		expectedCode := s.generateCode(secret, counter)
		if hmac.Equal([]byte(code), []byte(expectedCode)) {
			return true
		}
	}

	return false
}

// generateCode generates a TOTP code for a given counter value
func (s *TOTPService) generateCode(secret string, counter int64) string {
	// Decode the base32 secret
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err != nil {
		return ""
	}

	// Convert counter to big-endian bytes
	counterBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(counterBytes, uint64(counter))

	// Generate HMAC-SHA1
	h := hmac.New(sha1.New, key)
	h.Write(counterBytes)
	hash := h.Sum(nil)

	// Dynamic truncation (RFC 4226)
	offset := hash[len(hash)-1] & 0x0f
	binary := binary.BigEndian.Uint32(hash[offset:offset+4]) & 0x7fffffff

	// Generate 6-digit code
	otp := binary % 1000000
	return fmt.Sprintf("%06d", otp)
}

// GenerateBackupCodes generates a set of backup codes for account recovery
// Returns both the plain codes (to show user once) and hashed codes (to store)
func (s *TOTPService) GenerateBackupCodes(count int) (plainCodes []string, hashedCodes []string, err error) {
	plainCodes = make([]string, count)
	hashedCodes = make([]string, count)

	for i := 0; i < count; i++ {
		// Generate 8 random bytes = 16 hex characters
		codeBytes := make([]byte, 8)
		if _, err := rand.Read(codeBytes); err != nil {
			return nil, nil, fmt.Errorf("failed to generate backup code: %w", err)
		}

		// Format as XXXX-XXXX for readability
		code := fmt.Sprintf("%X-%X", codeBytes[:4], codeBytes[4:])
		plainCodes[i] = code

		// Hash for storage
		hash, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to hash backup code: %w", err)
		}
		hashedCodes[i] = string(hash)
	}

	return plainCodes, hashedCodes, nil
}

// ValidateBackupCode validates a backup code against the stored hashes
// Returns the index of the used code (-1 if invalid) so it can be removed
func (s *TOTPService) ValidateBackupCode(code string, hashedCodes []string) int {
	// Normalize the code (remove dashes, uppercase)
	code = strings.ToUpper(strings.ReplaceAll(code, "-", ""))

	// Also check with dashes for user convenience
	codeWithDashes := ""
	if len(code) == 16 {
		codeWithDashes = code[:8] + "-" + code[8:]
	}

	for i, hash := range hashedCodes {
		if hash == "" {
			continue // Already used
		}

		// Try without dashes
		if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(code)); err == nil {
			return i
		}

		// Try with dashes
		if codeWithDashes != "" {
			if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(codeWithDashes)); err == nil {
				return i
			}
		}
	}

	return -1
}

// FormatSecretForDisplay formats the secret in groups of 4 for easier manual entry
func (s *TOTPService) FormatSecretForDisplay(secret string) string {
	var groups []string
	for i := 0; i < len(secret); i += 4 {
		end := i + 4
		if end > len(secret) {
			end = len(secret)
		}
		groups = append(groups, secret[i:end])
	}
	return strings.Join(groups, " ")
}
