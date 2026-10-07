package session

import (
	"crypto/rand"
	"encoding/base64"
	"testing"
)

func testKey(t *testing.T) string {
	t.Helper()
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(b)
}

func TestCipherRoundTrip(t *testing.T) {
	c, err := NewCipher(testKey(t))
	if err != nil {
		t.Fatal(err)
	}
	sealed, err := c.Seal("eyJhbGciOi...")
	if err != nil {
		t.Fatal(err)
	}
	got, err := c.Open(sealed)
	if err != nil {
		t.Fatal(err)
	}
	if got != "eyJhbGciOi..." {
		t.Fatalf("round trip: %q", got)
	}
	// Nonce aleatório: mesmo texto, ciphertexts diferentes.
	again, _ := c.Seal("eyJhbGciOi...")
	if string(again) == string(sealed) {
		t.Fatal("ciphertext determinístico — nonce não é aleatório")
	}
}

func TestCipherRejeitaChaveInvalida(t *testing.T) {
	if _, err := NewCipher("curta"); err == nil {
		t.Fatal("esperava erro para chave inválida")
	}
	if _, err := NewCipher(base64.StdEncoding.EncodeToString(make([]byte, 16))); err == nil {
		t.Fatal("esperava erro para chave de 16 bytes")
	}
}

func TestCipherOpenComOutraChaveFalha(t *testing.T) {
	a, _ := NewCipher(testKey(t))
	b, _ := NewCipher(testKey(t))
	sealed, _ := a.Seal("segredo")
	if _, err := b.Open(sealed); err == nil {
		t.Fatal("esperava falha ao decifrar com outra chave")
	}
}

func TestNewTokenEHash(t *testing.T) {
	raw, id, err := NewToken()
	if err != nil {
		t.Fatal(err)
	}
	if raw == "" || len(id) != 64 {
		t.Fatalf("token=%q id=%q", raw, id)
	}
	if HashToken(raw) != id {
		t.Fatal("hash não bate")
	}
	raw2, _, _ := NewToken()
	if raw2 == raw {
		t.Fatal("tokens iguais")
	}
}
