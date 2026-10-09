package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/errrespond"
)

const masterPermCode = "all:manage"

// RequireModule autoriza o request quando o token carrega alguma permission do
// módulo (prefixo do subject, ex.: "finance" casa "finance.income:view") ou é
// master ("all:manage"). Deve rodar após RequireAuth. Sem permission → 403.
func RequireModule(module string) gin.HandlerFunc {
	prefix := module + "."
	return func(c *gin.Context) {
		perms, _ := c.Get(CtxPerms)
		codes, _ := perms.([]string)
		if hasModuleAccess(codes, prefix) {
			c.Next()
			return
		}
		errrespond.Message(c, http.StatusForbidden, errrespond.CodeForbidden,
			fmt.Sprintf("sem permissão para o módulo %s", module))
		c.Abort()
	}
}

func hasModuleAccess(codes []string, prefix string) bool {
	for _, code := range codes {
		if code == masterPermCode || strings.HasPrefix(code, prefix) {
			return true
		}
	}
	return false
}
