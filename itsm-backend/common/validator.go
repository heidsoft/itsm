package common

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	validate = validator.New()
	
	// 注册自定义验证规则
	validate.RegisterValidation("priority", validatePriority)
	validate.RegisterValidation("status", validateStatus)
	validate.RegisterValidation("phone", validatePhone)
	validate.RegisterValidation("chinese_name", validateChineseName)
}

// ValidateStruct 验证结构体
func ValidateStruct(s interface{}) error {
	return validate.Struct(s)
}

// ValidateAndBindJSON 验证并绑定JSON请求
func ValidateAndBindJSON(c *gin.Context, obj interface{}) error {
	if err := c.ShouldBindJSON(obj); err != nil {
		return NewBusinessError(ParamErrorCode, "请求参数格式错误", err.Error())
	}
	
	if err := ValidateStruct(obj); err != nil {
		return NewBusinessError(ValidationError, "参数验证失败", formatValidationError(err))
	}
	
	return nil
}

// ValidateAndBindQuery 验证并绑定查询参数
func ValidateAndBindQuery(c *gin.Context, obj interface{}) error {
	if err := c.ShouldBindQuery(obj); err != nil {
		return NewBusinessError(ParamErrorCode, "查询参数格式错误", err.Error())
	}
	
	if err := ValidateStruct(obj); err != nil {
		return NewBusinessError(ValidationError, "参数验证失败", formatValidationError(err))
	}
	
	return nil
}

// 格式化验证错误信息
func formatValidationError(err error) string {
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		var messages []string
		for _, e := range validationErrors {
			message := getValidationMessage(e)
			messages = append(messages, message)
		}
		return strings.Join(messages, "; ")
	}
	return err.Error()
}

// 获取验证错误的友好消息
func getValidationMessage(e validator.FieldError) string {
	field := e.Field()
	tag := e.Tag()
	param := e.Param()
	
	switch tag {
	case "required":
		return fmt.Sprintf("%s 是必填字段", field)
	case "min":
		return fmt.Sprintf("%s 最小长度为 %s", field, param)
	case "max":
		return fmt.Sprintf("%s 最大长度为 %s", field, param)
	case "email":
		return fmt.Sprintf("%s 必须是有效的邮箱地址", field)
	case "oneof":
		return fmt.Sprintf("%s 必须是以下值之一: %s", field, param)
	case "priority":
		return fmt.Sprintf("%s 必须是有效的优先级 (low, medium, high, critical)", field)
	case "status":
		return fmt.Sprintf("%s 必须是有效的状态", field)
	case "phone":
		return fmt.Sprintf("%s 必须是有效的手机号码", field)
	case "chinese_name":
		return fmt.Sprintf("%s 必须是有效的中文姓名", field)
	default:
		return fmt.Sprintf("%s 验证失败", field)
	}
}

// 自定义验证规则：优先级
func validatePriority(fl validator.FieldLevel) bool {
	priority := fl.Field().String()
	validPriorities := []string{"low", "medium", "high", "critical"}
	
	for _, valid := range validPriorities {
		if priority == valid {
			return true
		}
	}
	return false
}

// 自定义验证规则：状态
func validateStatus(fl validator.FieldLevel) bool {
	status := fl.Field().String()
	validStatuses := []string{"open", "in_progress", "resolved", "closed", "pending", "cancelled"}
	
	for _, valid := range validStatuses {
		if status == valid {
			return true
		}
	}
	return false
}

// 自定义验证规则：手机号码
func validatePhone(fl validator.FieldLevel) bool {
	phone := fl.Field().String()
	// 中国手机号码正则表达式
	phoneRegex := regexp.MustCompile(`^1[3-9]\d{9}$`)
	return phoneRegex.MatchString(phone)
}

// 自定义验证规则：中文姓名
func validateChineseName(fl validator.FieldLevel) bool {
	name := fl.Field().String()
	if name == "" {
		return false
	}
	
	// 检查长度（2-10个字符）
	if utf8.RuneCountInString(name) < 2 || utf8.RuneCountInString(name) > 10 {
		return false
	}
	
	// 检查是否包含中文字符
	chineseRegex := regexp.MustCompile(`[\p{Han}]+`)
	return chineseRegex.MatchString(name)
}

// 常用验证辅助函数
func IsValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

func IsValidURL(url string) bool {
	urlRegex := regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
	return urlRegex.MatchString(url)
}

func IsValidIPAddress(ip string) bool {
	ipRegex := regexp.MustCompile(`^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$`)
	return ipRegex.MatchString(ip)
}