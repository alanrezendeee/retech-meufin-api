package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

const defaultLimit = 20

// maxLimit acompanha a maior opção de "itens por página" do front (500).
const maxLimit = 500

func pagination(c *gin.Context) (limit, offset int) {
	limit = defaultLimit
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
			if limit > maxLimit {
				limit = maxLimit
			}
		}
	}
	offset = 0
	if v := c.Query("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	return limit, offset
}

// boolQuery lê um query param booleano ("true"/"false"); nil quando ausente
// ou inválido — filtro tri-state (todos / só true / só false).
func boolQuery(c *gin.Context, name string) *bool {
	switch c.Query(name) {
	case "true":
		v := true
		return &v
	case "false":
		v := false
		return &v
	}
	return nil
}
