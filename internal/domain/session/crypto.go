package session

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
)

const (
	tokenBytes = 32
	keyBytes   = 32
)

// NewToken gera o token aleatório do cookie (256 bits) e seu id (hash) para
// persistência. Só o hash vai ao banco: vazamento do banco não dá sessão.
func NewToken() (raw, id string, err error) {
	b := make([]byte, tokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("session: random: %w", err)
	}
	raw = base64.RawURLEncoding.EncodeToString(b)
	return raw, HashToken(raw), nil
}

// HashToken deriva o id persistido a partir do token do cookie.
func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// Cipher cifra/decifra tokens em repouso (AES-256-GCM, nonce aleatório
// prefixado ao ciphertext).
type Cipher struct{ aead cipher.AEAD }

// NewCipher recebe a chave em base64 padrão (32 bytes decodificados).
// Gere com: openssl rand -base64 32
func NewCipher(base64Key string) (*Cipher, error) {
	key, err := base64.StdEncoding.DecodeString(base64Key)
	if err != nil || len(key) != keyBytes {
		return nil, errors.New("session: SESSION_ENCRYPTION_KEY deve ser base64 de 32 bytes (openssl rand -base64 32)")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Cipher{aead: aead}, nil
}

// Seal cifra o texto.
func (c *Cipher) Seal(plain string) ([]byte, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return c.aead.Seal(nonce, nonce, []byte(plain), nil), nil
}

// Open decifra o texto.
func (c *Cipher) Open(sealed []byte) (string, error) {
	n := c.aead.NonceSize()
	if len(sealed) < n {
		return "", errors.New("session: ciphertext curto")
	}
	plain, err := c.aead.Open(nil, sealed[:n], sealed[n:], nil)
	if err != nil {
		return "", fmt.Errorf("session: decifrar: %w", err)
	}
	return string(plain), nil
}
