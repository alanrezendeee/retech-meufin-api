package middleware

import (
	"net/http/httptest"
	"testing"
)

func TestClientIPUsaUltimoHopDoXFF(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "10.0.0.1:5555"
	r.Header.Set("X-Forwarded-For", "1.1.1.1, 203.0.113.7")
	if got := ClientIP(r); got != "203.0.113.7" {
		t.Fatalf("got %q; o primeiro hop é controlado pelo cliente e não pode valer", got)
	}
}

func TestClientIPFallbacks(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "10.0.0.1:5555"
	r.Header.Set("X-Real-IP", "198.51.100.2")
	if got := ClientIP(r); got != "198.51.100.2" {
		t.Fatalf("X-Real-IP: got %q", got)
	}
	r.Header.Del("X-Real-IP")
	if got := ClientIP(r); got != "10.0.0.1" {
		t.Fatalf("RemoteAddr: got %q", got)
	}
}
