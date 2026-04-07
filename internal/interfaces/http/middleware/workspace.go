package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/errrespond"
)

const HeaderWorkspaceID = "X-Workspace-ID"
const CtxWorkspaceID = "workspace_id"

func RequireWorkspace() gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := c.GetHeader(HeaderWorkspaceID)
		if raw == "" {
			errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "cabeçalho X-Workspace-ID é obrigatório")
			c.Abort()
			return
		}
		id, err := uuid.Parse(raw)
		if err != nil {
			errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "X-Workspace-ID deve ser um UUID válido")
			c.Abort()
			return
		}
		c.Set(CtxWorkspaceID, id)
		c.Next()
	}
}

func WorkspaceID(c *gin.Context) (uuid.UUID, bool) {
	v, ok := c.Get(CtxWorkspaceID)
	if !ok {
		return uuid.Nil, false
	}
	id, ok := v.(uuid.UUID)
	return id, ok
}
