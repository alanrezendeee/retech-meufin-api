package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/errrespond"
)

// EnforcementMode controla o comportamento do RequireModule.
type EnforcementMode string

const (
	// EnforcementOff desativa a checagem (comportamento pré-perms).
	EnforcementOff EnforcementMode = "off"
	// EnforcementWarn permite e loga quando o token não passaria — modo de
	// transição enquanto tokens antigos (sem claim perms) circulam.
	EnforcementWarn EnforcementMode = "warn"
	// EnforcementStrict bloqueia com 403.
	EnforcementStrict EnforcementMode = "strict"
)

// EnforcementModeFromEnv lê PERMS_ENFORCEMENT (off|warn|strict; default warn).
func EnforcementModeFromEnv() EnforcementMode {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("PERMS_ENFORCEMENT"))) {
	case "off":
		return EnforcementOff
	case "strict":
		return EnforcementStrict
	default:
		return EnforcementWarn
	}
}

const masterPermCode = "all:manage"

// RequireModule autoriza o request quando o token carrega alguma permission do
// módulo (prefixo do subject, ex.: "finance" casa "finance.income:view") ou é
// master ("all:manage"). Deve rodar após RequireAuth.
//
// Tokens sem o claim perms (emitidos antes do deploy do auth com o claim):
//   - warn: permite e loga — janela de transição
//   - strict: 403 (exige relogin/refresh)
func RequireModule(module string, mode EnforcementMode) gin.HandlerFunc {
	prefix := module + "."
	return func(c *gin.Context) {
		if mode == EnforcementOff {
			c.Next()
			return
		}

		perms, _ := c.Get(CtxPerms)
		codes, _ := perms.([]string)

		if hasModuleAccess(codes, prefix) {
			c.Next()
			return
		}

		if mode == EnforcementWarn {
			slog.Warn("⚠️ autorização por módulo negaria este request (modo warn — permitido)",
				slog.String("module", module),
				slog.String("path", c.Request.URL.Path),
				slog.String("method", c.Request.Method),
				slog.Int("perms_no_token", len(codes)),
			)
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
