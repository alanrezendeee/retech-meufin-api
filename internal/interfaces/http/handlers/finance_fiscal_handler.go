package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	app "github.com/retechfin/retechfin-api/internal/application/finance"
	dom "github.com/retechfin/retechfin-api/internal/domain/finance"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/errrespond"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/middleware"
)

// FinanceFiscalHandler expõe o vínculo cupom/nota fiscal → despesa e o
// detalhamento item a item.
type FinanceFiscalHandler struct {
	svc *app.FiscalService
}

func NewFinanceFiscalHandler(svc *app.FiscalService) *FinanceFiscalHandler {
	return &FinanceFiscalHandler{svc: svc}
}

type fiscalItemJSON struct {
	Description   string  `json:"description" binding:"required"`
	QuantityMilli int64   `json:"quantity_milli"` // 1un = 1000; 0,455kg = 455
	UnitCents     int64   `json:"unit_cents"`
	AmountCents   int64   `json:"amount_cents" binding:"required"`
	Category      *string `json:"category"`
	// CategoryName/CategoryGroup acompanham categorias NOVAS (category_is_new)
	// para o auto-cadastro no save.
	CategoryName  *string `json:"category_name"`
	CategoryGroup *string `json:"category_group"`
}

type fiscalNewEntryJSON struct {
	Description    string     `json:"description" binding:"required"`
	AmountCents    int64      `json:"amount_cents" binding:"required"`
	DueDate        string     `json:"due_date" binding:"required"` // YYYY-MM-DD
	Status         string     `json:"status"`                      // prevista|realizada; default prevista
	Type           *string    `json:"type"`
	FamilyMemberID *uuid.UUID `json:"family_member_id"`
	SupplierID     *uuid.UUID `json:"supplier_id"`
	PurchaseDate   *string    `json:"purchase_date"` // YYYY-MM-DD
}

type fiscalConfirmJSON struct {
	EntryID  *uuid.UUID          `json:"entry_id"`
	NewEntry *fiscalNewEntryJSON `json:"new_entry"`
	Items    []fiscalItemJSON    `json:"items" binding:"required"`
}

// Confirm vincula o cupom (documento kind=fiscal) a uma despesa — existente
// (entry_id) ou criada agora (new_entry) — gravando os itens revisados.
func (h *FinanceFiscalHandler) Confirm(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	documentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "id inválido")
		return
	}
	var body fiscalConfirmJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido")
		return
	}

	in := app.FiscalConfirmInput{
		WorkspaceID: ws,
		DocumentID:  documentID,
		EntryID:     body.EntryID,
	}
	if body.EntryID == nil && body.NewEntry != nil {
		due, derr := time.Parse(entryDateLayout, body.NewEntry.DueDate)
		if derr != nil {
			errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "due_date inválida (use YYYY-MM-DD)")
			return
		}
		var purchaseDate *time.Time
		if body.NewEntry.PurchaseDate != nil && *body.NewEntry.PurchaseDate != "" {
			d, perr := time.Parse(entryDateLayout, *body.NewEntry.PurchaseDate)
			if perr != nil {
				errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "purchase_date inválida (use YYYY-MM-DD)")
				return
			}
			purchaseDate = &d
		}
		in.NewEntry = &app.CreateEntryInput{
			Status:         body.NewEntry.Status,
			AmountCents:    body.NewEntry.AmountCents,
			DueDate:        due,
			Type:           body.NewEntry.Type,
			Description:    body.NewEntry.Description,
			FamilyMemberID: body.NewEntry.FamilyMemberID,
			SupplierID:     body.NewEntry.SupplierID,
			PurchaseDate:   purchaseDate,
		}
	}
	for _, it := range body.Items {
		in.Items = append(in.Items, app.FiscalConfirmItem{
			Description:   it.Description,
			QuantityMilli: it.QuantityMilli,
			UnitCents:     it.UnitCents,
			AmountCents:   it.AmountCents,
			Category:      it.Category,
			CategoryName:  it.CategoryName,
			CategoryGroup: it.CategoryGroup,
		})
	}

	entry, items, createdCats, err := h.svc.Confirm(c.Request.Context(), in)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"entry":              mapFinancialEntry(entry),
		"items":              mapFiscalItems(items),
		"items_total":        len(items),
		"created_categories": mapCreatedCategories(createdCats),
	})
}

type createdCategoryResponse struct {
	Slug  string `json:"slug"`
	Name  string `json:"name"`
	Group string `json:"group"`
}

// mapCreatedCategories informa quais categorias foram auto-cadastradas neste
// save (para a UI avisar o usuário). Sempre não-nil (lista vazia em vez de null).
func mapCreatedCategories(cats []dom.ExpenseCategory) []createdCategoryResponse {
	out := make([]createdCategoryResponse, 0, len(cats))
	for _, c := range cats {
		out = append(out, createdCategoryResponse{Slug: c.Slug, Name: c.Name, Group: c.GroupSlug})
	}
	return out
}

// ListByEntry responde GET /entries/:id/fiscal-items.
func (h *FinanceFiscalHandler) ListByEntry(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	entryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "id inválido")
		return
	}
	items, err := h.svc.ListByEntry(c.Request.Context(), ws, entryID)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": mapFiscalItems(items), "total": len(items)})
}

type fiscalItemResponse struct {
	ID            uuid.UUID `json:"id"`
	EntryID       uuid.UUID `json:"entry_id"`
	DocumentID    uuid.UUID `json:"document_id"`
	Description   string    `json:"description"`
	QuantityMilli int64     `json:"quantity_milli"`
	UnitCents     int64     `json:"unit_cents"`
	AmountCents   int64     `json:"amount_cents"`
	Category      *string   `json:"category"`
}

func mapFiscalItems(items []dom.FiscalItem) []fiscalItemResponse {
	out := make([]fiscalItemResponse, len(items))
	for i, it := range items {
		out[i] = fiscalItemResponse{
			ID:            it.ID,
			EntryID:       it.EntryID,
			DocumentID:    it.DocumentID,
			Description:   it.Description,
			QuantityMilli: it.QuantityMilli,
			UnitCents:     it.UnitCents,
			AmountCents:   it.AmountCents,
			Category:      it.Category,
		}
	}
	return out
}
