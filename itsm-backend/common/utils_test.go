package common

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// ============== utils.go 测试 ==============

func TestGenerateUUID(t *testing.T) {
	id := GenerateUUID()
	if len(id) != 36 {
		t.Errorf("expected UUID length 36, got %d (id=%s)", len(id), id)
	}
	// UUID v4 格式：xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx（y ∈ [8,9,a,b]）
	if id[14] != '4' {
		t.Errorf("expected UUID v4 marker at position 14, got %c (id=%s)", id[14], id)
	}
}

func TestGenerateRequestID(t *testing.T) {
	id := GenerateRequestID()
	if id == "" {
		t.Error("GenerateRequestID returned empty string")
	}
}

func TestGenerateShortID(t *testing.T) {
	id := GenerateShortID()
	if len(id) != 8 {
		t.Errorf("expected short ID length 8, got %d (id=%s)", len(id), id)
	}
}

func TestGenerateRandomString(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"zero-length", 0},
		{"small", 8},
		{"medium", 32},
		{"large", 256},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := GenerateRandomString(tt.length)
			if len(s) != tt.length {
				t.Errorf("expected length %d, got %d", tt.length, len(s))
			}
		})
	}

	// 不同调用应返回不同字符串
	a := GenerateRandomString(16)
	b := GenerateRandomString(16)
	if a == b {
		t.Error("two random strings should differ")
	}
}

func TestMD5Hash(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "d41d8cd98f00b204e9800998ecf8427e"},
		{"hello", "5d41402abc4b2a76b9719d911017c592"},
		{"abc", "900150983cd24fb0d6963f7d28e17f72"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := MD5Hash(tt.input)
			if got != tt.expected {
				t.Errorf("MD5Hash(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}

	// 验证幂等性
	if MD5Hash("test") != MD5Hash("test") {
		t.Error("MD5Hash should be deterministic")
	}
}

func TestStringToInt(t *testing.T) {
	tests := []struct {
		input        string
		defaultValue int
		expected     int
	}{
		{"123", 0, 123},
		{"-456", 0, -456},
		{"abc", 99, 99},
		{"", 50, 50},
		{"0", 10, 0}, // 0 是有效整数（虽然语义上等价于默认）
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := StringToInt(tt.input, tt.defaultValue)
			if got != tt.expected {
				t.Errorf("StringToInt(%q, %d) = %d, want %d", tt.input, tt.defaultValue, got, tt.expected)
			}
		})
	}
}

func TestStringToInt64(t *testing.T) {
	if StringToInt64("123", 0) != 123 {
		t.Error("StringToInt64 should parse valid int64")
	}
	if StringToInt64("abc", 99) != 99 {
		t.Error("StringToInt64 should return default for invalid input")
	}
}

func TestStringToBool(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"true", true},
		{"TRUE", true},
		{"True", true},
		{"1", true},
		{"yes", true},
		{"YES", true},
		{"on", true},
		{"false", false},
		{"0", false},
		{"no", false},
		{"off", false},
		{"", false},
		{"random", false},
		{" true ", true}, // 验证 TrimSpace
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := StringToBool(tt.input)
			if got != tt.expected {
				t.Errorf("StringToBool(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestInSlice(t *testing.T) {
	t.Run("int-slice", func(t *testing.T) {
		if !InSlice(2, []int{1, 2, 3}) {
			t.Error("InSlice should find existing element")
		}
		if InSlice(4, []int{1, 2, 3}) {
			t.Error("InSlice should not find missing element")
		}
	})

	t.Run("string-slice", func(t *testing.T) {
		if !InSlice("b", []string{"a", "b", "c"}) {
			t.Error("InSlice should find string")
		}
	})

	t.Run("empty-slice", func(t *testing.T) {
		if InSlice(1, []int{}) {
			t.Error("InSlice on empty slice should return false")
		}
	})

	t.Run("not-slice", func(t *testing.T) {
		if InSlice(1, 1) {
			t.Error("InSlice on non-slice should return false")
		}
	})

	t.Run("nil-slice", func(t *testing.T) {
		if InSlice(1, []int(nil)) {
			t.Error("InSlice on nil slice should return false")
		}
	})
}

func TestRemoveDuplicates(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{"empty", []string{}, []string{}},
		{"no-duplicates", []string{"a", "b", "c"}, []string{"a", "b", "c"}},
		{"all-duplicates", []string{"a", "a", "a"}, []string{"a"}},
		{"mixed", []string{"a", "b", "a", "c", "b"}, []string{"a", "b", "c"}},
		{"nil", nil, []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RemoveDuplicates(tt.input)
			if len(got) != len(tt.expected) {
				t.Errorf("expected length %d, got %d (%v)", len(tt.expected), len(got), got)
				return
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("at index %d: expected %q, got %q", i, tt.expected[i], got[i])
				}
			}
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"hello", 10, "hello"},
		{"hello world", 5, "hello..."},
		{"abc", 3, "abc"},
		{"", 5, ""},
		{"test", 0, "..."},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := TruncateString(tt.input, tt.maxLen)
			if got != tt.expected {
				t.Errorf("TruncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.expected)
			}
		})
	}
}

