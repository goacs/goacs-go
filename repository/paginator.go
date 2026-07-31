package repository

import (
	"github.com/gin-gonic/gin"
	"math"
	"strconv"
	"strings"
)

type PaginatorRequest struct {
	Page    int               `json:"page"`
	PerPage int               `json:"per_page"`
	Filter  map[string]string `json:"filter"`
	Sort    map[string]string `json:"sort"`
}

type PaginatorResponse struct {
	PaginatorRequest
	NextPage int         `json:"next_page"`
	PrevPage int         `json:"prev_page"`
	Total    int         `json:"total"`
	Data     interface{} `json:"data"`
}

func DefaultPaginatorRequest(page int) PaginatorRequest {
	return PaginatorRequest{
		Page:    page,
		PerPage: 50,
	}
}

func PaginatorRequestFromContext(ctx *gin.Context) PaginatorRequest {
	qPage := ctx.Query("page")
	if qPage == "" {
		qPage = "1"
	}
	page, _ := strconv.Atoi(qPage)

	qPerPage := ctx.Query("per_page")
	if qPerPage == "" {
		qPerPage = "25"
	}
	perPage, _ := strconv.Atoi(qPerPage)

	filter := ctx.QueryMap("filter")
	sort := ctx.QueryMap("sort")

	return PaginatorRequest{
		Page:    page,
		PerPage: perPage,
		Filter:  filter,
		Sort:    sort,
	}
}

// SortOrExpressions returns the requested sort as goqu OrderedExpressions,
// e.g. sort[serial_number]=asc -> [goqu.C("serial_number").Asc()].
// Unrecognised directions default to ascending. Callers should validate
// column names against an allow-list before use to avoid sorting by
// arbitrary/unindexed columns.
func (p *PaginatorRequest) SortColumns() []SortColumn {
	columns := make([]SortColumn, 0, len(p.Sort))
	for column, direction := range p.Sort {
		columns = append(columns, SortColumn{Column: column, Descending: strings.EqualFold(direction, "desc")})
	}

	return columns
}

type SortColumn struct {
	Column     string
	Descending bool
}

func NewPaginatorResponse(request PaginatorRequest, total int, data interface{}) PaginatorResponse {
	pages := int(math.Ceil(float64(total) / float64(request.PerPage)))

	var prev int
	if request.Page > 1 {
		prev = request.Page - 1
	} else {
		prev = 1
	}

	var next int
	if request.Page < pages {
		next = request.Page + 1
	} else {
		next = pages
	}

	return PaginatorResponse{
		PaginatorRequest: request,
		PrevPage:         prev,
		NextPage:         next,
		Total:            total,
		Data:             data,
	}
}

func (p *PaginatorRequest) CalcOffset() int {
	return (p.Page - 1) * p.PerPage
}
