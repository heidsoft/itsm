package service

import (
	"encoding/json"
	"fmt"

	"go.uber.org/zap"
)

// SafeMarshal JSON序列化，错误时返回空对象
func SafeMarshal(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		return []byte("{}")
	}
	return data
}

// SafeMarshalIndent 带缩进的JSON序列化
func SafeMarshalIndent(v interface{}) string {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(data)
}

// MustMarshal JSON序列化，错误时panic
func MustMarshal(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal: %v", err))
	}
	return data
}

// SafeUnmarshal JSON反序列化
func SafeUnmarshal(data []byte, v interface{}) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, v)
}

// SafeConvert 安全的类型转换
func SafeConvert[T any](value interface{}, defaultValue T) T {
	if value == nil {
		return defaultValue
	}
	if v, ok := value.(T); ok {
		return v
	}
	return defaultValue
}

// LogError 记录错误日志
func LogError(logger *zap.SugaredLogger, message string, err error) {
	if err != nil {
		logger.Errorw(message, "error", err.Error())
	}
}

// LogErrorWithCtx 带上下文的错误日志
func LogErrorWithCtx(logger *zap.SugaredLogger, message string, err error, context map[string]interface{}) {
	if err != nil {
		context["error"] = err.Error()
		logger.Errorw(message, zap.Any("context", context))
	}
}

// RecoverError panic恢复
func RecoverError(logger *zap.SugaredLogger, message string) {
	if r := recover(); r != nil {
		logger.Errorw(message, "recovered", fmt.Sprintf("%v", r))
	}
}

// IgnoreError 静默忽略错误（仅用于安全的操作）
// 使用此函数前请确认错误可以安全忽略
func IgnoreError(_ ...interface{}) {
	// Intentionally empty to ignore errors
}

// CheckError 检查错误并记录
func CheckError(logger *zap.SugaredLogger, err error, message string) bool {
	if err != nil {
		LogError(logger, message, err)
		return true
	}
	return false
}

// ErrorHandler 统一的错误处理模式
type ErrorHandler struct {
	logger    *zap.SugaredLogger
	operation string
}

func NewErrorHandler(logger *zap.SugaredLogger, operation string) *ErrorHandler {
	return &ErrorHandler{
		logger:    logger,
		operation: operation,
	}
}

// Handle 处理错误，返回是否发生错误
func (h *ErrorHandler) Handle(err error) bool {
	if err == nil {
		return false
	}
	h.logger.Errorw(h.operation+" failed", "error", err.Error())
	return true
}

// HandleWithCtx 处理错误，带额外上下文
func (h *ErrorHandler) HandleWithCtx(err error, ctx map[string]interface{}) bool {
	if err == nil {
		return false
	}
	LogErrorWithCtx(h.logger, h.operation+" failed", err, ctx)
	return true
}

// LogInfo 记录信息日志
func (h *ErrorHandler) LogInfo(message string) {
	h.logger.Infow(h.operation + ": " + message)
}

// LogWarn 记录警告日志
func (h *ErrorHandler) LogWarn(message string) {
	h.logger.Warnw(h.operation + ": " + message)
}

// FatalOnError 错误时记录并退出（用于初始化阶段）
func FatalOnError(logger *zap.SugaredLogger, err error, message string) {
	if err != nil {
		logger.Fatalw(message, "error", err.Error())
	}
}

// SafeOperation 安全执行操作
func SafeOperation(logger *zap.SugaredLogger, operation string, fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%s panicked: %v", operation, r)
			logger.Errorw(operation+" panicked", "error", err)
		}
	}()
	return fn()
}

// BatchProcess 批量处理，支持错误聚合
type BatchProcess struct {
	Logger    *zap.SugaredLogger
	Operation string
	Errors    []error
}

func NewBatchProcess(logger *zap.SugaredLogger, operation string) *BatchProcess {
	return &BatchProcess{
		Logger:    logger,
		Operation: operation,
		Errors:    make([]error, 0),
	}
}

// Add 处理单个项目
func (b *BatchProcess) Add(index int, err error) {
	if err != nil {
		b.Errors = append(b.Errors, fmt.Errorf("%s [%d]: %w", b.Operation, index, err))
	}
}

// Result 返回处理结果
func (b *BatchProcess) Result() (success int, failed int) {
	if len(b.Errors) == 0 {
		return 0, 0 // 表示没有失败
	}
	return 0, len(b.Errors)
}

// Summary 返回错误摘要
func (b *BatchProcess) Summary() string {
	if len(b.Errors) == 0 {
		return "All operations succeeded"
	}
	return fmt.Sprintf("%d operations failed", len(b.Errors))
}

// LogResult 记录处理结果
func (b *BatchProcess) LogResult(logger *zap.SugaredLogger) {
	if len(b.Errors) == 0 {
		logger.Infow(b.Operation + ": All operations succeeded")
	} else {
		logger.Warnw(b.Operation+" completed with errors", "failed", len(b.Errors))
	}
}