func TestCamelToSnake(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"UserId", "user_id"},
		{"userId", "user_id"},
		// 实际实现：每个大写字母前都插入下划线（不识别连续大写缩写）
		{"HTTPServer", "h_t_t_p_server"},
		{"simple", "simple"},
		{"", ""},
		{"A", "a"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := CamelToSnake(tt.input)
			if got != tt.expected {
				t.Errorf("CamelToSnake(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestSnakeToCamel(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"user_id", "userId"},
		{"first_name", "firstName"},
		{"simple", "simple"},
		{"a_b_c", "aBC"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := SnakeToCamel(tt.input)
			if got != tt.expected {
				t.Errorf("SnakeToCamel(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestParseTime(t *testing.T) {
	tests := []struct {
		input    string
		success  bool
	}{
		{"2024-01-15 10:30:00", true},
		{"2024-01-15T10:30:00Z", true},
		{"2024-01-15T10:30:00.000Z", true},
		{"2024-01-15", true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			_, err := ParseTime(tt.input)
			if (err == nil) != tt.success {
				t.Errorf("ParseTime(%q) success=%v, want %v (err=%v)", tt.input, err == nil, tt.success, err)
			}
		})
	}
}

func TestGetTimeRange(t *testing.T) {
	tests := []string{"today", "yesterday", "this_week", "this_month", "last_30_days", "unknown"}

	for _, rangeType := range tests {
		t.Run(rangeType, func(t *testing.T) {
			start, end := GetTimeRange(rangeType)
			if start.IsZero() {
				t.Errorf("GetTimeRange(%q) returned zero start time", rangeType)
			}
			if end.IsZero() {
				t.Errorf("GetTimeRange(%q) returned zero end time", rangeType)
			}
			if !end.After(start) && !end.Equal(start) {
				t.Errorf("GetTimeRange(%q): end (%v) should be >= start (%v)", rangeType, end, start)
			}
		})
	}
}

func TestSafePointers(t *testing.T) {
	t.Run("SafeString", func(t *testing.T) {
		if SafeString(nil) != "" {
			t.Error("SafeString(nil) should return empty string")
		}
		s := "hello"
		if SafeString(&s) != "hello" {
			t.Error("SafeString should dereference pointer")
		}
	})

	t.Run("SafeInt", func(t *testing.T) {
		if SafeInt(nil) != 0 {
			t.Error("SafeInt(nil) should return 0")
		}
		i := 42
		if SafeInt(&i) != 42 {
			t.Error("SafeInt should dereference pointer")
		}
	})

	t.Run("SafeInt64", func(t *testing.T) {
		if SafeInt64(nil) != 0 {
			t.Error("SafeInt64(nil) should return 0")
		}
		i := int64(123)
		if SafeInt64(&i) != 123 {
			t.Error("SafeInt64 should dereference pointer")
		}
	})

	t.Run("SafeBool", func(t *testing.T) {
		if SafeBool(nil) != false {
			t.Error("SafeBool(nil) should return false")
		}
		b := true
		if !SafeBool(&b) {
			t.Error("SafeBool should dereference pointer")
		}
	})

	t.Run("PointerConstructors", func(t *testing.T) {
		if *StringPtr("x") != "x" {
			t.Error("StringPtr should return *string")
		}
		if *IntPtr(5) != 5 {
			t.Error("IntPtr should return *int")
		}
		if *Int64Ptr(10) != 10 {
			t.Error("Int64Ptr should return *int64")
		}
		if !*BoolPtr(true) {
			t.Error("BoolPtr should return *bool")
		}
	})
}

func TestIsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected bool
	}{
		{"nil", nil, true},
		{"empty-string", "", true},
		{"non-empty-string", "x", false},
		{"empty-slice", []int{}, true},
		{"non-empty-slice", []int{1}, false},
		{"empty-map", map[string]int{}, true},
		{"non-empty-map", map[string]int{"a": 1}, false},
		{"nil-pointer", (*string)(nil), true},
		{"non-nil-pointer", StringPtr("x"), false},
		{"int-zero", 0, false}, // 0 是 int 默认值，但非空
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsEmpty(tt.value)
			if got != tt.expected {
				t.Errorf("IsEmpty(%v) = %v, want %v", tt.value, got, tt.expected)
			}
		})
	}
}

