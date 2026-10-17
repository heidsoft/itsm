//go:build !unix

package sysstat

// 非类 Unix 平台（如 Windows CI 构建目标）没有 Getrusage/Statfs 的等价系统调用，
// 相应指标标记为不可测量，由调用方展示「未接入」，不得伪造数值。

func processCPUTimeSecs() (float64, bool) { return 0, false }

func diskUsage(string) (uint64, uint64, bool) { return 0, 0, false }
