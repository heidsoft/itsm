package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"itsm-backend/dto"
	"itsm-backend/internal/sysstat"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// httpMetricsRegistry 用生产同名指标搭一个可控注册表，
// 让 buildSystemStats 读的是真的 Gather() 结果而不是内部结构。
func httpMetricsRegistry(t *testing.T, statusCounts map[string]float64, histogramCount int, histogramSum float64) prometheus.Gatherer {
	t.Helper()

	reg := prometheus.NewRegistry()
	counter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "itsm_http_requests_total",
		Help: "Total number of HTTP requests",
	}, []string{"method", "path", "status"})
	histogram := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "itsm_http_request_duration_seconds",
		Help:    "HTTP request duration in seconds",
		Buckets: []float64{0.001, 1, 10},
	}, []string{"method", "path", "status"})
	require.NoError(t, reg.Register(counter))
	require.NoError(t, reg.Register(histogram))

	for status, value := range statusCounts {
		counter.WithLabelValues("GET", "/api/v1/tickets", status).Add(value)
	}
	if histogramCount > 0 {
		// 每条样本耗时相同，总数即 histogramSum，便于断言平均时延。
		for range histogramCount {
			histogram.WithLabelValues("GET", "/api/v1/tickets", "200").Observe(histogramSum / float64(histogramCount))
		}
	}
	return reg
}

func TestBuildSystemStatsMarksHTTPUnavailableWithoutSamples(t *testing.T) {
	sources := hostMetricSources{gatherer: prometheus.NewRegistry()}

	resp, current, err := buildSystemStats(context.Background(), sysstat.New(t.TempDir()), sources, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, current)

	assert.Equal(t, metricUnavailable, resp.Availability.HTTP)
	assert.Zero(t, resp.RequestsPerSecond)
	assert.Zero(t, resp.ErrorRate)
	// 未注入 db / redis 时必须显式标注，而不是 0 MB / 0% 命中率。
	assert.Equal(t, metricUnavailable, resp.Availability.DBSize)
	assert.Equal(t, metricUnconfigured, resp.Availability.Cache)
	assert.Zero(t, resp.DBSize)
	assert.Zero(t, resp.CacheHitRate)
}

func TestBuildSystemStatsFirstSampleUsesUptimeWindow(t *testing.T) {
	sources := hostMetricSources{gatherer: httpMetricsRegistry(t,
		map[string]float64{"200": 180, "500": 20}, 200, 4)}

	resp, _, err := buildSystemStats(context.Background(), sysstat.New(t.TempDir()), sources, nil, nil)
	require.NoError(t, err)

	require.Equal(t, metricMeasured, resp.Availability.HTTP)
	assert.InDelta(t, resp.Uptime, resp.MetricsWindowSeconds, 1)
	assert.InDelta(t, 200/resp.Uptime, resp.RequestsPerSecond, 1)
	assert.InDelta(t, 10, resp.ErrorRate, 0.001, "错误率只算 5xx，4xx 不算服务故障")
	assert.InDelta(t, 20, resp.AvgResponseTime, 0.001)
}

func TestBuildSystemStatsUsesDeltaSincePreviousSample(t *testing.T) {
	sources := hostMetricSources{gatherer: httpMetricsRegistry(t,
		map[string]float64{"200": 270, "500": 30}, 300, 6)}
	previous := &httpTotals{
		at:            time.Now().Add(-10 * time.Second),
		requests:      100,
		serverErrors:  10,
		durationCount: 100,
		durationSum:   2,
	}

	resp, current, err := buildSystemStats(context.Background(), sysstat.New(t.TempDir()), sources, nil, previous)
	require.NoError(t, err)
	require.NotNil(t, current)

	// 区间约 10s：新增 200 条请求、20 条 5xx、200 个耗时样本、4s 总耗时。
	assert.InDelta(t, 10, resp.MetricsWindowSeconds, 0.5)
	assert.InDelta(t, 20, resp.RequestsPerSecond, 1.5)
	assert.InDelta(t, 10, resp.ErrorRate, 1.5)
	assert.InDelta(t, 20, resp.AvgResponseTime, 1.5)
	assert.NotSame(t, previous, current, "区间足够时应推进基线")
}