func TestCoalesceString(t *testing.T) {
	tests := []struct {
		input    []string
		expected string
	}{
		{[]string{}, ""},
		{[]string{"a"}, "a"},
		{[]string{"", "b"}, "b"},
		{[]string{"", "", "c"}, "c"},
		{[]string{"", "", ""}, ""},
	}

	for _, tt := range tests {
		got := CoalesceString(tt.input...)
		if got != tt.expected {
			t.Errorf("CoalesceString(%v) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestCoalesceInt(t *testing.T) {
	tests := []struct {
		input    []int
		expected int
	}{
		{[]int{}, 0},
		{[]int{0}, 0},
		{[]int{5}, 5},
		{[]int{0, 0, 10}, 10},
		{[]int{0, 7, 10}, 7},
	}

	for _, tt := range tests {
		got := CoalesceInt(tt.input...)
		if got != tt.expected {
			t.Errorf("CoalesceInt(%v) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}

// ============== pagination.go 测试 ==============

func setupGinTest() *gin.Context {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	return c
}

func TestPaginationRequestOffset(t *testing.T) {
	tests := []struct {
		page     int
		pageSize int
		offset   int
		limit    int
	}{
		{1, 20, 0, 20},
		{2, 20, 20, 20},
		{3, 10, 20, 10},
		{1, 1, 0, 1},
	}

	for _, tt := range tests {
		p := &PaginationRequest{Page: tt.page, PageSize: tt.pageSize}
		if got := p.GetOffset(); got != tt.offset {
			t.Errorf("Page=%d, Size=%d: offset=%d, want %d", tt.page, tt.pageSize, got, tt.offset)
		}
		if got := p.GetLimit(); got != tt.limit {
			t.Errorf("Page=%d, Size=%d: limit=%d, want %d", tt.page, tt.pageSize, got, tt.limit)
		}
	}
}

func TestNewPaginationResponse(t *testing.T) {
	tests := []struct {
		name        string
		page        int
		pageSize    int
		total       int64
		wantPages   int
		wantHasNext bool
		wantHasPrev bool
	}{
		{"empty", 1, 20, 0, 0, false, false},
		{"single-page", 1, 20, 5, 1, false, false},
		{"exact-page", 1, 20, 20, 1, false, false},
		{"multi-page-first", 1, 10, 25, 3, true, false},
		{"multi-page-mid", 2, 10, 25, 3, true, true},
		{"multi-page-last", 3, 10, 25, 3, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewPaginationResponse(tt.page, tt.pageSize, tt.total)
			if got.TotalPages != tt.wantPages {
				t.Errorf("TotalPages=%d, want %d", got.TotalPages, tt.wantPages)
			}
			if got.HasNext != tt.wantHasNext {
				t.Errorf("HasNext=%v, want %v", got.HasNext, tt.wantHasNext)
			}
			if got.HasPrev != tt.wantHasPrev {
				t.Errorf("HasPrev=%v, want %v", got.HasPrev, tt.wantHasPrev)
			}
			if got.Page != tt.page {
				t.Errorf("Page=%d, want %d", got.Page, tt.page)
			}
			if got.PageSize != tt.pageSize {
				t.Errorf("PageSize=%d, want %d", got.PageSize, tt.pageSize)
			}
		})
	}
}

func TestValidatePagination(t *testing.T) {
	tests := []struct {
		page         int
		pageSize     int
		wantPage     int
		wantPageSize int
	}{
		{-1, 20, DefaultPage, 20},
		{0, 20, DefaultPage, 20},
		{5, 20, 5, 20},
		{5, -1, 5, DefaultPageSize},
		{5, 0, 5, DefaultPageSize},
		{5, 9999, 5, MaxPageSize},
	}

	for _, tt := range tests {
		gotPage, gotSize := ValidatePagination(tt.page, tt.pageSize)
		if gotPage != tt.wantPage {
			t.Errorf("ValidatePagination(%d,%d) page=%d, want %d", tt.page, tt.pageSize, gotPage, tt.wantPage)
		}
		if gotSize != tt.wantPageSize {
			t.Errorf("ValidatePagination(%d,%d) size=%d, want %d", tt.page, tt.pageSize, gotSize, tt.wantPageSize)
		}
	}
}

func TestGetPaginationFromQuery(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		c := setupGinTest()
		p := GetPaginationFromQuery(c)
		if p.Page != 1 {
			t.Errorf("default page=%d, want 1", p.Page)
		}
		if p.PageSize != 20 {
			t.Errorf("default pageSize=%d, want 20", p.PageSize)
		}
	})

	t.Run("valid-params", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/v1/tickets?page=3&page_size=50", nil)
		p := GetPaginationFromQuery(c)
		if p.Page != 3 {
			t.Errorf("page=%d, want 3", p.Page)
		}
		if p.PageSize != 50 {
			t.Errorf("pageSize=%d, want 50", p.PageSize)
		}
	})

	t.Run("invalid-params-fallback", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/v1/tickets?page=abc&page_size=xyz", nil)
		p := GetPaginationFromQuery(c)
		if p.Page != 1 {
			t.Errorf("invalid page should fallback to 1, got %d", p.Page)
		}
		if p.PageSize != 20 {
			t.Errorf("invalid pageSize should fallback to 20, got %d", p.PageSize)
		}
	})

	t.Run("oversize-falls-back-to-default", func(t *testing.T) {
		// 实际行为：超过 100 时整个 if 跳过，pageSize 保留默认 20
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/v1/tickets?page_size=9999", nil)
		p := GetPaginationFromQuery(c)
		if p.PageSize != 20 {
			t.Errorf("oversize pageSize should fallback to default 20, got %d", p.PageSize)
		}
	})
}

func TestNewPaginationMeta(t *testing.T) {
	m := NewPaginationMeta(3, 25)
	if m.Page != 3 || m.Size != 25 || m.Offset != 50 || m.Limit != 25 {
		t.Errorf("unexpected meta: %+v", m)
	}

	// 验证 ValidatePagination 被调用
	mInvalid := NewPaginationMeta(-1, 9999)
	if mInvalid.Page != DefaultPage {
		t.Errorf("invalid page should be sanitized, got %d", mInvalid.Page)
	}
	if mInvalid.Size != MaxPageSize {
		t.Errorf("oversize size should clamp, got %d", mInvalid.Size)
	}
}

// ============== response.go 测试 ==============

func TestSuccessResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Success(c, map[string]string{"key": "value"})

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, `"code":0`) {
		t.Errorf("expected code=0, body=%s", body)
	}
	if !strings.Contains(body, `"message":"success"`) {
		t.Errorf("expected message=success, body=%s", body)
	}
	if !strings.Contains(body, `"key":"value"`) {
		t.Errorf("expected data, body=%s", body)
	}
}

