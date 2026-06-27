package errrespond

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	domb "github.com/retechfin/retechfin-api/internal/domain/budget"
	doml "github.com/retechfin/retechfin-api/internal/domain/ledger"
)

type Body struct {
	Error Detail `json:"error"`
}

type Detail struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

const (
	CodeValidation          = "VALIDATION_ERROR"
	CodeNotFound            = "NOT_FOUND"
	CodeConflict            = "CONFLICT"
	CodeInternal            = "INTERNAL_ERROR"
	CodeBadRequest          = "BAD_REQUEST"
	CodeWorkspaceRequired   = "WORKSPACE_HEADER_REQUIRED"
	CodeUnauthorized        = "UNAUTHORIZED"
	CodeForbidden           = "FORBIDDEN"
)

func Write(c *gin.Context, err error) {
	rid, _ := c.Get("request_id")
	ridStr, _ := rid.(string)

	var lv *doml.ValidationError
	if errors.As(err, &lv) {
		c.JSON(http.StatusBadRequest, Body{Error: Detail{Code: CodeValidation, Message: lv.Msg, RequestID: ridStr}})
		return
	}
	var bv *domb.ValidationError
	if errors.As(err, &bv) {
		c.JSON(http.StatusBadRequest, Body{Error: Detail{Code: CodeValidation, Message: bv.Msg, RequestID: ridStr}})
		return
	}

	switch {
	case errors.Is(err, doml.ErrNotFound):
		c.JSON(http.StatusNotFound, Body{Error: Detail{Code: CodeNotFound, Message: err.Error(), RequestID: ridStr}})
	case errors.Is(err, domb.ErrNotFound):
		c.JSON(http.StatusNotFound, Body{Error: Detail{Code: CodeNotFound, Message: err.Error(), RequestID: ridStr}})
	case errors.Is(err, doml.ErrConflict):
		c.JSON(http.StatusConflict, Body{Error: Detail{Code: CodeConflict, Message: err.Error(), RequestID: ridStr}})
	case errors.Is(err, domb.ErrConflict):
		c.JSON(http.StatusConflict, Body{Error: Detail{Code: CodeConflict, Message: err.Error(), RequestID: ridStr}})
	case errors.Is(err, doml.ErrCategoryKindMismatch):
		c.JSON(http.StatusBadRequest, Body{Error: Detail{Code: CodeValidation, Message: err.Error(), RequestID: ridStr}})
	default:
		c.JSON(http.StatusInternalServerError, Body{Error: Detail{Code: CodeInternal, Message: "erro interno", RequestID: ridStr}})
	}
}

func Message(c *gin.Context, status int, code, message string) {
	rid, _ := c.Get("request_id")
	ridStr, _ := rid.(string)
	c.JSON(status, Body{Error: Detail{Code: code, Message: message, RequestID: ridStr}})
}
