package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
)

type AESCrypto struct {
	aead cipher.AEAD
}

func New(secret string) (*AESCrypto, error) {
	sum := sha256.Sum256([]byte(secret))
	block, err := aes.NewCipher(sum[:])
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &AESCrypto{aead: aead}, nil
}

func (c *AESCrypto) Encrypt(data []byte) (string, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	out := c.aead.Seal(nonce, nonce, data, nil)
	return base64.StdEncoding.EncodeToString(out), nil
}

func (c *AESCrypto) Decrypt(encoded string) ([]byte, error) {
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}
	ns := c.aead.NonceSize()
	if len(raw) < ns {
		return nil, fmt.Errorf("encrypted payload is shorter than nonce")
	}
	return c.aead.Open(nil, raw[:ns], raw[ns:], nil)
}
