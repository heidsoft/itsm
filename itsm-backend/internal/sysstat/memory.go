package sysstat

import (
	"os"
	"runtime"
	"strconv"
	"strings"
)

// processMemoryBytes 统计 Go 运行时已占用的堆与栈字节数。相对 m.Sys（向内核申请的
// 全部地址空间），in-use 更接近容器内存限额真正计入的量。
func processMemoryBytes() uint64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.HeapInuse + m.StackInuse
}

// readProcMemTotal 在 Linux 上读取 /proc/meminfo 的 MemTotal；其他平台文件不存在，
// 返回 0 表示「无分母」，由调用方标记内存指标未接入。
func readProcMemTotal() uint64 {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, "MemTotal:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return 0
		}
		kib, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			return 0
		}
		return kib * 1024
	}
	return 0
}