func TestFailResponse(t *testing.T) {
	tests := []struct {
		code         int
		wantHTTPCode int
	}{
		{ParamErrorCode, http.StatusBadRequest},
		{ValidationError, http.StatusBadRequest},
		{BadRequestCode, http.StatusBadRequest},
		{AuthFailedCode, http.StatusUnauthorized},
		{ForbiddenCode, http.StatusForbidden},
		{NotFoundCode, http.StatusNotFound},
		{ConflictCode, http.StatusConflict},
		{InternalErrorCode, http.StatusInternalServerError},
		{9999, http.StatusOK}, // 未知 code 走 200
	}

	for _, tt := range tests {
		t.Run("code-"+string(rune(tt.code)), func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			Fail(c, tt.code, "test message")

			if w.Code != tt.wantHTTPCode {
				t.Errorf("code=%d: expected HTTP %d, got %d", tt.code, tt.wantHTTPCode, w.Code)
			}
		})
	}
}

func TestFailWithDataResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	FailWithData(c, AuthFailedCode, "auth failed", gin.H{"hint": "login required"})

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"hint":"login required"`) {
		t.Errorf("expected data field, body=%s", w.Body.String())
	}
}

func TestSuccessWithMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	SuccessWithMessage(c, "custom message", "data")

	body := w.Body.String()
	if !strings.Contains(body, `"message":"custom message"`) {
		t.Errorf("expected custom message, body=%s", body)
	}
}

func TestConflictResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Conflict(c, "version conflict", gin.H{"version": 5})

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"code":4090`) {
		t.Errorf("expected code=4090, body=%s", w.Body.String())
	}
}

