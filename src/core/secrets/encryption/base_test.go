package encryption_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/Nigel2392/go-django/src/core/secrets/encryption"
)

func TestEncryption(t *testing.T) {
	// Standard 32-byte AES-256 keys for testing
	key1 := bytes.Repeat([]byte{0x01}, 32)
	key2 := bytes.Repeat([]byte{0x02}, 32)
	key3 := bytes.Repeat([]byte{0x03}, 32)

	plaintext := []byte("the quick brown fox jumps over the lazy dog")
	ctx := context.Background()

	// -------------------------------------------------------------------------
	// HAPPY PATHS
	// -------------------------------------------------------------------------

	t.Run("HappyPath_BasicEncryption", func(t *testing.T) {
		enc, err := encryption.GetCrypto(encryption.DEFAULT, key1, nil)
		if err != nil || enc == nil {
			t.Fatalf("failed to initialize encryption: %v", err)
		}

		ciphertext, err := enc.Encrypt(ctx, plaintext)
		if err != nil {
			t.Fatalf("encryption failed: %v", err)
		}

		decrypted, err := enc.Decrypt(ctx, ciphertext)
		if err != nil {
			t.Fatalf("decryption failed: %v", err)
		}

		if !bytes.Equal(plaintext, decrypted) {
			t.Errorf("decrypted text does not match original plaintext")
		}
	})

	t.Run("HappyPath_WithAdditionalData", func(t *testing.T) {
		enc, _ := encryption.GetCrypto(encryption.AES, key1, nil)

		// Tie the encryption to a specific context (e.g., user ID)
		ctxWithAAD := encryption.ContextWithAdditionalData(ctx, []byte("user-123"))

		ciphertext, err := enc.Encrypt(ctxWithAAD, plaintext)
		if err != nil {
			t.Fatalf("encryption with AAD failed: %v", err)
		}

		// Decrypting with the exact same context must succeed
		decrypted, err := enc.Decrypt(ctxWithAAD, ciphertext)
		if err != nil {
			t.Fatalf("decryption with AAD failed: %v", err)
		}

		if !bytes.Equal(plaintext, decrypted) {
			t.Errorf("decrypted text does not match original plaintext")
		}
	})

	t.Run("HappyPath_FallbackKeyRotation", func(t *testing.T) {
		// 1. Encrypt with an "old" key
		encOld, _ := encryption.GetCrypto(encryption.DEFAULT, key2, nil)
		ciphertext, _ := encOld.Encrypt(ctx, plaintext)

		// 2. Initialize a new engine where key1 is primary, but key2 is a fallback
		encNew, _ := encryption.GetCrypto(encryption.DEFAULT, key1, [][]byte{key2, key3})

		// 3. Decrypting the old ciphertext with the new engine should succeed via fallback
		decrypted, err := encNew.Decrypt(ctx, ciphertext)
		if err != nil {
			t.Fatalf("failed to decrypt using fallback key: %v", err)
		}

		if !bytes.Equal(plaintext, decrypted) {
			t.Errorf("decrypted text does not match original plaintext")
		}
	})

	// -------------------------------------------------------------------------
	// UNHAPPY PATHS
	// -------------------------------------------------------------------------

	t.Run("UnhappyPath_UnknownEngine", func(t *testing.T) {
		enc, err := encryption.GetCrypto("encryption.NON_EXISTENT", key1, nil)
		if err != nil {
			t.Fatalf("expected no error for unknown engine, got %v", err)
		}
		if enc != nil {
			t.Errorf("expected nil engine for unknown registry key, got %v", enc)
		}
	})

	t.Run("UnhappyPath_InvalidKeySize", func(t *testing.T) {
		badKey := []byte("too-short") // AES requires 16, 24, or 32 bytes
		_, err := encryption.GetCrypto(encryption.DEFAULT, badKey, nil)
		if err == nil {
			t.Errorf("expected an error when initializing with invalid key size, got nil")
		}
	})

	t.Run("UnhappyPath_WrongKeyUnknownID", func(t *testing.T) {
		enc1, _ := encryption.GetCrypto(encryption.DEFAULT, key1, nil)
		ciphertext, _ := enc1.Encrypt(ctx, plaintext)

		// Try to decrypt with a completely different key (different Key ID)
		enc2, _ := encryption.GetCrypto(encryption.DEFAULT, key2, nil)
		_, err := enc2.Decrypt(ctx, ciphertext)

		if err == nil {
			t.Errorf("expected decryption to fail with unknown key, but it succeeded")
		}
	})

	t.Run("UnhappyPath_TamperedCiphertext", func(t *testing.T) {
		enc, _ := encryption.GetCrypto(encryption.DEFAULT, key1, nil)
		ciphertext, _ := enc.Encrypt(ctx, plaintext)

		// Flip a bit in the authentication tag (the last 16 bytes of the payload)
		ciphertext[len(ciphertext)-1] ^= 0xFF

		_, err := enc.Decrypt(ctx, ciphertext)
		if err == nil {
			t.Errorf("expected decryption to fail on tampered ciphertext, but it succeeded")
		}
	})

	t.Run("UnhappyPath_AADMismatchSwapAttack", func(t *testing.T) {
		enc, _ := encryption.GetCrypto(encryption.DEFAULT, key1, nil)

		ctxUserA := encryption.ContextWithAdditionalData(ctx, []byte("user-A"))
		ctxUserB := encryption.ContextWithAdditionalData(ctx, []byte("user-B"))

		// Encrypt for User A
		ciphertext, _ := enc.Encrypt(ctxUserA, plaintext)

		// Try to decrypt in User B's context
		_, err := enc.Decrypt(ctxUserB, ciphertext)
		if err == nil {
			t.Errorf("expected decryption to fail due to AAD mismatch, but it succeeded")
		}

		// Try to decrypt with no AAD at all
		_, err = enc.Decrypt(ctx, ciphertext)
		if err == nil {
			t.Errorf("expected decryption to fail with missing AAD, but it succeeded")
		}
	})

	t.Run("UnhappyPath_PayloadTooShort", func(t *testing.T) {
		enc, _ := encryption.GetCrypto(encryption.DEFAULT, key1, nil)

		shortPayload := []byte("short")
		_, err := enc.Decrypt(ctx, shortPayload)

		if err == nil {
			t.Errorf("expected error for payload too short, got nil")
		}
	})

	// coverage
	t.Run("Registry_OverwriteEngine", func(t *testing.T) {
		customEngineName := "encryption.CUSTOM"

		overwritten := encryption.RegisterCrypto(customEngineName, func(k []byte, f [][]byte) (encryption.Encryption, error) {
			return nil, nil
		})
		if overwritten {
			t.Errorf("expected overwritten to be false on first registration")
		}

		overwritten = encryption.RegisterCrypto(customEngineName, func(k []byte, f [][]byte) (encryption.Encryption, error) {
			return nil, nil
		})
		if !overwritten {
			t.Errorf("expected overwritten to be true on second registration")
		}
	})
}
