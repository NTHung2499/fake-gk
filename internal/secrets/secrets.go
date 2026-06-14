package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"strings"
)

type Cipher struct {
	aead cipher.AEAD
}

func NewCipher(secret string) (*Cipher, error) {
	secret = strings.TrimSpace(secret)
	if len(secret) < 16 {
		return nil, errors.New("APP_SECRET must be at least 16 characters")
	}

	key := sha256.Sum256([]byte(secret))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Cipher{aead: aead}, nil
}

func (c *Cipher) Encrypt(plaintext string) (string, error) {
	plaintext = strings.TrimSpace(plaintext)
	if plaintext == "" {
		return "", errors.New("secret value is required")
	}

	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := c.aead.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (c *Cipher) Decrypt(encoded string) (string, error) {
	payload, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	if len(payload) < c.aead.NonceSize() {
		return "", errors.New("encrypted value is malformed")
	}
	nonce := payload[:c.aead.NonceSize()]
	ciphertext := payload[c.aead.NonceSize():]
	plaintext, err := c.aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func Hint(apiKey string) string {
	apiKey = strings.TrimSpace(apiKey)
	if len(apiKey) <= 8 {
		return "********"
	}
	prefix := apiKey[:min(len(apiKey), 3)]
	suffix := apiKey[len(apiKey)-4:]
	return prefix + "..." + suffix
}

func LooksLikeOpenAIKey(apiKey string) bool {
	apiKey = strings.TrimSpace(apiKey)
	return strings.HasPrefix(apiKey, "sk-") && len(apiKey) >= 20
}
