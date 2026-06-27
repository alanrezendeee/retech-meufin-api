package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CtxWorkspaceID é a chave de contexto do workspace, preenchida por RequireAuth
// a partir do tenant_id do token validado.
const CtxWorkspaceID = "workspace_id"

// WorkspaceID lê o workspace derivado do token no contexto da request.
func WorkspaceID(c *gin.Context) (uuid.UUID, bool) {
	v, ok := c.Get(CtxWorkspaceID)
	if !ok {
		return uuid.Nil, false
	}
	id, ok := v.(uuid.UUID)
	return id, ok
}
