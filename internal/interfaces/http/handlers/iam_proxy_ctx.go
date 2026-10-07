package handlers

import (
	"context"
	"net/http"
)

func contextWithIAMToken(r *http.Request, token string) context.Context {
	return context.WithValue(r.Context(), iamTokenKey{}, token)
}
