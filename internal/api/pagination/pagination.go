package pagination

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

const (
	defaultPage = 1
	defaultSize = 20
	maxSize     = 100
)

// Params represents normalized pagination query parameters.
type Params struct {
	Page int
	Size int
}

// FromQuery parses page and page_size from the request.
func FromQuery(c *gin.Context) Params {
	page, _ := strconv.Atoi(c.DefaultQuery("page", strconv.Itoa(defaultPage)))
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", strconv.Itoa(defaultSize)))
	if page < 1 {
		page = defaultPage
	}
	if size < 1 {
		size = defaultSize
	}
	if size > maxSize {
		size = maxSize
	}
	return Params{Page: page, Size: size}
}

// Offset returns the SQL offset for LIMIT/OFFSET style queries.
func (p Params) Offset() int {
	return (p.Page - 1) * p.Size
}
