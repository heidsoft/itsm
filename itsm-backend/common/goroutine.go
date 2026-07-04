package common

import (
	"fmt"
	"runtime/debug"

	"go.uber.org/zap"
)

// defaultLogger 包级默认 logger，用于没有显式传入 logger 的场景
var defaultLogger = zap.NewNop().Sugar()

// SetDefaultLogger 设置包级默认 logger（应在应用启动时调用）
func SetDefaultLogger(l *zap.SugaredLogger) {
	if l != nil {
		defaultLogger = l
	}
}

// GoSafeOptions 安全协程选项
type GoSafeOptions struct {
	Logger   *zap.SugaredLogger
	TaskName string
}

// GoSafe 安全地启动协程，自动捕获panic并记录日志
// 用法：common.GoSafe(func() { ... })
func GoSafe(fn func(), opts ...GoSafeOptions) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()

				// 如果提供了logger，使用结构化日志
				logger := defaultLogger
				taskName := "unknown"
				if len(opts) > 0 && opts[0].Logger != nil {
					logger = opts[0].Logger
					if opts[0].TaskName != "" {
						taskName = opts[0].TaskName
					}
				}
				logger.Errorw(
					"Goroutine panic recovered",
					"task", taskName,
					"error", fmt.Sprintf("%v", r),
					"stack", string(stack),
				)
			}
		}()

		fn()
	}()
}

// GoSafeWithContext 安全地启动协程，带context和logger
// 推荐用于异步任务处理
func GoSafeWithContext(fn func(), logger *zap.SugaredLogger, taskName string) {
	GoSafe(fn, GoSafeOptions{
		Logger:   logger,
		TaskName: taskName,
	})
}

// MustGoSafe 必须安全启动协程（带默认panic处理）
// 适用于简单场景，不需要自定义logger
func MustGoSafe(fn func()) {
	GoSafe(fn)
}
