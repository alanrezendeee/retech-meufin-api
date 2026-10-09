package authclient

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func TestPost_AssinaBodyTimestampENonce(t *testing.T) {
	const secret = "segredo-de-teste"
	var gotBody []byte
	var gotTS, gotNonce, gotSig string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		gotTS, gotNonce, gotSig = r.Header.Get("X-Timestamp"), r.Header.Get("X-Nonce"), r.Header.Get("X-Signature")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, Secret: secret})
	resp, err := c.post(context.Background(), "/v1/x", map[string]string{"email": "u@x"})
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if _, err := strconv.ParseInt(gotTS, 10, 64); err != nil {
		t.Fatalf("timestamp inválido: %q", gotTS)
	}
	if len(gotNonce) != 32 {
		t.Fatalf("esperava nonce de 32 chars hex, obteve %q", gotNonce)
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(gotBody)
	mac.Write([]byte(gotTS))
	mac.Write([]byte(gotNonce))
	if want := hex.EncodeToString(mac.Sum(nil)); want != gotSig {
		t.Fatalf("assinatura não cobre body||timestamp||nonce")
	}
}
