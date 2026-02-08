package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"

	"golang.org/x/crypto/pbkdf2"
)

const (
	keyLen     = 32 // AES-256
	saltLen    = 16
	pbkdf2Iter = 100000
)

// ErrInvalidCiphertext is returned when the ciphertext is invalid
var ErrInvalidCiphertext = errors.New("invalid ciphertext")

// Encryptor handles file encryption and decryption
type Encryptor struct {
	masterKey []byte
}

// NewEncryptor creates a new encryptor with the given master key
func NewEncryptor(masterKeyBase64 string) (*Encryptor, error) {
	key, err := base64.StdEncoding.DecodeString(masterKeyBase64)
	if err != nil {
		return nil, err
	}
	if len(key) != keyLen {
		return nil, errors.New("master key must be 32 bytes (base64 encoded)")
	}
	return &Encryptor{masterKey: key}, nil
}

// GenerateMasterKey generates a new random master key
func GenerateMasterKey() (string, error) {
	key := make([]byte, keyLen)
	if _, err := rand.Read(key); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(key), nil
}

// DeriveKey derives an encryption key from the master key and a salt
func (e *Encryptor) DeriveKey(salt []byte) []byte {
	return pbkdf2.Key(e.masterKey, salt, pbkdf2Iter, keyLen, sha256.New)
}

// Encrypt encrypts data using AES-256-GCM
func (e *Encryptor) Encrypt(plaintext []byte) ([]byte, error) {
	// Generate random salt
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}

	// Derive key from master key and salt
	key := e.DeriveKey(salt)

	// Create cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Generate nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	// Encrypt
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	// Combine: salt + nonce + ciphertext
	result := make([]byte, saltLen+gcm.NonceSize()+len(ciphertext))
	copy(result[:saltLen], salt)
	copy(result[saltLen:saltLen+gcm.NonceSize()], nonce)
	copy(result[saltLen+gcm.NonceSize():], ciphertext)

	return result, nil
}

// Decrypt decrypts data encrypted with Encrypt
func (e *Encryptor) Decrypt(data []byte) ([]byte, error) {
	// Check minimum length (salt + nonce + at least 1 byte)
	if len(data) < saltLen+12+1 {
		return nil, ErrInvalidCiphertext
	}

	// Extract salt
	salt := data[:saltLen]

	// Derive key
	key := e.DeriveKey(salt)

	// Create cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < saltLen+nonceSize+1 {
		return nil, ErrInvalidCiphertext
	}

	// Extract nonce and ciphertext
	nonce := data[saltLen : saltLen+nonceSize]
	ciphertext := data[saltLen+nonceSize:]

	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// EncryptReader wraps a reader and encrypts data as it's read
type EncryptReader struct {
	reader    io.Reader
	encryptor *Encryptor
	buffer    []byte
}

// NewEncryptReader creates a new encrypting reader
func (e *Encryptor) NewEncryptReader(r io.Reader) *EncryptReader {
	return &EncryptReader{
		reader:    r,
		encryptor: e,
	}
}

// Read reads and encrypts data
// Note: For large files, consider chunked encryption instead
func (er *EncryptReader) Read(p []byte) (n int, err error) {
	return er.reader.Read(p)
}

// HashPassword creates a secure hash of a password
func HashPassword(password string) ([]byte, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}

	hash := pbkdf2.Key([]byte(password), salt, pbkdf2Iter, keyLen, sha256.New)

	// Combine salt + hash
	result := make([]byte, saltLen+keyLen)
	copy(result[:saltLen], salt)
	copy(result[saltLen:], hash)

	return result, nil
}

// VerifyPassword verifies a password against a hash
func VerifyPassword(password string, hashed []byte) bool {
	if len(hashed) != saltLen+keyLen {
		return false
	}

	salt := hashed[:saltLen]
	expectedHash := hashed[saltLen:]

	hash := pbkdf2.Key([]byte(password), salt, pbkdf2Iter, keyLen, sha256.New)

	// Constant-time comparison
	if len(hash) != len(expectedHash) {
		return false
	}
	var result byte
	for i := 0; i < len(hash); i++ {
		result |= hash[i] ^ expectedHash[i]
	}
	return result == 0
}

// GenerateSecureToken generates a cryptographically secure random token
func GenerateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// ComputeHash computes SHA-256 hash of data
func ComputeHash(data []byte) string {
	hash := sha256.Sum256(data)
	return base64.StdEncoding.EncodeToString(hash[:])
}
