package common

import (
	"fmt"
	"runtime/debug"

	"go.uber.org/zap"
)

// GoSafeOptions 安全协程选项
type GoSafeOptions struct {
	Logger  *zap.SugaredLogger
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
				if len(opts) > 0 && opts[0].Logger != nil {
					logger := opts[0].Logger
					taskName := opts[0].TaskName
					if taskName == "" {
						taskName = "unknown"
					}
					logger.Errorw("Goroutine panic recovered",
						"task", taskName,
						"error", fmt.Sprintf("%v", r),
						"stack", string(stack),
					)
				} else {
					// 回退到标准输出
					fmt.Printf("[PANIC] Goroutine panic recovered: %v\n", r)
					fmt.Printf("[PANIC] Stack trace:\n%s\n", stack)
				}
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
