// Package sysstat 采集进程与主机运行指标，供仪表盘系统健康卡使用。
//
// CPU 占用率需要两次采样才能算出区间值，因此采集器必须跨请求复用，
// 而不是每次请求新建；同一份口径也由 /dashboard/stats/system 单一出口提供。
package sysstat

import (
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Host 是一次采样得到的运行指标。Available 为 false 表示该指标在当前运行环境
// 无法测量（例如 macOS 开发机没有 cgroup 内存上限），调用方必须据此展示
// 「未接入」而不是 0%。
type Host struct {
	SampledAt time.Time

	UptimeSeconds float64
	Cores         int

	CPUUsagePercent    float64
	CPUAvailable       bool
	MemoryUsagePercent float64
	MemoryAvailable    bool
	MemoryUsedBytes    uint64
	MemoryLimitBytes   uint64
	DiskUsagePercent   float64
	DiskAvailable      bool
	DiskPath           string
}

// Collector 持有上一次采样，用于把累计值（进程 CPU 时间）换算成区间占用率。
// 采样是并发安全的：多个端点可以共享同一个 Collector。
type Collector struct {
	mu         sync.Mutex
	startedAt  time.Time
	diskPath   string
	lastSample *sample
}

type sample struct {
	at      time.Time
	cpuSecs float64
}

// New 创建采集器。diskPath 为空时使用进程工作目录。
func New(diskPath string) *Collector {
	if diskPath == "" {
		diskPath = os.Getenv("ITSM_DATA_DIR")
	}
	if diskPath == "" {
		if wd, err := os.Getwd(); err == nil {
			diskPath = wd
		}
	}
	return &Collector{startedAt: time.Now(), diskPath: diskPath}
}

// Sample 读取一次运行指标。CPU 占用率取「距上一次采样的区间」内的进程 CPU 时间占比；
// 首次采样没有区间，退化为进程启动以来的平均值，并在 CPUAvailable 为 true 时给出。
func (c *Collector) Sample() Host {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	uptime := now.Sub(c.startedAt).Seconds()

	host := Host{
		SampledAt:        now,
		UptimeSeconds:    uptime,
		Cores:            runtime.NumCPU(),
		DiskPath:         c.diskPath,
		MemoryUsedBytes:  processMemoryBytes(),
		MemoryLimitBytes: memoryLimitBytes(),
	}
	if host.MemoryLimitBytes > 0 && host.MemoryUsedBytes > 0 {
		host.MemoryUsagePercent = clampPercent(float64(host.MemoryUsedBytes) / float64(host.MemoryLimitBytes) * 100)
		host.MemoryAvailable = true
	}

	if cpuSecs, ok := processCPUTimeSecs(); ok {
		host.CPUAvailable = true
		if c.lastSample != nil {
			deltaWall := now.Sub(c.lastSample.at).Seconds()
			deltaCPU := cpuSecs - c.lastSample.cpuSecs
			if deltaWall > 0.05 && deltaCPU >= 0 {
				host.CPUUsagePercent = clampPercent(deltaCPU / (deltaWall * float64(host.Cores)) * 100)
			}
		}
		if host.CPUUsagePercent == 0 && uptime > 0 {
			// 首次采样或区间过短：用启动以来的平均值，仍是真实累计量换算，不是常量。
			host.CPUUsagePercent = clampPercent(cpuSecs / (uptime * float64(host.Cores)) * 100)
		}
		c.lastSample = &sample{at: now, cpuSecs: cpuSecs}
	}

	if total, free, ok := diskUsage(c.diskPath); ok && total > 0 {
		used := total - free
		host.DiskUsagePercent = clampPercent(float64(used) / float64(total) * 100)
		host.DiskAvailable = true
	}

	return host
}

func clampPercent(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}

// memoryLimitBytes 依次尝试 cgroup v2、cgroup v1 的内存上限；没有上限（容器未配置
// 内存限制或本机开发环境）时退回 /proc/meminfo 的 MemTotal；都拿不到返回 0。
func memoryLimitBytes() uint64 {
	if v := readCgroupLimit("/sys/fs/cgroup/memory.max"); v > 0 {
		return v
	}
	if v := readCgroupLimit("/sys/fs/cgroup/memory/memory.limit_in_bytes"); v > 0 {
		return v
	}
	return readProcMemTotal()
}

// readCgroupLimit 解析单值文件；"max" 或接近无符号上限的哨兵值表示未设置限制。
func readCgroupLimit(path string) uint64 {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" || trimmed == "max" {
		return 0
	}
	v, err := strconv.ParseUint(trimmed, 10, 64)
	if err != nil {
		return 0
	}
	// 内核用 0x7FFFFFFFFFFFFFFF 之类的极大值表示「无限制」。
	const unlimitedThreshold = uint64(1) << 62
	if v >= unlimitedThreshold {
		return 0
	}
	return v
}
