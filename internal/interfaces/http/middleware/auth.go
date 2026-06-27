package middleware

import (
	"net/http"
	"strings"

	"github.com/MicahParks/keyfunc/v2"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/retechfin/retechfin-api/internal/interfaces/http/errrespond"
)

// Chaves de contexto preenchidas pelo middleware de autenticação.
const (
	CtxUserID = "user_id"
	CtxEmail  = "email"
	CtxRoles  = "roles"
)

// AuthClaims espelha os claims emitidos pelo retech-auth-api (RS256).
// O workspace do MeuFin vem de tenant_id — nunca de header do cliente.
type AuthClaims struct {
	UserID        string   `json:"user_id"`
	Email         string   `json:"email"`
	Name          string   `json:"name"`
	ApplicationID string   `json:"application_id"`
	TenantID      *string  `json:"tenant_id"`
	Roles         []string `json:"roles"`
	jwt.RegisteredClaims
}

// RequireAuth valida o JWT (Authorization: Bearer) contra o JWKS do auth,
// confere a aplicação (se applicationID != "") e deriva o workspace do tenant_id.
func RequireAuth(jwks *keyfunc.JWKS, applicationID string) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := c.GetHeader("Authorization")
		if raw == "" {
			errrespond.Message(c, http.StatusUnauthorized, errrespond.CodeUnauthorized, "token de autenticação ausente")
			c.Abort()
			return
		}
		parts := strings.SplitN(raw, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			errrespond.Message(c, http.StatusUnauthorized, errrespond.CodeUnauthorized, "formato inválido: use 'Authorization: Bearer <token>'")
			c.Abort()
			return
		}

		claims := &AuthClaims{}
		token, err := jwt.ParseWithClaims(parts[1], claims, jwks.Keyfunc,
			jwt.WithValidMethods([]string{"RS256"}))
		if err != nil || !token.Valid {
			errrespond.Message(c, http.StatusUnauthorized, errrespond.CodeUnauthorized, "token inválido ou expirado")
			c.Abort()
			return
		}

		// Defesa em profundidade: garante que o token é desta aplicação.
		if applicationID != "" && claims.ApplicationID != applicationID {
			errrespond.Message(c, http.StatusForbidden, errrespond.CodeForbidden, "token não pertence a esta aplicação")
			c.Abort()
			return
		}

		// Workspace = tenant_id do token. Header X-Workspace-ID é ignorado.
		if claims.TenantID == nil || strings.TrimSpace(*claims.TenantID) == "" {
			errrespond.Message(c, http.StatusForbidden, errrespond.CodeForbidden, "usuário sem workspace (tenant_id) no token")
			c.Abort()
			return
		}
		ws, err := uuid.Parse(strings.TrimSpace(*claims.TenantID))
		if err != nil {
			errrespond.Message(c, http.StatusForbidden, errrespond.CodeForbidden, "tenant_id do token não é um UUID válido")
			c.Abort()
			return
		}

		c.Set(CtxWorkspaceID, ws)
		c.Set(CtxUserID, claims.UserID)
		c.Set(CtxEmail, claims.Email)
		c.Set(CtxRoles, claims.Roles)
		c.Next()
	}
}
