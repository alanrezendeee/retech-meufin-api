package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	app "github.com/retechfin/retechfin-api/internal/application/entitlement"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/errrespond"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/middleware"
)

// EntitlementHandler expõe o plano/cota do workspace (para a UI mostrar
// consumo de consultas fiscais verificadas e o degrau de plano).
type EntitlementHandler struct {
	svc *app.Service
}

func NewEntitlementHandler(svc *app.Service) *EntitlementHandler {
	return &EntitlementHandler{svc: svc}
}

type fiscalQuotaResponse struct {
	Tier      string `json:"tier"`
	Quota     int    `json:"quota"`
	Used      int    `json:"used"`
	Remaining int    `json:"remaining"`
}

type entitlementResponse struct {
	FiscalSEFAZ fiscalQuotaResponse `json:"fiscal_sefaz"`
}

// Get responde GET /finance/entitlements — tier e cota SEFAZ do mês corrente.
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
		FiscalSEFAZ: fiscalQuotaResponse{
			Tier:      string(st.Tier),
			Quota:     st.Quota,
			Used:      st.Used,
			Remaining: st.Remaining,
		},
	})
}
