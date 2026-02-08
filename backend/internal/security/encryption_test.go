package security

import (
	"bytes"
	"encoding/base64"
	"testing"
)

func createTestEncryptor(t *testing.T) *Encryptor {
	key, err := GenerateMasterKey()
	if err != nil {
		t.Fatalf("GenerateMasterKey() error = %v", err)
	}
	encryptor, err := NewEncryptor(key)
	if err != nil {
		t.Fatalf("NewEncryptor() error = %v", err)
	}
	return encryptor
}

func TestGenerateMasterKey(t *testing.T) {
	t.Run("generates valid key", func(t *testing.T) {
		key, err := GenerateMasterKey()
		if err != nil {
			t.Fatalf("GenerateMasterKey() error = %v", err)
		}

		// Decode and check length
		decoded, err := base64.StdEncoding.DecodeString(key)
		if err != nil {
			t.Fatalf("GenerateMasterKey() produced invalid base64: %v", err)
		}
		if len(decoded) != 32 {
			t.Errorf("GenerateMasterKey() key length = %d, want 32", len(decoded))
		}
	})

	t.Run("generates unique keys", func(t *testing.T) {
		keys := make(map[string]bool)
		for i := 0; i < 100; i++ {
			key, err := GenerateMasterKey()
			if err != nil {
				t.Fatalf("GenerateMasterKey() error = %v", err)
			}
			if keys[key] {
				t.Error("GenerateMasterKey() generated duplicate key")
			}
			keys[key] = true
		}
	})
}

func TestNewEncryptor(t *testing.T) {
	t.Run("creates encryptor with valid key", func(t *testing.T) {
		key, _ := GenerateMasterKey()
		encryptor, err := NewEncryptor(key)
		if err != nil {
			t.Fatalf("NewEncryptor() error = %v", err)
		}
		if encryptor == nil {
			t.Error("NewEncryptor() returned nil encryptor")
		}
	})

	t.Run("rejects invalid base64", func(t *testing.T) {
		_, err := NewEncryptor("not-valid-base64!!!")
		if err == nil {
			t.Error("NewEncryptor() should reject invalid base64")
		}
	})

	t.Run("rejects wrong key length", func(t *testing.T) {
		shortKey := base64.StdEncoding.EncodeToString([]byte("short"))
		_, err := NewEncryptor(shortKey)
		if err == nil {
			t.Error("NewEncryptor() should reject short key")
		}
	})
}

func TestEncryptor_EncryptDecrypt(t *testing.T) {
	encryptor := createTestEncryptor(t)

	t.Run("encrypts and decrypts data", func(t *testing.T) {
		plaintext := []byte("Hello, World! This is a test message.")

		ciphertext, err := encryptor.Encrypt(plaintext)
		if err != nil {
			t.Fatalf("Encrypt() error = %v", err)
		}

		decrypted, err := encryptor.Decrypt(ciphertext)
		if err != nil {
			t.Fatalf("Decrypt() error = %v", err)
		}

		if !bytes.Equal(plaintext, decrypted) {
			t.Errorf("Decrypt() = %s, want %s", decrypted, plaintext)
		}
	})

	t.Run("handles empty data", func(t *testing.T) {
		plaintext := []byte("")

		ciphertext, err := encryptor.Encrypt(plaintext)
		if err != nil {
			t.Fatalf("Encrypt() error = %v", err)
		}

		decrypted, err := encryptor.Decrypt(ciphertext)
		if err != nil {
			t.Fatalf("Decrypt() error = %v", err)
		}

		if !bytes.Equal(plaintext, decrypted) {
			t.Errorf("Decrypt() = %s, want empty", decrypted)
		}
	})

	t.Run("handles large data", func(t *testing.T) {
		plaintext := make([]byte, 1024*1024) // 1MB
		for i := range plaintext {
			plaintext[i] = byte(i % 256)
		}

		ciphertext, err := encryptor.Encrypt(plaintext)
		if err != nil {
			t.Fatalf("Encrypt() error = %v", err)
		}

		decrypted, err := encryptor.Decrypt(ciphertext)
		if err != nil {
			t.Fatalf("Decrypt() error = %v", err)
		}

		if !bytes.Equal(plaintext, decrypted) {
			t.Error("Decrypt() returned different data for large file")
		}
	})

	t.Run("ciphertext is different from plaintext", func(t *testing.T) {
		plaintext := []byte("Secret data")

		ciphertext, _ := encryptor.Encrypt(plaintext)

		if bytes.Equal(plaintext, ciphertext) {
			t.Error("Encrypt() returned plaintext as ciphertext")
		}
	})

	t.Run("same plaintext produces different ciphertext", func(t *testing.T) {
		plaintext := []byte("Same message")

		ciphertext1, _ := encryptor.Encrypt(plaintext)
		ciphertext2, _ := encryptor.Encrypt(plaintext)

		if bytes.Equal(ciphertext1, ciphertext2) {
			t.Error("Encrypt() produced same ciphertext for same plaintext")
		}
	})

	t.Run("different keys produce different ciphertext", func(t *testing.T) {
		encryptor2 := createTestEncryptor(t)
		plaintext := []byte("Test message")

		ciphertext1, _ := encryptor.Encrypt(plaintext)
		ciphertext2, _ := encryptor2.Encrypt(plaintext)

		if bytes.Equal(ciphertext1, ciphertext2) {
			t.Error("Different keys produced same ciphertext")
		}
	})
}

