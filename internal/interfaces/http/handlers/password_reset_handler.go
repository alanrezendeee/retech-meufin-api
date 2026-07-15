package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	appacc "github.com/retechfin/retechfin-api/internal/application/account"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/errrespond"
)

// PasswordResetHandler expõe o fluxo público "esqueci a senha".
// Rotas SEM RequireAuth — o usuário está deslogado.
type PasswordResetHandler struct {
	svc *appacc.PasswordResetService
}

func NewPasswordResetHandler(svc *appacc.PasswordResetService) *PasswordResetHandler {
	return &PasswordResetHandler{svc: svc}
}

type passwordResetRequestJSON struct {
	Email string `json:"email" binding:"required,email"`
}

type passwordResetConfirmJSON struct {
	Token    string `json:"token" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Request dispara o e-mail de redefinição. Responde SEMPRE 202 com mensagem
// genérica quando o input é válido — nunca revela se o e-mail existe.
func (h *PasswordResetHandler) Request(c *gin.Context) {
	var body passwordResetRequestJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeValidation, "informe um e-mail válido")
		return
	}

	if err := h.svc.Request(c.Request.Context(), body.Email); err != nil {
		errrespond.Message(c, http.StatusInternalServerError, errrespond.CodeInternal,
			"não foi possível processar a solicitação agora; tente novamente em instantes")
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message": "se este e-mail estiver cadastrado, você receberá um link de redefinição em instantes",
	})
}

// Confirm troca a senha a partir do token recebido por e-mail.
func (h *PasswordResetHandler) Confirm(c *gin.Context) {
	var body passwordResetConfirmJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeValidation, "token e password são obrigatórios")
		return
	}

	err := h.svc.Confirm(c.Request.Context(), body.Token, body.Password)
	switch {
	case err == nil:
		c.JSON(http.StatusOK, gin.H{"message": "senha redefinida com sucesso; faça login com a nova senha"})
	case errors.Is(err, appacc.ErrWeakPassword):
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeValidation, err.Error())
	case errors.Is(err, appacc.ErrTokenInvalid):
		errrespond.Message(c, http.StatusUnprocessableEntity, errrespond.CodeValidation,
			"link inválido, expirado ou já utilizado — solicite uma nova redefinição")
	default:
		errrespond.Message(c, http.StatusInternalServerError, errrespond.CodeInternal,
			"não foi possível redefinir a senha agora; tente novamente em instantes")
	}
}
