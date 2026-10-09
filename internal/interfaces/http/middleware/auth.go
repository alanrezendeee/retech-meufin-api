package middleware

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/theretechlabs/retech-authkit/cookie"
	"github.com/theretechlabs/retech-authkit/csrf"
	"github.com/theretechlabs/retech-authkit/jwtverify"
	"github.com/theretechlabs/retech-authkit/session"

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
	CtxAccessToken = "access_token" // JWT do auth da sessão desta requisição (usado por /auth/me e proxy IAM)
	CtxSessionID   = "session_id"   // id (hash) da sessão
)

// AuthClaims espelha os claims emitidos pelo retech-auth-api (RS256).
type AuthClaims = jwtverify.Claims

// SessionResolver transforma o token do cookie em sessão com access token
// válido e verificado. Implementado por application/session.Service.
type SessionResolver interface {
	Resolve(ctx context.Context, rawToken string) (session.Resolved, error)
}

// AuthOptions configura RequireAuth.
type AuthOptions struct {
	// Sessions resolve o cookie em sessão (application/session.Service).
	Sessions SessionResolver
	Cookie   cookie.Config
	// AllowedOrigins alimenta a verificação CSRF das requisições por cookie
	// (normalmente a mesma lista do CORS).
	AllowedOrigins []string
}

// RequireAuth autentica a requisição pelo cookie de sessão do gateway — a
// ÚNICA forma de autenticação (não há `Authorization: Bearer`; JWT nunca sai
// do servidor). O cookie vira sessão → access token do auth (renovado se
// preciso) → JWT verificado (JWKS, iss, aud, typ, tenant) → workspace derivado
// do tenant_id. Requisições que mudam estado passam ainda pela verificação CSRF.
func RequireAuth(opts AuthOptions) gin.HandlerFunc {
	allowed := csrf.OriginSet(opts.AllowedOrigins)

	return func(c *gin.Context) {
		if c.GetHeader("Authorization") != "" {
			errrespond.Message(c, http.StatusUnauthorized, errrespond.CodeUnauthorized, "esta API não aceita Authorization: Bearer; autentique via POST /api/v1/auth/login (cookie de sessão)")
			c.Abort()
			return
		}

		cookieToken := opts.Cookie.Read(c.Request)
		if opts.Sessions == nil || cookieToken == "" {
			errrespond.Message(c, http.StatusUnauthorized, errrespond.CodeUnauthorized, "sessão ausente; faça login")
			c.Abort()
			return
		}

		if csrf.Reject(c.Request, allowed) {
			errrespond.Message(c, http.StatusForbidden, errrespond.CodeForbidden, "requisição cross-site recusada")
			c.Abort()
			return
		}

		res, err := opts.Sessions.Resolve(c.Request.Context(), cookieToken)
		if err != nil {
			switch {
			case errors.Is(err, domsess.ErrNotFound), errors.Is(err, domsess.ErrRefreshRejected), errors.Is(err, domsess.ErrTokenInvalid):
				opts.Cookie.Clear(c.Writer)
				errrespond.Message(c, http.StatusUnauthorized, errrespond.CodeUnauthorized, "sessão expirada; faça login novamente")
			case errors.Is(err, domsess.ErrAuthUnavailable):
				errrespond.Message(c, http.StatusServiceUnavailable, errrespond.CodeInternal, "serviço de autenticação indisponível; tente novamente em instantes")
			default:
				errrespond.Message(c, http.StatusInternalServerError, errrespond.CodeInternal, "falha ao validar a sessão")
			}
			c.Abort()
			return
		}
		claims := res.Claims
		if claims == nil {
			errrespond.Message(c, http.StatusInternalServerError, errrespond.CodeInternal, "sessão sem token verificado")
			c.Abort()
			return
		}

		// Workspace = tenant_id do token (o kit já exigiu UUID válido). Header X-Workspace-ID é ignorado.
		ws := claims.TenantUUID()
		if ws == uuid.Nil {
			errrespond.Message(c, http.StatusForbidden, errrespond.CodeForbidden, "usuário sem workspace (tenant_id) no token")
			c.Abort()
			return
		}

		c.Set(CtxSessionID, res.ID)
		c.Set(CtxWorkspaceID, ws)
		c.Set(CtxUserID, claims.UserID.String())
		c.Set(CtxEmail, claims.Email)
		c.Set(CtxRoles, claims.Roles)
		c.Set(CtxPerms, claims.Perms)
		c.Set(CtxAccessToken, res.Tokens.Access)
		// O ator também vai no context da REQUISIÇÃO (não só no do gin) para a
		// camada de aplicação registrar quem fez a ação (trilha de eventos).
		c.Request = c.Request.WithContext(appctx.WithActor(c.Request.Context(), claims.UserID))
		c.Next()
	}
}

// CSRF aplica a verificação cross-site a uma rota fora do RequireAuth que usa
// o cookie (ex.: logout).
func CSRF(allowedOrigins []string) gin.HandlerFunc {
	allowed := csrf.OriginSet(allowedOrigins)
	return func(c *gin.Context) {
		if csrf.Reject(c.Request, allowed) {
			errrespond.Message(c, http.StatusForbidden, errrespond.CodeForbidden, "requisição cross-site recusada")
			c.Abort()
			return
		}
		c.Next()
	}
}
