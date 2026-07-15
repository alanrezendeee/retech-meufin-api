package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	appacc "github.com/retechfin/retechfin-api/internal/application/account"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/errrespond"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/middleware"
)

// MeHandler expõe o perfil do usuário logado (avatar).
type MeHandler struct {
	svc *appacc.ProfileService
}

func NewMeHandler(svc *appacc.ProfileService) *MeHandler {
	return &MeHandler{svc: svc}
}

type avatarURLResponse struct {
	AvatarURL *string `json:"avatar_url"`
}

// UploadAvatar recebe a foto de perfil (multipart, campo 'file').
func (h *MeHandler) UploadAvatar(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	userID, ok := userIDFromCtx(c)
	if !ok {
		errrespond.Message(c, http.StatusUnauthorized, errrespond.CodeUnauthorized, "usuário inválido no token")
		return
	}
	fileHeader, err := c.FormFile("file")
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "campo 'file' obrigatório")
		return
	}
	f, err := fileHeader.Open()
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "não foi possível ler o arquivo")
		return
	}
	defer f.Close()

	url, err := h.svc.UploadAvatar(c.Request.Context(), appacc.UploadAvatarInput{
		UserID:      userID,
		WorkspaceID: ws,
		MimeType:    fileHeader.Header.Get("Content-Type"),
		Size:        fileHeader.Size,
		Content:     f,
	})
	if err != nil {
		if appacc.IsProfileValidation(err) {
			errrespond.Message(c, http.StatusBadRequest, errrespond.CodeValidation, err.Error())
			return
		}
		errrespond.Message(c, http.StatusInternalServerError, errrespond.CodeInternal, err.Error())
		return
	}
	var out *string
	if url != "" {
		out = &url
	}
	c.JSON(http.StatusOK, avatarURLResponse{AvatarURL: out})
}

// AvatarURL retorna a URL presignada (15min) do avatar do usuário, ou null.
func (h *MeHandler) AvatarURL(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		errrespond.Message(c, http.StatusUnauthorized, errrespond.CodeUnauthorized, "usuário inválido no token")
		return
	}
	url, err := h.svc.AvatarURL(c.Request.Context(), userID)
	if err != nil {
		errrespond.Message(c, http.StatusInternalServerError, errrespond.CodeInternal, err.Error())
		return
	}
	var out *string
	if url != "" {
		out = &url
	}
	c.JSON(http.StatusOK, avatarURLResponse{AvatarURL: out})
}

// DeleteAvatar remove a foto de perfil (não apaga o objeto do storage).
func (h *MeHandler) DeleteAvatar(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		errrespond.Message(c, http.StatusUnauthorized, errrespond.CodeUnauthorized, "usuário inválido no token")
		return
	}
	if err := h.svc.RemoveAvatar(c.Request.Context(), userID); err != nil {
		errrespond.Message(c, http.StatusInternalServerError, errrespond.CodeInternal, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}
