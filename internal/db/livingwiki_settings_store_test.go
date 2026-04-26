// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package db

import (
	"strings"
	"testing"
)

// TestLivingWikiEncryption verifies that:
// 1. encrypt + decrypt round-trips correctly.
// 2. The stored value is not plaintext (i.e. the secret is encrypted at rest).
// 3. Empty strings are stored as empty (no overhead).
// 4. Dev-mode (empty key) stores plaintext and decrypts it unchanged.
func TestLivingWikiEncryption(t *testing.T) {
	store := &LivingWikiSettingsStore{encryptionKey: "test-key-for-unit-tests"}

	t.Run("round-trip", func(t *testing.T) {
		plaintext := "ghp_super_secret_token_123"
		enc, err := store.EncryptForTest(plaintext)
		if err != nil {
			t.Fatalf("encrypt: %v", err)
		}
		dec, err := store.DecryptForTest(enc)
		if err != nil {
			t.Fatalf("decrypt: %v", err)
		}
		if dec != plaintext {
			t.Errorf("expected %q, got %q", plaintext, dec)
		}
	})

	t.Run("ciphertext_is_not_plaintext", func(t *testing.T) {
		plaintext := "very-secret-webhook-hmac-key"
		enc, err := store.EncryptForTest(plaintext)
		if err != nil {
			t.Fatalf("encrypt: %v", err)
		}
		if strings.Contains(enc, plaintext) {
			t.Error("encrypted value should not contain the plaintext secret")
		}
	})

	t.Run("empty_string_round_trip", func(t *testing.T) {
		enc, err := store.EncryptForTest("")
		if err != nil {
			t.Fatalf("encrypt empty: %v", err)
		}
		if enc != "" {
			t.Errorf("expected empty ciphertext for empty input, got %q", enc)
		}
		dec, err := store.DecryptForTest("")
		if err != nil {
			t.Fatalf("decrypt empty: %v", err)
		}
		if dec != "" {
			t.Errorf("expected empty plaintext, got %q", dec)
		}
	})

	t.Run("different_nonce_each_time", func(t *testing.T) {
		plaintext := "same-secret"
		enc1, _ := store.EncryptForTest(plaintext)
		enc2, _ := store.EncryptForTest(plaintext)
		if enc1 == enc2 {
			t.Error("expected different ciphertexts for same plaintext (nonce must be random)")
		}
	})

	t.Run("dev_mode_no_encryption", func(t *testing.T) {
		devStore := &LivingWikiSettingsStore{encryptionKey: ""}
		plaintext := "dev-token"
		enc, err := devStore.EncryptForTest(plaintext)
		if err != nil {
			t.Fatalf("dev encrypt: %v", err)
		}
		if enc != plaintext {
			t.Errorf("dev mode should store plaintext, got %q", enc)
		}
		dec, err := devStore.DecryptForTest(enc)
		if err != nil {
			t.Fatalf("dev decrypt: %v", err)
		}
		if dec != plaintext {
			t.Errorf("dev mode should return plaintext, got %q", dec)
		}
	})
}
