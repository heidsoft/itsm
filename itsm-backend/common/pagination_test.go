package common

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ==================== PaginationRequest 测试 ====================

func TestPaginationRequest_Default(t *testing.T) {
	req := PaginationRequest{
		Page:     1,
		PageSize: 20,
	}

	assert.Equal(t, 1, req.Page)
	assert.Equal(t, 20, req.PageSize)
}

func TestPaginationRequest_GetOffset(t *testing.T) {
	tests := []struct {
		name     string
		page     int
		pageSize int
		expected int
	}{
		{"第1页", 1, 20, 0},
		{"第2页", 2, 20, 20},
		{"第3页", 3, 10, 20},
		{"第10页", 10, 50, 450},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := PaginationRequest{Page: tt.page, PageSize: tt.pageSize}
			assert.Equal(t, tt.expected, req.GetOffset())
		})
	}
}

func TestPaginationRequest_GetLimit(t *testing.T) {
	tests := []struct {
		name     string
		pageSize int
		expected int
	}{
		{"默认大小", 20, 20},
		{"大页面", 100, 100},
		{"小页面", 10, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := PaginationRequest{Page: 1, PageSize: tt.pageSize}
			assert.Equal(t, tt.expected, req.GetLimit())
		})
	}
}

// ==================== PaginationResponse 测试 ====================

func TestPaginationResponse_FirstPage(t *testing.T) {
	resp := PaginationResponse{
		Page:       1,
		PageSize:   20,
		Total:      100,
		TotalPages: 5,
		HasNext:    true,
		HasPrev:    false,
	}

	assert.Equal(t, 1, resp.Page)
	assert.True(t, resp.HasNext)
	assert.False(t, resp.HasPrev)
}

func TestPaginationResponse_LastPage(t *testing.T) {
	resp := PaginationResponse{
		Page:       5,
		PageSize:   20,
		Total:      100,
		TotalPages: 5,
		HasNext:    false,
		HasPrev:    true,
	}

	assert.Equal(t, 5, resp.Page)
	assert.False(t, resp.HasNext)
	assert.True(t, resp.HasPrev)
}

func TestPaginationResponse_MiddlePage(t *testing.T) {
	resp := PaginationResponse{
		Page:       3,
		PageSize:   20,
		Total:      100,
		TotalPages: 5,
		HasNext:    true,
		HasPrev:    true,
	}

	assert.Equal(t, 3, resp.Page)
	assert.True(t, resp.HasNext)
	assert.True(t, resp.HasPrev)
}

func TestPaginationResponse_Empty(t *testing.T) {
	resp := PaginationResponse{
		Page:       1,
		PageSize:   20,
		Total:      0,
		TotalPages: 0,
		HasNext:    false,
		HasPrev:    false,
	}

	assert.Equal(t, int64(0), resp.Total)
	assert.Equal(t, 0, resp.TotalPages)
	assert.False(t, resp.HasNext)
	assert.False(t, resp.HasPrev)
}

func TestPaginationResponse_JSONSerialization(t *testing.T) {
	resp := PaginationResponse{
		Page:       1,
		PageSize:   20,
		Total:      100,
		TotalPages: 5,
		HasNext:    true,
		HasPrev:    false,
	}

	jsonBytes, err := json.Marshal(resp)
	assert.NoError(t, err)

	var decoded PaginationResponse
	err = json.Unmarshal(jsonBytes, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, resp.Page, decoded.Page)
	assert.Equal(t, resp.Total, decoded.Total)
}

// ==================== ListResponse 测试 ====================

func TestListResponse_WithData(t *testing.T) {
	data := []int{1, 2, 3, 4, 5}
	pagination := &PaginationResponse{
		Page:       1,
		PageSize:   20,
		Total:      5,
		TotalPages: 1,
		HasNext:    false,
		HasPrev:    false,
	}

	resp := ListResponse{
		Data:       data,
		Pagination: pagination,
	}

	assert.Len(t, resp.Data.([]int), 5)
	assert.NotNil(t, resp.Pagination)
}

func TestListResponse_WithNilPagination(t *testing.T) {
	resp := ListResponse{
		Data:       []string{"a", "b", "c"},
		Pagination: nil,
	}

	assert.Len(t, resp.Data.([]string), 3)
	assert.Nil(t, resp.Pagination)
}

