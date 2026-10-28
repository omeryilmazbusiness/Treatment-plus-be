package crypto_test

import (
	"testing"

	"github.com/treatment-plus/treatment-plus-be/internal/shared/crypto"
)

func TestAESGCM_RequiresKey(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for empty KMS key")
		}
	}()
	_ = crypto.NewAESGCM("")
}

func TestAESGCM_EncryptDecrypt(t *testing.T) {
	enc := crypto.NewAESGCM("my-super-secret-key-12345")

	input := "+90 555 123 4567 | patient@example.com"
	ciphertext, err := enc.Encrypt(input)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}
	if ciphertext == input {
		t.Fatalf("Ciphertext should not equal plaintext")
	}

	decrypted, err := enc.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}
	if decrypted != input {
		t.Fatalf("Expected %q, got %q", input, decrypted)
	}
}

func TestAESGCM_Empty(t *testing.T) {
	enc := crypto.NewAESGCM("test-kms-key-must-be-present-32b")
	cipher, err := enc.Encrypt("")
	if err != nil || cipher != "" {
		t.Fatalf("Expected empty, got %q, err=%v", cipher, err)
	}
	plain, err := enc.Decrypt("")
	if err != nil || plain != "" {
		t.Fatalf("Expected empty, got %q, err=%v", plain, err)
	}
}
