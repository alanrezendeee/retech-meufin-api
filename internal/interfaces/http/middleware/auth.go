package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/MicahParks/keyfunc/v2"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/retechfin/retechfin-api/internal/appctx"
	domsess "github.com/retechfin/retechfin-api/internal/domain/session"

	"github.com/retechfin/retechfin-api/internal/interfaces/http/errrespond"
)

// Chaves de contexto preenchidas pelo middleware de autenticação.
const (
	CtxUserID      = "user_id"
	CtxEmail       = "email"
	CtxRoles       = "roles"
	CtxPerms       = "perms"
	CtxAccessToken = "access_token" // JWT do auth em uso nesta requisição (Bearer ou da sessão)
	CtxSessionID   = "session_id"   // id (hash) da sessão quando autenticado por cookie
	CtxAuthVia     = "auth_via"     // "bearer" | "cookie"
)

// Valores de CtxAuthVia.
const (
	AuthViaBearer = "bearer"
	AuthViaCookie = "cookie"
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
	Perms         []string `json:"perms"` // codes "subject:action" das permissions efetivas; master = ["all:manage"]
	jwt.RegisteredClaims
}

// SessionResolver transforma o token do cookie em sessão com access token
// válido (renovado se preciso). Implementado por application/session.Service.
type SessionResolver interface {
	Resolve(ctx context.Context, rawToken string) (domsess.Session, error)
}

// AuthOptions configura RequireAuth.
type AuthOptions struct {
	JWKS          *keyfunc.JWKS
	ApplicationID string
	// Sessions habilita autenticação por cookie (gateway). Nil = só Bearer.
	Sessions SessionResolver
	// BearerDisabled recusa `Authorization: Bearer` (401): após o rollout do
	// admin por cookie, fecha a segunda porta — mesmo padrão do CashFlowfy.
	BearerDisabled bool
	Cookie         SessionCookie
	// AllowedOrigins alimenta a verificação CSRF das requisições por cookie
	// (normalmente a mesma lista do CORS).
	AllowedOrigins []string
}

// RequireAuth autentica a requisição por `Authorization: Bearer` OU pelo
// cookie de sessão do gateway. Nos dois casos o JWT do auth é validado contra
// o JWKS, a aplicação é conferida (se applicationID != "") e o workspace é
// derivado do tenant_id.
//
// Bearer tem precedência (clientes não-browser). Pelo cookie, requisições
// que mudam estado passam ainda pela verificação CSRF (Sec-Fetch-Site/Origin).
func RequireAuth(opts AuthOptions) gin.HandlerFunc {
	allowed := originSet(opts.AllowedOrigins)

	return func(c *gin.Context) {
		rawJWT, via, ok := resolveCredential(c, opts, allowed)
		if !ok {
			return // resposta já escrita
		}

		claims := &AuthClaims{}
		token, err := jwt.ParseWithClaims(rawJWT, claims, opts.JWKS.Keyfunc,
			jwt.WithValidMethods([]string{"RS256"}))
		if err != nil || !token.Valid {
			if via == AuthViaCookie {
				opts.Cookie.Clear(c)
			}
			errrespond.Message(c, http.StatusUnauthorized, errrespond.CodeUnauthorized, "token inválido ou expirado")
			c.Abort()
			return
		}

		// Defesa em profundidade: garante que o token é desta aplicação.
		if opts.ApplicationID != "" && claims.ApplicationID != opts.ApplicationID {
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
		c.Set(CtxPerms, claims.Perms)
		c.Set(CtxAccessToken, rawJWT)
		c.Set(CtxAuthVia, via)
		// O ator também vai no context da REQUISIÇÃO (não só no do gin) para a
		// camada de aplicação registrar quem fez a ação (trilha de eventos)
		// sem precisar plumbar o user_id por todas as assinaturas de serviço.
		if id, err := uuid.Parse(strings.TrimSpace(claims.UserID)); err == nil {
			c.Request = c.Request.WithContext(appctx.WithActor(c.Request.Context(), id))
		}
		c.Next()
	}
}

// resolveCredential obtém o JWT a validar: do header Bearer ou da sessão do
// cookie. Em falha, escreve a resposta e devolve ok=false.
func resolveCredential(c *gin.Context, opts AuthOptions, allowed map[string]struct{}) (jwtRaw, via string, ok bool) {
	if raw := c.GetHeader("Authorization"); raw != "" {
		if opts.BearerDisabled {
			errrespond.Message(c, http.StatusUnauthorized, errrespond.CodeUnauthorized, "autenticação por Bearer desabilitada; use a sessão (cookie)")
			c.Abort()
			return "", "", false
		}
		parts := strings.SplitN(raw, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			errrespond.Message(c, http.StatusUnauthorized, errrespond.CodeUnauthorized, "formato inválido: use 'Authorization: Bearer <token>'")
			c.Abort()
			return "", "", false
		}
		return strings.TrimSpace(parts[1]), AuthViaBearer, true
	}

	cookieToken := opts.Cookie.Read(c)
	if opts.Sessions == nil || cookieToken == "" {
		errrespond.Message(c, http.StatusUnauthorized, errrespond.CodeUnauthorized, "token de autenticação ausente")
		c.Abort()
		return "", "", false
	}

	if !safeMethod(c.Request.Method) && crossSiteRequest(c.Request, allowed) {
		errrespond.Message(c, http.StatusForbidden, errrespond.CodeForbidden, "requisição cross-site recusada")
		c.Abort()
		return "", "", false
	}

	sess, err := opts.Sessions.Resolve(c.Request.Context(), cookieToken)
	if err != nil {
		switch {
		case errors.Is(err, domsess.ErrNotFound), errors.Is(err, domsess.ErrRefreshRejected):
			opts.Cookie.Clear(c)
			errrespond.Message(c, http.StatusUnauthorized, errrespond.CodeUnauthorized, "sessão expirada; faça login novamente")
		case errors.Is(err, domsess.ErrAuthUnavailable):
			errrespond.Message(c, http.StatusServiceUnavailable, errrespond.CodeInternal, "serviço de autenticação indisponível; tente novamente em instantes")
		default:
			errrespond.Message(c, http.StatusInternalServerError, errrespond.CodeInternal, "falha ao validar a sessão")
		}
		c.Abort()
		return "", "", false
	}
	c.Set(CtxSessionID, sess.ID)
	return sess.Tokens.Access, AuthViaCookie, true
}
