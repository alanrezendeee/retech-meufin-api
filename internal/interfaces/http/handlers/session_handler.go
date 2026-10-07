package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	appsess "github.com/retechfin/retechfin-api/internal/application/session"
	domsess "github.com/retechfin/retechfin-api/internal/domain/session"
	"github.com/retechfin/retechfin-api/internal/infrastructure/authclient"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/errrespond"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/middleware"
)

// SessionHandler expõe o gateway de autenticação do browser:
// POST /auth/login, POST /auth/logout (cookie HttpOnly) e GET /auth/me.
type SessionHandler struct {
	svc    *appsess.Service // nil = gateway desabilitado (SESSION_ENCRYPTION_KEY ausente)
	auth   *authclient.PublicAuthenticator
	cookie middleware.SessionCookie
}

func NewSessionHandler(svc *appsess.Service, auth *authclient.PublicAuthenticator, cookie middleware.SessionCookie) *SessionHandler {
	return &SessionHandler{svc: svc, auth: auth, cookie: cookie}
}

type loginJSON struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

// Login autentica no auth central e emite o cookie de sessão. 204 sem corpo:
// o admin chama GET /auth/me em seguida para obter usuário e abilities.
func (h *SessionHandler) Login(c *gin.Context) {
	if h.svc == nil {
		errrespond.Message(c, http.StatusServiceUnavailable, errrespond.CodeInternal, "login por sessão desabilitado neste ambiente")
		return
	}
	var body loginJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeValidation, "informe e-mail válido e senha (mínimo 6 caracteres)")
		return
	}
	meta := appsess.ClientMeta{IP: c.ClientIP(), UserAgent: c.Request.UserAgent()}
	raw, _, err := h.svc.Login(c.Request.Context(), strings.TrimSpace(body.Email), body.Password, meta)
	switch {
	case err == nil:
		h.cookie.Set(c, raw)
		c.Status(http.StatusNoContent)
	case errors.Is(err, domsess.ErrInvalidCredentials):
		errrespond.Message(c, http.StatusUnauthorized, errrespond.CodeUnauthorized, "e-mail ou senha incorretos")
	case errors.Is(err, domsess.ErrUserInactive):
		errrespond.Message(c, http.StatusForbidden, errrespond.CodeForbidden, "usuário ou aplicação inativos")
	case errors.Is(err, domsess.ErrAuthUnavailable):
		errrespond.Message(c, http.StatusServiceUnavailable, errrespond.CodeInternal, "serviço de autenticação indisponível; tente novamente em instantes")
	default:
		errrespond.Message(c, http.StatusInternalServerError, errrespond.CodeInternal, "não foi possível autenticar agora")
	}
}

// Logout revoga a sessão e limpa o cookie. Idempotente.
func (h *SessionHandler) Logout(c *gin.Context) {
	raw := h.cookie.Read(c)
	if h.svc != nil && raw != "" {
		if err := h.svc.Logout(c.Request.Context(), raw); err != nil {
			errrespond.Message(c, http.StatusInternalServerError, errrespond.CodeInternal, "não foi possível encerrar a sessão")
			return
		}
	}
	h.cookie.Clear(c)
	c.Status(http.StatusNoContent)
}

// Me repassa GET /v1/me do auth (usuário, roles, permissions, abilities CASL)
// usando o access token da requisição — Bearer ou sessão. Protegido por
// RequireAuth.
func (h *SessionHandler) Me(c *gin.Context) {
	token := c.GetString(middleware.CtxAccessToken)
	if token == "" {
		errrespond.Message(c, http.StatusUnauthorized, errrespond.CodeUnauthorized, "token de autenticação ausente")
		return
	}
	status, body, err := h.auth.Me(c.Request.Context(), token)
	if err != nil {
		errrespond.Message(c, http.StatusServiceUnavailable, errrespond.CodeInternal, "serviço de autenticação indisponível; tente novamente em instantes")
		return
	}
	switch status {
	case http.StatusOK:
		c.Data(http.StatusOK, "application/json; charset=utf-8", body)
	case http.StatusUnauthorized, http.StatusForbidden:
		if c.GetString(middleware.CtxAuthVia) == middleware.AuthViaCookie {
			h.cookie.Clear(c)
		}
		errrespond.Message(c, http.StatusUnauthorized, errrespond.CodeUnauthorized, "sessão inválida; faça login novamente")
	default:
		errrespond.Message(c, http.StatusBadGateway, errrespond.CodeInternal, "auth respondeu com erro ao carregar o perfil")
	}
}
