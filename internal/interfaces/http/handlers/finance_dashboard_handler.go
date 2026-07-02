package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	app "github.com/retechfin/retechfin-api/internal/application/finance"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/errrespond"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/middleware"
)

// FinanceDashboardHandler expõe os agregados da dashboard financeira.
// Todos os valores em cents; o front formata.
type FinanceDashboardHandler struct {
	svc *app.FinanceDashboardService
}

func NewFinanceDashboardHandler(svc *app.FinanceDashboardService) *FinanceDashboardHandler {
	return &FinanceDashboardHandler{svc: svc}
}

type dashboardCategoryJSON struct {
	Category   string `json:"category"`
	TotalCents int64  `json:"total_cents"`
}

type dashboardSummaryJSON struct {
	Year                 int                     `json:"year"`
	Month                int                     `json:"month"`
	IncomeRealizedCents  int64                   `json:"income_realized_cents"`
	IncomeExpectedCents  int64                   `json:"income_expected_cents"`
	ExpenseRealizedCents int64                   `json:"expense_realized_cents"`
	ExpenseExpectedCents int64                   `json:"expense_expected_cents"`
	BalanceRealizedCents int64                   `json:"balance_realized_cents"`
	BalanceExpectedCents int64                   `json:"balance_expected_cents"`
	ReceivableCents      int64                   `json:"receivable_cents"`
	PayableCents         int64                   `json:"payable_cents"`
	Categories           []dashboardCategoryJSON `json:"categories"`
	FutureInstallments   struct {
		TotalCents  int64   `json:"total_cents"`
		Count       int64   `json:"count"`
		LastDueDate *string `json:"last_due_date"`
	} `json:"future_installments"`
}

// Summary responde GET /finance/dashboard?year=&month=&family_member_id=.
// year/month default: mês corrente.
func (h *FinanceDashboardHandler) Summary(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	now := time.Now().UTC()
	year, month := now.Year(), int(now.Month())
	var err error
	if v := c.Query("year"); v != "" {
		if year, err = strconv.Atoi(v); err != nil {
			errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "year inválido")
			return
		}
	}
	if v := c.Query("month"); v != "" {
		if month, err = strconv.Atoi(v); err != nil {
			errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "month inválido")
			return
		}
	}
	var familyMemberID *uuid.UUID
	if v := c.Query("family_member_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "family_member_id inválido")
			return
		}
		familyMemberID = &id
	}

	sum, err := h.svc.Summary(c.Request.Context(), ws, year, month, familyMemberID)
	if err != nil {
		errrespond.Write(c, err)
		return
	}

	resp := dashboardSummaryJSON{
		Year:                 year,
		Month:                month,
		IncomeRealizedCents:  sum.IncomeRealizedCents,
		IncomeExpectedCents:  sum.IncomeExpectedCents,
		ExpenseRealizedCents: sum.ExpenseRealizedCents,
		ExpenseExpectedCents: sum.ExpenseExpectedCents,
		BalanceRealizedCents: sum.IncomeRealizedCents - sum.ExpenseRealizedCents,
		BalanceExpectedCents: sum.IncomeExpectedCents - sum.ExpenseExpectedCents,
		ReceivableCents:      sum.ReceivableCents,
		PayableCents:         sum.PayableCents,
		Categories:           make([]dashboardCategoryJSON, len(sum.Categories)),
	}
	for i := range sum.Categories {
		resp.Categories[i] = dashboardCategoryJSON{
			Category:   sum.Categories[i].Category,
			TotalCents: sum.Categories[i].TotalCents,
		}
	}
	resp.FutureInstallments.TotalCents = sum.FutureInstallments.TotalCents
	resp.FutureInstallments.Count = sum.FutureInstallments.Count
	if sum.FutureInstallments.LastDueDate != nil {
		v := sum.FutureInstallments.LastDueDate.Format(entryDateLayout)
		resp.FutureInstallments.LastDueDate = &v
	}
	c.JSON(http.StatusOK, resp)
}

type dashboardMonthJSON struct {
	Month                int   `json:"month"`
	IncomeRealizedCents  int64 `json:"income_realized_cents"`
	IncomeExpectedCents  int64 `json:"income_expected_cents"`
	ExpenseRealizedCents int64 `json:"expense_realized_cents"`
	ExpenseExpectedCents int64 `json:"expense_expected_cents"`
	BalanceExpectedCents int64 `json:"balance_expected_cents"`
}

// Monthly responde GET /finance/dashboard/monthly?year=&family_member_id=.
// year default: ano corrente. Sempre 12 meses.
func (h *FinanceDashboardHandler) Monthly(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	year := time.Now().UTC().Year()
	var err error
	if v := c.Query("year"); v != "" {
		if year, err = strconv.Atoi(v); err != nil {
			errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "year inválido")
			return
		}
	}
	var familyMemberID *uuid.UUID
	if v := c.Query("family_member_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "family_member_id inválido")
			return
		}
		familyMemberID = &id
	}

	points, err := h.svc.MonthlySeries(c.Request.Context(), ws, year, familyMemberID)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	items := make([]dashboardMonthJSON, len(points))
	for i, p := range points {
		items[i] = dashboardMonthJSON{
			Month:                p.Month,
			IncomeRealizedCents:  p.IncomeRealizedCents,
			IncomeExpectedCents:  p.IncomeExpectedCents,
			ExpenseRealizedCents: p.ExpenseRealizedCents,
			ExpenseExpectedCents: p.ExpenseExpectedCents,
			BalanceExpectedCents: p.IncomeExpectedCents - p.ExpenseExpectedCents,
		}
	}
	c.JSON(http.StatusOK, gin.H{"year": year, "months": items})
}
