// Package pagination provides standard pagination utilities for list endpoints.
package pagination

import (
	"math"
	"strconv"

	"github.com/gin-gonic/gin"
)

const (
	DefaultPage    = 1
	DefaultPerPage = 20
	MaxPerPage     = 100
)

// Params holds pagination parameters parsed from query string.
type Params struct {
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
}

// Offset calculates the SQL offset for the current page.
func (p Params) Offset() int {
	return (p.Page - 1) * p.PerPage
}

// Meta holds pagination metadata for API responses.
type Meta struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// NewMeta creates pagination metadata from params and total count.
func NewMeta(params Params, total int64) Meta {
	totalPages := int(math.Ceil(float64(total) / float64(params.PerPage)))
	return Meta{
		Page:       params.Page,
		PerPage:    params.PerPage,
		Total:      int(total),
		TotalPages: totalPages,
	}
}

// FromContext extracts pagination parameters from a Gin request context.
// Defaults to page=1, per_page=20. Caps per_page at 100.
func FromContext(c *gin.Context) Params {
	page := queryInt(c, "page", DefaultPage)
	perPage := queryInt(c, "per_page", DefaultPerPage)

	if page < 1 {
		page = DefaultPage
	}
	if perPage < 1 {
		perPage = DefaultPerPage
	}
	if perPage > MaxPerPage {
		perPage = MaxPerPage
	}

	return Params{
		Page:    page,
		PerPage: perPage,
	}
}

func queryInt(c *gin.Context, key string, defaultVal int) int {
	if val := c.Query(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}