// ==================== GetPaginationFromQuery 测试 ====================

func TestGetPaginationFromQuery_Defaults(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/api/test", nil)

	pagination := GetPaginationFromQuery(c)

	assert.Equal(t, 1, pagination.Page)
	assert.Equal(t, 20, pagination.PageSize)
}

func TestGetPaginationFromQuery_WithPage(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/api/test?page=5", nil)

	pagination := GetPaginationFromQuery(c)

	assert.Equal(t, 5, pagination.Page)
	assert.Equal(t, 20, pagination.PageSize)
}

func TestGetPaginationFromQuery_WithPageSize(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/api/test?page_size=50", nil)

	pagination := GetPaginationFromQuery(c)

	assert.Equal(t, 1, pagination.Page)
	assert.Equal(t, 50, pagination.PageSize)
}

func TestGetPaginationFromQuery_WithBoth(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/api/test?page=3&page_size=30", nil)

	pagination := GetPaginationFromQuery(c)

	assert.Equal(t, 3, pagination.Page)
	assert.Equal(t, 30, pagination.PageSize)
}

func TestGetPaginationFromQuery_InvalidPage(t *testing.T) {
	tests := []struct {
		name  string
		query string
		page  int
	}{
		{"负数页码", "page=-1", 1},
		{"零页码", "page=0", 1},
		{"非数字页码", "page=abc", 1},
		{"超大页码", "page=1000", 1000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("GET", "/api/test?"+tt.query, nil)

			pagination := GetPaginationFromQuery(c)
			assert.Equal(t, tt.page, pagination.Page)
		})
	}
}

func TestGetPaginationFromQuery_InvalidPageSize(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		pageSize int
	}{
		{"负数大小", "page_size=-1", 20},
		{"零大小", "page_size=0", 20},
		{"非数字大小", "page_size=abc", 20},
		{"超大限制", "page_size=200", 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("GET", "/api/test?"+tt.query, nil)

			pagination := GetPaginationFromQuery(c)
			assert.Equal(t, tt.pageSize, pagination.PageSize)
		})
	}
}

func TestGetPaginationFromQuery_ValidPageSize(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		pageSize int
	}{
		{"边界值1", "page_size=1", 1},
		{"边界值100", "page_size=100", 100},
		{"正常值50", "page_size=50", 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("GET", "/api/test?"+tt.query, nil)

			pagination := GetPaginationFromQuery(c)
			assert.Equal(t, tt.pageSize, pagination.PageSize)
		})
	}
}

// ==================== SuccessWithPagination 测试 ====================

func TestSuccessWithPagination(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	data := []string{"a", "b", "c"}
	SuccessWithPagination(c, data, 1, 10, 3)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, SuccessCode, resp.Code)

	// 验证内部结构
	listResp, ok := resp.Data.(map[string]interface{})
	assert.True(t, ok)
	assert.NotNil(t, listResp["data"])
	assert.NotNil(t, listResp["pagination"])
}

// ==================== 边界情况测试 ====================

func TestPagination_ZeroTotal(t *testing.T) {
	resp := NewPaginationResponse(1, 20, 0)

	assert.Equal(t, int64(0), resp.Total)
	assert.Equal(t, 0, resp.TotalPages)
	assert.False(t, resp.HasNext)
	assert.False(t, resp.HasPrev)
}

func TestPagination_SinglePage(t *testing.T) {
	resp := NewPaginationResponse(1, 20, 10)

	assert.Equal(t, 1, resp.TotalPages)
	assert.False(t, resp.HasNext)
	assert.False(t, resp.HasPrev)
}

func TestPagination_LargeTotal(t *testing.T) {
	resp := NewPaginationResponse(1, 10, 10000)

	assert.Equal(t, 1000, resp.TotalPages)
	assert.True(t, resp.HasNext)
	assert.False(t, resp.HasPrev)
}

func TestPagination_LastPageOfMany(t *testing.T) {
	resp := NewPaginationResponse(1000, 10, 10000)

	assert.Equal(t, 1000, resp.TotalPages)
	assert.False(t, resp.HasNext)
	assert.True(t, resp.HasPrev)
}