func TestSuccessWithList(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	SuccessWithList(c, []string{"a", "b", "c"}, 3, 1, 20)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, `"data"`) {
		t.Errorf("expected data field, body=%s", body)
	}
	if !strings.Contains(body, `"pagination"`) {
		t.Errorf("expected pagination field, body=%s", body)
	}
}

func TestConvenienceFailHelpers(t *testing.T) {
	tests := []struct {
		name        string
		fn          func(c *gin.Context, msg string)
		wantCode    int
		wantHTTPCode int
	}{
		{"ParamError", ParamError, ParamErrorCode, http.StatusBadRequest},
		{"ValidationErrorResponse", ValidationErrorResponse, ValidationError, http.StatusBadRequest},
		{"AuthFailed", AuthFailed, AuthFailedCode, http.StatusUnauthorized},
		{"Forbidden", Forbidden, ForbiddenCode, http.StatusForbidden},
		{"NotFound", NotFound, NotFoundCode, http.StatusNotFound},
		{"InternalError", InternalError, InternalErrorCode, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			tt.fn(c, "test msg")

			if w.Code != tt.wantHTTPCode {
				t.Errorf("expected HTTP %d, got %d", tt.wantHTTPCode, w.Code)
			}
			if !strings.Contains(w.Body.String(), `"code":`+itoa(tt.wantCode)) {
				t.Errorf("expected code=%d, body=%s", tt.wantCode, w.Body.String())
			}
		})
	}
}

// ============== errors.go 测试 ==============

func TestBusinessError(t *testing.T) {
	err := NewBusinessError(4001, "ticket not found", "id=123")
	if err.Code != 4001 {
		t.Errorf("expected code=4001, got %d", err.Code)
	}
	if err.Message != "ticket not found" {
		t.Errorf("unexpected message: %s", err.Message)
	}
	if err.Detail != "id=123" {
		t.Errorf("unexpected detail: %s", err.Detail)
	}

	msg := err.Error()
	if !strings.Contains(msg, "4001") || !strings.Contains(msg, "ticket not found") {
		t.Errorf("Error() should contain code and message, got: %s", msg)
	}
}

func TestVersionConflictError(t *testing.T) {
	err := NewVersionConflictError("ticket", 100, 3, 5)
	if err.ResourceName != "ticket" {
		t.Errorf("unexpected ResourceName: %s", err.ResourceName)
	}
	if err.ResourceID != 100 {
		t.Errorf("unexpected ResourceID: %d", err.ResourceID)
	}
	if err.CurrentVersion != 3 {
		t.Errorf("unexpected CurrentVersion: %d", err.CurrentVersion)
	}
	if err.ServerVersion != 5 {
		t.Errorf("unexpected ServerVersion: %d", err.ServerVersion)
	}

	msg := err.Error()
	if !strings.Contains(msg, "ticket") || !strings.Contains(msg, "100") || !strings.Contains(msg, "3") || !strings.Contains(msg, "5") {
		t.Errorf("Error() should contain all fields, got: %s", msg)
	}

	if !IsVersionConflictError(err) {
		t.Error("IsVersionConflictError should return true for VersionConflictError")
	}
	if IsVersionConflictError(nil) {
		t.Error("IsVersionConflictError(nil) should return false")
	}
	if IsVersionConflictError(NewBusinessError(4001, "x", "y")) {
		t.Error("IsVersionConflictError should return false for other errors")
	}
}

// 辅助函数
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	negative := false
	if i < 0 {
		negative = true
		i = -i
	}
	digits := []byte{}
	for i > 0 {
		digits = append([]byte{byte('0' + i%10)}, digits...)
		i /= 10
	}
	if negative {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}
