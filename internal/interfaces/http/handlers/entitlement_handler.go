package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	app "github.com/retechfin/retechfin-api/internal/application/entitlement"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/errrespond"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/middleware"
)

// EntitlementHandler expõe o plano/cota do workspace (para a UI mostrar
// "X de Y consultas fiscais verificadas usadas neste mês" e o degrau de plano).
type EntitlementHandler struct {
	svc *app.Service
}

func NewEntitlementHandler(svc *app.Service) *EntitlementHandler {
	return &EntitlementHandler{svc: svc}
}

// fiscalVerificationUsage é o contador de consultas fiscais verificadas do mês
// para a tenant. Rótulo neutro de propósito: não expõe o fornecedor (Infosimples)
// nem detalhe de infraestrutura — só o que o usuário precisa ver (usado/limite).
type fiscalVerificationUsage struct {
	Tier      string `json:"tier"`
	Limit     int    `json:"limit"`
	Used      int    `json:"used"`
	Remaining int    `json:"remaining"`
	// Period identifica o mês do contador (AAAA-MM); o uso zera a cada mês.
	Period string `json:"period"`
}

type entitlementResponse struct {
	FiscalVerification fiscalVerificationUsage `json:"fiscal_verification"`
}

// Get responde GET /finance/entitlements — uso de consultas fiscais verificadas
// da tenant no mês corrente (usado/limite/restante) e o tier atual.
func (h *EntitlementHandler) Get(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	st, err := h.svc.FiscalStatusFor(c.Request.Context(), ws)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, entitlementResponse{
		FiscalVerification: fiscalVerificationUsage{
			Tier:      string(st.Tier),
			Limit:     st.Quota,
			Used:      st.Used,
			Remaining: st.Remaining,
			Period:    st.Period,
		},
	})
}
