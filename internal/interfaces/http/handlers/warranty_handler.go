package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	app "github.com/retechfin/retechfin-api/internal/application/warranty"
	dom "github.com/retechfin/retechfin-api/internal/domain/warranty"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/errrespond"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/middleware"
)

const warrantyDateLayout = "2006-01-02"

// WarrantyHandler agrupa os endpoints do módulo de garantias de bens.
type WarrantyHandler struct {
	svc    *app.Service
	docSvc *app.DocumentService
}

func NewWarrantyHandler(svc *app.Service, docSvc *app.DocumentService) *WarrantyHandler {
	return &WarrantyHandler{svc: svc, docSvc: docSvc}
}

// ─── Response types ────────────────────────────────────────────────────────────

type warrantyResponse struct {
	ID                        string  `json:"id"`
	ItemName                  string  `json:"item_name"`
	Category                  string  `json:"category"`
	Brand                     *string `json:"brand"`
	Model                     *string `json:"model"`
	SerialNumber              *string `json:"serial_number"`
	Store                     *string `json:"store"`
	SupplierName              *string `json:"supplier_name"`
	PurchaseDate              string  `json:"purchase_date"`
	PriceCents                *int64  `json:"price_cents"`
	InvoiceNumber             *string `json:"invoice_number"`
	EntryID                   *string `json:"entry_id"`
	FiscalItemID              *string `json:"fiscal_item_id"`
	LegalWarrantyDays         int     `json:"legal_warranty_days"`
	ContractualWarrantyMonths int     `json:"contractual_warranty_months"`
	ExtendedWarrantyMonths    int     `json:"extended_warranty_months"`
	ExtendedProvider          *string `json:"extended_provider"`
	ExtendedCostCents         int64   `json:"extended_cost_cents"`
	CoverageNotes             *string `json:"coverage_notes"`
	Notes                     *string `json:"notes"`
	Active                    bool    `json:"active"`
	// Campos calculados
	ExpiresAt     string `json:"expires_at"`
	Status        string `json:"status"`
	DaysRemaining int    `json:"days_remaining"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

func mapWarranty(w *dom.Warranty) warrantyResponse {
	now := time.Now().UTC()
	r := warrantyResponse{
		ID:                        w.ID.String(),
		ItemName:                  w.ItemName,
		Category:                  string(w.Category),
		Brand:                     w.Brand,
		Model:                     w.Model,
		SerialNumber:              w.SerialNumber,
		Store:                     w.Store,
		SupplierName:              w.SupplierName,
		PurchaseDate:              w.PurchaseDate.Format(warrantyDateLayout),
		PriceCents:                w.PriceCents,
		InvoiceNumber:             w.InvoiceNumber,
		LegalWarrantyDays:         w.LegalWarrantyDays,
		ContractualWarrantyMonths: w.ContractualWarrantyMonths,
		ExtendedWarrantyMonths:    w.ExtendedWarrantyMonths,
		ExtendedProvider:          w.ExtendedProvider,
		ExtendedCostCents:         w.ExtendedCostCents,
		CoverageNotes:             w.CoverageNotes,
		Notes:                     w.Notes,
		Active:                    w.Active,
		ExpiresAt:                 w.ExpiresAt().Format(warrantyDateLayout),
		Status:                    string(w.StatusAt(now)),
		DaysRemaining:             w.DaysRemaining(now),
		CreatedAt:                 w.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:                 w.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if w.EntryID != nil {
		s := w.EntryID.String()
		r.EntryID = &s
	}
	if w.FiscalItemID != nil {
		s := w.FiscalItemID.String()
		r.FiscalItemID = &s
	}
	return r
}

type warrantyDocumentResponse struct {
	ID               string  `json:"id"`
	WarrantyID       string  `json:"warranty_id"`
	DocType          string  `json:"doc_type"`
	FileName         string  `json:"file_name"`
	OriginalFileName string  `json:"original_file_name"`
	ContentType      string  `json:"content_type"`
	SizeBytes        int64   `json:"size_bytes"`
	Notes            *string `json:"notes"`
	CreatedAt        string  `json:"created_at"`
}

func mapWarrantyDocument(d *dom.Document) warrantyDocumentResponse {
	return warrantyDocumentResponse{
		ID:               d.ID.String(),
		WarrantyID:       d.WarrantyID.String(),
		DocType:          string(d.DocType),
		FileName:         d.FileName,
		OriginalFileName: d.OriginalFileName,
		ContentType:      d.ContentType,
		SizeBytes:        d.SizeBytes,
		Notes:            d.Notes,
		CreatedAt:        d.CreatedAt.UTC().Format(time.RFC3339),
	}
}

type warrantySummaryResponse struct {
	TotalActive       int                    `json:"total_active"`
	TotalCoveredCents int64                  `json:"total_covered_cents"`
	ExpiringIn30Count int                    `json:"expiring_in_30_count"`
	ExpiringIn60Count int                    `json:"expiring_in_60_count"`
	ExpiringIn90Count int                    `json:"expiring_in_90_count"`
	ExpiredThisYear   int                    `json:"expired_this_year"`
	ExpiringSoon      []summaryExpiringItem  `json:"expiring_soon"`
	ByCategory        []summaryCategoryCount `json:"by_category"`
}

type summaryExpiringItem struct {
	ID            string `json:"id"`
	ItemName      string `json:"item_name"`
	Category      string `json:"category"`
	ExpiresAt     string `json:"expires_at"`
	DaysRemaining int    `json:"days_remaining"`
}

type summaryCategoryCount struct {
	Category string `json:"category"`
	Count    int    `json:"count"`
}

// ─── Input types ──────────────────────────────────────────────────────────────

type warrantyJSON struct {
	ItemName                  string  `json:"item_name" binding:"required"`
	Category                  string  `json:"category"`
	Brand                     *string `json:"brand"`
	Model                     *string `json:"model"`
	SerialNumber              *string `json:"serial_number"`
	Store                     *string `json:"store"`
	SupplierName              *string `json:"supplier_name"`
	PurchaseDate              string  `json:"purchase_date" binding:"required"`
	PriceCents                *int64  `json:"price_cents"`
	InvoiceNumber             *string `json:"invoice_number"`
	EntryID                   *string `json:"entry_id"`
	FiscalItemID              *string `json:"fiscal_item_id"`
	LegalWarrantyDays         *int    `json:"legal_warranty_days"`
	ContractualWarrantyMonths *int    `json:"contractual_warranty_months"`
	ExtendedWarrantyMonths    *int    `json:"extended_warranty_months"`
	ExtendedProvider          *string `json:"extended_provider"`
	ExtendedCostCents         *int64  `json:"extended_cost_cents"`
	CoverageNotes             *string `json:"coverage_notes"`
	Notes                     *string `json:"notes"`
	Active                    *bool   `json:"active"`
}

// ─── Handlers: warranties ─────────────────────────────────────────────────────

func (h *WarrantyHandler) List(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	result, err := h.svc.List(c.Request.Context(), ws, dom.ListParams{
		Category: c.Query("category"),
		Status:   c.Query("status"),
		Query:    c.Query("q"),
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	items := make([]warrantyResponse, len(result.Items))
	for i := range result.Items {
		items[i] = mapWarranty(&result.Items[i])
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": result.Total})
}

func (h *WarrantyHandler) Create(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	var body warrantyJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido: "+err.Error())
		return
	}
	purchaseDate, err := time.Parse(warrantyDateLayout, body.PurchaseDate)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "purchase_date inválida (use YYYY-MM-DD)")
		return
	}
	entryID, err := parseOptionalUUID(body.EntryID)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "entry_id inválido")
		return
	}
	fiscalItemID, err := parseOptionalUUID(body.FiscalItemID)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "fiscal_item_id inválido")
		return
	}
	w, err := h.svc.Create(c.Request.Context(), app.CreateInput{
		WorkspaceID:               ws,
		ItemName:                  body.ItemName,
		Category:                  body.Category,
		Brand:                     body.Brand,
		Model:                     body.Model,
		SerialNumber:              body.SerialNumber,
		Store:                     body.Store,
		SupplierName:              body.SupplierName,
		PurchaseDate:              purchaseDate,
		PriceCents:                body.PriceCents,
		InvoiceNumber:             body.InvoiceNumber,
		EntryID:                   entryID,
		FiscalItemID:              fiscalItemID,
		LegalWarrantyDays:         body.LegalWarrantyDays,
		ContractualWarrantyMonths: body.ContractualWarrantyMonths,
		ExtendedWarrantyMonths:    body.ExtendedWarrantyMonths,
		ExtendedProvider:          body.ExtendedProvider,
		ExtendedCostCents:         body.ExtendedCostCents,
		CoverageNotes:             body.CoverageNotes,
		Notes:                     body.Notes,
		Active:                    body.Active,
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, mapWarranty(w))
}

func (h *WarrantyHandler) Get(c *gin.Context) {
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
	w, err := h.svc.Get(c.Request.Context(), ws, id)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapWarranty(w))
}

func (h *WarrantyHandler) Update(c *gin.Context) {
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
	var body warrantyJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "JSON inválido: "+err.Error())
		return
	}
	purchaseDate, err := time.Parse(warrantyDateLayout, body.PurchaseDate)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "purchase_date inválida (use YYYY-MM-DD)")
		return
	}
	entryID, err := parseOptionalUUID(body.EntryID)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "entry_id inválido")
		return
	}
	fiscalItemID, err := parseOptionalUUID(body.FiscalItemID)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "fiscal_item_id inválido")
		return
	}
	w, err := h.svc.Update(c.Request.Context(), app.UpdateInput{
		WorkspaceID:               ws,
		ID:                        id,
		ItemName:                  body.ItemName,
		Category:                  body.Category,
		Brand:                     body.Brand,
		Model:                     body.Model,
		SerialNumber:              body.SerialNumber,
		Store:                     body.Store,
		SupplierName:              body.SupplierName,
		PurchaseDate:              purchaseDate,
		PriceCents:                body.PriceCents,
		InvoiceNumber:             body.InvoiceNumber,
		EntryID:                   entryID,
		FiscalItemID:              fiscalItemID,
		LegalWarrantyDays:         body.LegalWarrantyDays,
		ContractualWarrantyMonths: body.ContractualWarrantyMonths,
		ExtendedWarrantyMonths:    body.ExtendedWarrantyMonths,
		ExtendedProvider:          body.ExtendedProvider,
		ExtendedCostCents:         body.ExtendedCostCents,
		CoverageNotes:             body.CoverageNotes,
		Notes:                     body.Notes,
		Active:                    body.Active,
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, mapWarranty(w))
}

func (h *WarrantyHandler) Delete(c *gin.Context) {
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

// ─── Handler: summary ─────────────────────────────────────────────────────────

func (h *WarrantyHandler) Summary(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	sum, err := h.svc.Summary(c.Request.Context(), ws)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	resp := warrantySummaryResponse{
		TotalActive:       sum.TotalActive,
		TotalCoveredCents: sum.TotalCoveredCents,
		ExpiringIn30Count: sum.ExpiringIn30Count,
		ExpiringIn60Count: sum.ExpiringIn60Count,
		ExpiringIn90Count: sum.ExpiringIn90Count,
		ExpiredThisYear:   sum.ExpiredThisYear,
	}
	resp.ExpiringSoon = make([]summaryExpiringItem, len(sum.ExpiringSoon))
	for i, it := range sum.ExpiringSoon {
		resp.ExpiringSoon[i] = summaryExpiringItem{
			ID:            it.ID.String(),
			ItemName:      it.ItemName,
			Category:      string(it.Category),
			ExpiresAt:     it.ExpiresAt.Format(warrantyDateLayout),
			DaysRemaining: it.DaysRemaining,
		}
	}
	resp.ByCategory = make([]summaryCategoryCount, len(sum.ByCategory))
	for i, cc := range sum.ByCategory {
		resp.ByCategory[i] = summaryCategoryCount{Category: string(cc.Category), Count: cc.Count}
	}
	c.JSON(http.StatusOK, resp)
}

// ─── Handlers: documents ──────────────────────────────────────────────────────

func (h *WarrantyHandler) UploadDocument(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	warrantyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "id inválido")
		return
	}
	docType := c.PostForm("doc_type")
	var notes *string
	if v := c.PostForm("notes"); v != "" {
		notes = &v
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

	doc, err := h.docSvc.Upload(c.Request.Context(), app.UploadDocumentInput{
		WorkspaceID:      ws,
		WarrantyID:       warrantyID,
		DocType:          docType,
		Notes:            notes,
		OriginalFileName: fileHeader.Filename,
		ContentType:      fileHeader.Header.Get("Content-Type"),
		Size:             fileHeader.Size,
		Content:          f,
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, mapWarrantyDocument(doc))
}

func (h *WarrantyHandler) ListDocuments(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	warrantyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "id inválido")
		return
	}
	docs, err := h.docSvc.List(c.Request.Context(), ws, warrantyID)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	items := make([]warrantyDocumentResponse, len(docs))
	for i := range docs {
		items[i] = mapWarrantyDocument(&docs[i])
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *WarrantyHandler) DocumentDownloadURL(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	docID, err := uuid.Parse(c.Param("docId"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "docId inválido")
		return
	}
	url, err := h.docSvc.DownloadURL(c.Request.Context(), ws, docID)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"url": url})
}

func (h *WarrantyHandler) DeleteDocument(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	docID, err := uuid.Parse(c.Param("docId"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "docId inválido")
		return
	}
	if err := h.docSvc.Delete(c.Request.Context(), ws, docID); err != nil {
		errrespond.Write(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func parseOptionalUUID(s *string) (*uuid.UUID, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	id, err := uuid.Parse(*s)
	if err != nil {
		return nil, err
	}
	return &id, nil
}