func TestBuildSystemStatsKeepsBaselineWhenWindowIsTooShort(t *testing.T) {
	sources := hostMetricSources{gatherer: httpMetricsRegistry(t,
		map[string]float64{"200": 180, "500": 20}, 200, 4)}
	// 健康卡与系统配置页共用同一份基线，两个轮询者可以把区间压到几十毫秒，
	// 此时单条请求就能算出几十 QPS 的尖峰。
	previous := &httpTotals{
		at:            time.Now().Add(-20 * time.Millisecond),
		requests:      100,
		serverErrors:  10,
		durationCount: 100,
		durationSum:   2,
	}

	resp, baseline, err := buildSystemStats(context.Background(), sysstat.New(t.TempDir()), sources, nil, previous)
	require.NoError(t, err)

	assert.Same(t, previous, baseline, "区间不足时保留旧基线，让窗口继续累积")
	assert.InDelta(t, resp.Uptime, resp.MetricsWindowSeconds, 1, "口径退回启动以来累计值")
	assert.InDelta(t, 10, resp.ErrorRate, 0.001, "用累计错误率而不是 20ms 区间的样本")
	assert.Positive(t, resp.RequestsPerSecond)
}

func TestBuildSystemStatsIgnoresClientErrorsInErrorRate(t *testing.T) {
	sources := hostMetricSources{gatherer: httpMetricsRegistry(t,
		map[string]float64{"200": 100, "404": 50}, 150, 3)}

	resp, _, err := buildSystemStats(context.Background(), sysstat.New(t.TempDir()), sources, nil, nil)
	require.NoError(t, err)

	assert.Equal(t, metricMeasured, resp.Availability.HTTP)
	assert.Zero(t, resp.ErrorRate)
	assert.InDelta(t, 20, resp.AvgResponseTime, 0.001)
}

func TestBuildSystemStatsRequiresGatherer(t *testing.T) {
	_, _, err := buildSystemStats(context.Background(), sysstat.New(t.TempDir()), hostMetricSources{}, nil, nil)
	require.Error(t, err)
}

func TestBuildSystemStatsReportsHostAvailabilityConsistently(t *testing.T) {
	sources := hostMetricSources{gatherer: prometheus.NewRegistry()}

	resp, _, err := buildSystemStats(context.Background(), sysstat.New(t.TempDir()), sources, nil, nil)
	require.NoError(t, err)

	// 可用性标注必须与数值口径一致：未测到的指标不能带数值进入评级。
	if resp.Availability.CPU != metricMeasured {
		assert.Zero(t, resp.CPUUsage)
	}
	if resp.Availability.Disk != metricMeasured {
		assert.Zero(t, resp.DiskUsage)
	}
	assert.Positive(t, resp.Cores)
	assert.False(t, resp.SampledAt.IsZero())
}

func TestDashboardServiceGetSystemStatsIsStableAcrossCalls(t *testing.T) {
	// 走生产入口：handler -> DashboardService.GetSystemStats，采集器必须跨请求复用，
	// 否则每次都是首次采样，CPU 区间占用率永远拿不到增量基线。
	svc := NewDashboardServiceWithDB(nil, nil, zap.NewNop().Sugar())

	first, err := svc.GetSystemStats(context.Background())
	require.NoError(t, err)
	second, err := svc.GetSystemStats(context.Background())
	require.NoError(t, err)

	assert.False(t, first.SampledAt.After(second.SampledAt))
	assert.Equal(t, metricUnavailable, second.Availability.DBSize)
	assert.Positive(t, second.Goroutines, "goroutines 是运行时真实计数，系统配置页直接展示")
}

func TestSystemStatsResponseJSONContract(t *testing.T) {
	payload, err := json.Marshal(dto.SystemStatsResponse{
		Availability: dto.SystemMetricAvailability{
			CPU: metricMeasured, Memory: metricUnavailable, Disk: metricMeasured,
			HTTP: metricUnavailable, DBSize: metricUnavailable, Cache: metricUnconfigured,
		},
	})
	require.NoError(t, err)

	var fields map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(payload, &fields))

	for _, key := range []string{
		"uptime", "cpuUsage", "memoryUsage", "diskUsage",
		"avgResponseTime", "requestsPerSecond", "errorRate", "metricsWindowSeconds",
		"dbConnections", "dbSize", "cacheHitRate", "cores", "goroutines", "sampledAt", "availability",
	} {
		require.Contains(t, fields, key)
	}

	var availability map[string]string
	require.NoError(t, json.Unmarshal(fields["availability"], &availability))
	assert.Equal(t, map[string]string{
		"cpu": metricMeasured, "memory": metricUnavailable, "disk": metricMeasured,
		"http": metricUnavailable, "dbSize": metricUnavailable, "cache": metricUnconfigured,
	}, availability)
}
