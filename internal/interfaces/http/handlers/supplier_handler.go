package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	app "github.com/retechfin/retechfin-api/internal/application/finance"
	dom "github.com/retechfin/retechfin-api/internal/domain/finance"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/errrespond"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/middleware"
)

type SupplierHandler struct {
	svc *app.SupplierService
}

func NewSupplierHandler(svc *app.SupplierService) *SupplierHandler {
	return &SupplierHandler{svc: svc}
}

type supplierResponse struct {
	ID                 uuid.UUID  `json:"id"`
	WorkspaceID        *uuid.UUID `json:"workspace_id"`
	IsGlobal           bool       `json:"is_global"`
	Name               string     `json:"name"`
	Category           string     `json:"category"`
	DefaultBillingType *string    `json:"default_billing_type"`
	PixKey             *string    `json:"pix_key"`
	BankName           *string    `json:"bank_name"`
	BankAgency         *string    `json:"bank_agency"`
	BankAccount        *string    `json:"bank_account"`
	BankAccountType    *string    `json:"bank_account_type"`
	Notes              *string    `json:"notes"`
	Active             bool       `json:"active"`
	CreatedAt          string     `json:"created_at"`
	UpdatedAt          string     `json:"updated_at"`
}

func mapSupplier(s *dom.Supplier) supplierResponse {
	var billing *string
	if s.DefaultBillingType != nil {
		v := string(*s.DefaultBillingType)
		billing = &v
	}
	return supplierResponse{
		ID:                 s.ID,
		WorkspaceID:        s.WorkspaceID,
		IsGlobal:           s.IsGlobal(),
		Name:               s.Name,
		Category:           string(s.Category),
		DefaultBillingType: billing,
		PixKey:             s.PixKey,
		BankName:           s.BankName,
		BankAgency:         s.BankAgency,
		BankAccount:        s.BankAccount,
		BankAccountType:    s.BankAccountType,
		Notes:              s.Notes,
		Active:             s.Active,
		CreatedAt:          s.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:          s.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

type supplierCreateJSON struct {
	Name               string  `json:"name" binding:"required"`
	Category           string  `json:"category"`
	DefaultBillingType *string `json:"default_billing_type"`
	PixKey             *string `json:"pix_key"`
	BankName           *string `json:"bank_name"`
	BankAgency         *string `json:"bank_agency"`
	BankAccount        *string `json:"bank_account"`
	BankAccountType    *string `json:"bank_account_type"`
	Notes              *string `json:"notes"`
	Active             *bool   `json:"active"`
}

func (h *SupplierHandler) Create(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	var body supplierCreateJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido")
		return
	}
	sup, err := h.svc.Create(c.Request.Context(), app.CreateSupplierInput{
		WorkspaceID:        ws,
		Name:               body.Name,
		Category:           body.Category,
		DefaultBillingType: body.DefaultBillingType,
		PixKey:             body.PixKey,
		BankName:           body.BankName,
		BankAgency:         body.BankAgency,
		BankAccount:        body.BankAccount,
		BankAccountType:    body.BankAccountType,
		Notes:              body.Notes,
		Active:             body.Active,
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, mapSupplier(sup))
}

func (h *SupplierHandler) Get(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "id inválido")
		return
	}
	sup, err := h.svc.Get(c.Request.Context(), ws, id)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapSupplier(sup))
}

func (h *SupplierHandler) List(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	limit, offset := pagination(c)
	filter := dom.SupplierFilter{
		Query:    strings.TrimSpace(c.Query("query")),
		Category: c.Query("category"),
		Active:   boolQuery(c, "active"),
	}
	res, err := h.svc.List(c.Request.Context(), ws, filter, limit, offset)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	items := make([]supplierResponse, len(res.Items))
	for i := range res.Items {
		items[i] = mapSupplier(&res.Items[i])
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": res.Total})
}

type supplierUpdateJSON struct {
	Name               string  `json:"name" binding:"required"`
	Category           string  `json:"category"`
	DefaultBillingType *string `json:"default_billing_type"`
	PixKey             *string `json:"pix_key"`
	BankName           *string `json:"bank_name"`
	BankAgency         *string `json:"bank_agency"`
	BankAccount        *string `json:"bank_account"`
	BankAccountType    *string `json:"bank_account_type"`
	Notes              *string `json:"notes"`
	Active             *bool   `json:"active"`
}

func (h *SupplierHandler) Update(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "id inválido")
		return
	}
	var body supplierUpdateJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido")
		return
	}
	sup, err := h.svc.Update(c.Request.Context(), app.UpdateSupplierInput{
		WorkspaceID:        ws,
		ID:                 id,
		Name:               body.Name,
		Category:           body.Category,
		DefaultBillingType: body.DefaultBillingType,
		PixKey:             body.PixKey,
		BankName:           body.BankName,
		BankAgency:         body.BankAgency,
		BankAccount:        body.BankAccount,
		BankAccountType:    body.BankAccountType,
		Notes:              body.Notes,
		Active:             body.Active,
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapSupplier(sup))
}

func (h *SupplierHandler) Delete(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "id inválido")
		return
	}
	if err := h.svc.Delete(c.Request.Context(), ws, id); err != nil {
		errrespond.Write(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
