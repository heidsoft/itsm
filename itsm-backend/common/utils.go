package common

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
)

// GenerateUUID 生成UUID
func GenerateUUID() string {
	return uuid.New().String()
}

// GenerateShortID 生成短ID（8位）
func GenerateShortID() string {
	return GenerateUUID()[:8]
}

// GenerateRandomString 生成指定长度的随机字符串
func GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	rand.Read(b)
	for i := range b {
		b[i] = charset[b[i]%byte(len(charset))]
	}
	return string(b)
}

// MD5Hash 计算MD5哈希值
func MD5Hash(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}

// StringToInt 字符串转整数，带默认值
func StringToInt(s string, defaultValue int) int {
	if i, err := strconv.Atoi(s); err == nil {
		return i
	}
	return defaultValue
}

// StringToInt64 字符串转int64，带默认值
func StringToInt64(s string, defaultValue int64) int64 {
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i
	}
	return defaultValue
}

// StringToBool 字符串转布尔值
func StringToBool(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "true" || s == "1" || s == "yes" || s == "on"
}

// InSlice 检查元素是否在切片中
func InSlice(item interface{}, slice interface{}) bool {
	s := reflect.ValueOf(slice)
	if s.Kind() != reflect.Slice {
		return false
	}

	for i := 0; i < s.Len(); i++ {
		if reflect.DeepEqual(item, s.Index(i).Interface()) {
			return true
		}
	}
	return false
}

// RemoveDuplicates 去除切片中的重复元素
func RemoveDuplicates(slice []string) []string {
	keys := make(map[string]bool)
	var result []string
	
	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}
	return result
}

// TruncateString 截断字符串
func TruncateString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	return s[:maxLength] + "..."
}

// CamelToSnake 驼峰转蛇形命名
func CamelToSnake(s string) string {
	var result []rune
	for i, r := range s {
		if i > 0 && unicode.IsUpper(r) {
			result = append(result, '_')
		}
		result = append(result, unicode.ToLower(r))
	}
	return string(result)
}

// SnakeToCamel 蛇形转驼峰命名
func SnakeToCamel(s string) string {
	parts := strings.Split(s, "_")
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}

// FormatTime 格式化时间
func FormatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

// FormatTimeISO 格式化时间为ISO格式
func FormatTimeISO(t time.Time) string {
	return t.Format(time.RFC3339)
}

// ParseTime 解析时间字符串
func ParseTime(timeStr string) (time.Time, error) {
	layouts := []string{
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05.000Z",
		time.RFC3339,
		"2006-01-02",
	}
	
	for _, layout := range layouts {
		if t, err := time.Parse(layout, timeStr); err == nil {
			return t, nil
		}
	}
	
	return time.Time{}, fmt.Errorf("无法解析时间格式: %s", timeStr)
}

// GetTimeRange 获取时间范围
func GetTimeRange(rangeType string) (time.Time, time.Time) {
	now := time.Now()
	var start, end time.Time
	
	switch rangeType {
	case "today":
		start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		end = start.Add(24 * time.Hour)
	case "yesterday":
		yesterday := now.AddDate(0, 0, -1)
		start = time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, yesterday.Location())
		end = start.Add(24 * time.Hour)
	case "this_week":
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		start = now.AddDate(0, 0, -weekday+1)
		start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
		end = start.AddDate(0, 0, 7)
	case "this_month":
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		end = start.AddDate(0, 1, 0)
	case "last_30_days":
		start = now.AddDate(0, 0, -30)
		start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
		end = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
	default:
		// 默认返回今天
		start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		end = start.Add(24 * time.Hour)
	}
	
	return start, end
}

// SafeString 安全获取字符串指针的值
func SafeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// SafeInt 安全获取整数指针的值
func SafeInt(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

// SafeInt64 安全获取int64指针的值
func SafeInt64(i *int64) int64 {
	if i == nil {
		return 0
	}
	return *i
}

// SafeBool 安全获取布尔指针的值
func SafeBool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

// StringPtr 获取字符串指针
func StringPtr(s string) *string {
	return &s
}

// IntPtr 获取整数指针
func IntPtr(i int) *int {
	return &i
}

// Int64Ptr 获取int64指针
func Int64Ptr(i int64) *int64 {
	return &i
}

// BoolPtr 获取布尔指针
func BoolPtr(b bool) *bool {
	return &b
}

// IsEmpty 检查值是否为空
func IsEmpty(value interface{}) bool {
	if value == nil {
		return true
	}
	
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.String:
		return v.Len() == 0
	case reflect.Slice, reflect.Map, reflect.Array:
		return v.Len() == 0
	case reflect.Ptr:
		return v.IsNil()
	default:
		return false
	}
}

// CoalesceString 返回第一个非空字符串
func CoalesceString(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// CoalesceInt 返回第一个非零整数
func CoalesceInt(values ...int) int {
	for _, v := range values {
		if v != 0 {
			return v
		}
	}
	return 0
}