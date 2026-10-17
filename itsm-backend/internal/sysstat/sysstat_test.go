package sysstat

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadCgroupLimit(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		name string
		body string
		want uint64
	}{
		{"cgroup v2 unlimited keyword", "max\n", 0},
		{"cgroup v2 512MiB", "536870912\n", 536870912},
		{"cgroup v1 unlimited sentinel", "9223372036854771712\n", 0},
		{"empty file", "", 0},
		{"non numeric", "not-a-number\n", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(dir, tc.name)
			require.NoError(t, os.WriteFile(path, []byte(tc.body), 0o600))
			assert.Equal(t, tc.want, readCgroupLimit(path))
		})
	}

	assert.Equal(t, uint64(0), readCgroupLimit(filepath.Join(dir, "missing")))
}

func TestClampPercent(t *testing.T) {
	assert.Equal(t, float64(0), clampPercent(-3))
	assert.Equal(t, float64(42), clampPercent(42))
	assert.Equal(t, float64(100), clampPercent(180))
}

func TestSampleReturnsUptimeCoresAndBoundedPercents(t *testing.T) {
	collector := New(t.TempDir())
	// 制造可观测的 CPU 时间，让区间采样有真实增量而不是 0/常量。
	deadline := time.Now().Add(120 * time.Millisecond)
	busy := 0
	for time.Now().Before(deadline) {
		busy++
	}
	require.Positive(t, busy)

	first := collector.Sample()
	require.True(t, first.UptimeSeconds > 0)
	require.Equal(t, runtime.NumCPU(), first.Cores)
	require.False(t, first.SampledAt.IsZero())

	assert.GreaterOrEqual(t, first.CPUUsagePercent, float64(0))
	assert.LessOrEqual(t, first.CPUUsagePercent, float64(100))
	assert.GreaterOrEqual(t, first.MemoryUsagePercent, float64(0))
	assert.LessOrEqual(t, first.MemoryUsagePercent, float64(100))
	assert.Greater(t, first.MemoryUsedBytes, uint64(0), "Go 运行时堆内存应始终可测量")

	if runtime.GOOS == "linux" {
		assert.Positive(t, memoryLimitBytes(), "Linux 上应至少能从 /proc/meminfo 取到内存分母")
	}

	// 类 Unix 平台必须真的读到 CPU 与磁盘；其他平台应显式标记不可测量。
	if runtime.GOOS != "windows" {
		assert.True(t, first.CPUAvailable)
		assert.True(t, first.DiskAvailable)
		assert.LessOrEqual(t, first.DiskUsagePercent, float64(100))
	} else {
		assert.False(t, first.CPUAvailable)
		assert.False(t, first.DiskAvailable)
	}
}

func TestSampleCPUDeltaAcrossCalls(t *testing.T) {
	collector := New(t.TempDir())
	// 首次采样建立基线，第二次采样才体现区间 CPU 占用率。
	collector.Sample()

	deadline := time.Now().Add(200 * time.Millisecond)
	sum := 0
	for time.Now().Before(deadline) {
		sum++
	}
	after := collector.Sample()

	if runtime.GOOS == "windows" {
		t.Skip("非类 Unix 平台不采集 CPU")
	}
	require.True(t, after.CPUAvailable)
	assert.Greater(t, sum, 0)
	assert.Greater(t, after.CPUUsagePercent, float64(0),
		"忙等之后的区间采样必须测到非零 CPU 时间")
	assert.LessOrEqual(t, after.CPUUsagePercent, float64(100))
}