func TestEncryptor_DecryptErrors(t *testing.T) {
	encryptor := createTestEncryptor(t)

	t.Run("rejects too short ciphertext", func(t *testing.T) {
		_, err := encryptor.Decrypt([]byte("short"))
		if err != ErrInvalidCiphertext {
			t.Errorf("Decrypt() error = %v, want ErrInvalidCiphertext", err)
		}
	})

	t.Run("rejects tampered ciphertext", func(t *testing.T) {
		plaintext := []byte("Original message")
		ciphertext, _ := encryptor.Encrypt(plaintext)

		// Tamper with ciphertext
		ciphertext[len(ciphertext)-1] ^= 0xFF

		_, err := encryptor.Decrypt(ciphertext)
		if err == nil {
			t.Error("Decrypt() should reject tampered ciphertext")
		}
	})

	t.Run("rejects ciphertext from different key", func(t *testing.T) {
		encryptor2 := createTestEncryptor(t)
		plaintext := []byte("Test message")

		ciphertext, _ := encryptor.Encrypt(plaintext)

		_, err := encryptor2.Decrypt(ciphertext)
		if err == nil {
			t.Error("Decrypt() should reject ciphertext from different key")
		}
	})
}

func TestHashPassword(t *testing.T) {
	t.Run("creates hash", func(t *testing.T) {
		hash, err := HashPassword("mypassword123")
		if err != nil {
			t.Fatalf("HashPassword() error = %v", err)
		}
		if len(hash) != saltLen+keyLen {
			t.Errorf("HashPassword() length = %d, want %d", len(hash), saltLen+keyLen)
		}
	})

	t.Run("same password produces different hashes", func(t *testing.T) {
		hash1, _ := HashPassword("password")
		hash2, _ := HashPassword("password")

		if bytes.Equal(hash1, hash2) {
			t.Error("HashPassword() produced same hash for same password")
		}
	})
}

func TestVerifyPassword(t *testing.T) {
	t.Run("verifies correct password", func(t *testing.T) {
		password := "correctpassword"
		hash, _ := HashPassword(password)

		if !VerifyPassword(password, hash) {
			t.Error("VerifyPassword() rejected correct password")
		}
	})

	t.Run("rejects wrong password", func(t *testing.T) {
		hash, _ := HashPassword("correctpassword")

		if VerifyPassword("wrongpassword", hash) {
			t.Error("VerifyPassword() accepted wrong password")
		}
	})

	t.Run("rejects invalid hash length", func(t *testing.T) {
		if VerifyPassword("password", []byte("short")) {
			t.Error("VerifyPassword() accepted invalid hash length")
		}
	})
}

func TestGenerateSecureToken(t *testing.T) {
	t.Run("generates token of correct length", func(t *testing.T) {
		token, err := GenerateSecureToken(32)
		if err != nil {
			t.Fatalf("GenerateSecureToken() error = %v", err)
		}

		// base64 URL encoding: 32 bytes -> ~43 characters
		decoded, err := base64.URLEncoding.DecodeString(token)
		if err != nil {
			t.Fatalf("GenerateSecureToken() produced invalid base64: %v", err)
		}
		if len(decoded) != 32 {
			t.Errorf("GenerateSecureToken() decoded length = %d, want 32", len(decoded))
		}
	})

	t.Run("generates unique tokens", func(t *testing.T) {
		tokens := make(map[string]bool)
		for i := 0; i < 100; i++ {
			token, err := GenerateSecureToken(32)
			if err != nil {
				t.Fatalf("GenerateSecureToken() error = %v", err)
			}
			if tokens[token] {
				t.Error("GenerateSecureToken() generated duplicate token")
			}
			tokens[token] = true
		}
	})
}

func TestComputeHash(t *testing.T) {
	t.Run("computes consistent hash", func(t *testing.T) {
		data := []byte("test data")
		hash1 := ComputeHash(data)
		hash2 := ComputeHash(data)

		if hash1 != hash2 {
			t.Errorf("ComputeHash() produced different hashes: %s != %s", hash1, hash2)
		}
	})

	t.Run("different data produces different hash", func(t *testing.T) {
		hash1 := ComputeHash([]byte("data1"))
		hash2 := ComputeHash([]byte("data2"))

		if hash1 == hash2 {
			t.Error("ComputeHash() produced same hash for different data")
		}
	})
}

// Benchmark tests
func BenchmarkEncrypt(b *testing.B) {
	key, _ := GenerateMasterKey()
	encryptor, _ := NewEncryptor(key)
	plaintext := make([]byte, 1024) // 1KB

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		encryptor.Encrypt(plaintext)
	}
}

func BenchmarkDecrypt(b *testing.B) {
	key, _ := GenerateMasterKey()
	encryptor, _ := NewEncryptor(key)
	plaintext := make([]byte, 1024) // 1KB
	ciphertext, _ := encryptor.Encrypt(plaintext)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		encryptor.Decrypt(ciphertext)
	}
}

func BenchmarkHashPassword(b *testing.B) {
	for i := 0; i < b.N; i++ {
		HashPassword("testpassword123")
	}
}

func BenchmarkVerifyPassword(b *testing.B) {
	hash, _ := HashPassword("testpassword123")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		VerifyPassword("testpassword123", hash)
	}
}
