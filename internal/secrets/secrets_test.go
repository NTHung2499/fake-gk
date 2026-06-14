package secrets

import "testing"

func TestCipherEncryptDecrypt(t *testing.T) {
	cipher, err := NewCipher("this-is-a-long-test-secret")
	if err != nil {
		t.Fatalf("NewCipher() error = %v", err)
	}

	encrypted, err := cipher.Encrypt("sk-test-secret")
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	if encrypted == "sk-test-secret" {
		t.Fatal("Encrypt() returned plaintext")
	}

	decrypted, err := cipher.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}
	if decrypted != "sk-test-secret" {
		t.Fatalf("Decrypt() = %q, want %q", decrypted, "sk-test-secret")
	}
}

func TestCipherRejectsWrongSecret(t *testing.T) {
	cipherA, err := NewCipher("this-is-a-long-test-secret")
	if err != nil {
		t.Fatalf("NewCipher(A) error = %v", err)
	}
	cipherB, err := NewCipher("this-is-another-test-secret")
	if err != nil {
		t.Fatalf("NewCipher(B) error = %v", err)
	}
	encrypted, err := cipherA.Encrypt("sk-test-secret")
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	if _, err := cipherB.Decrypt(encrypted); err == nil {
		t.Fatal("Decrypt() with wrong secret succeeded")
	}
}

func TestHintAndValidation(t *testing.T) {
	if got := Hint("sk-1234567890abcdef"); got != "sk-...cdef" {
		t.Fatalf("Hint() = %q", got)
	}
	if !LooksLikeOpenAIKey("sk-1234567890abcdef123") {
		t.Fatal("LooksLikeOpenAIKey() rejected valid-looking key")
	}
	if LooksLikeOpenAIKey("not-a-key") {
		t.Fatal("LooksLikeOpenAIKey() accepted invalid key")
	}
}
