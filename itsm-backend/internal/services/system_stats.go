package service

import (
	"context"
	"database/sql"
	"errors"
	"runtime"
	"strconv"
	"strings"
	"time"

	dto "itsm-backend/dto"
	"itsm-backend/internal/sysstat"

	"github.com/prometheus/client_golang/prometheus"
	dto_metric "github.com/prometheus/client_model/go"
	"github.com/redis/go-redis/v9"
)

// 指标可用性口径：前端必须据此区分「测到 0%」与「没接进来」，
// 否则未接入的指标会以假数据的形式出现在系统健康卡上。
const (
	metricMeasured     = "measured"
	metricUnavailable  = "unavailable"
	metricUnconfigured = "unconfigured"
)

// 单次仪表盘请求允许的采集预算。DB/Redis 探测属于旁路观测，不能拖慢页面。
const systemStatsProbeTimeout = 2 * time.Second

// 低于该区间的增量窗口不足以支撑一个可信的速率读数，改用累计口径。
const minHTTPSampleWindowSeconds = 1.0

// httpTotals 是从 Prometheus 默认注册表读到的 HTTP 累计量（自进程启动以来）。
type httpTotals struct {
	at            time.Time
	requests      float64
	serverErrors  float64
	durationCount float64
	durationSum   float64
}

// hostMetricSources 汇总一次系统健康采样的各个数据源。
// 单元测试可替换 gatherer / redis 以摆脱真实注册表和外部依赖。
type hostMetricSources struct {
	gatherer prometheus.Gatherer
	redis    *redis.Client
}

// buildSystemStats 把主机指标、HTTP 指标、数据库与缓存探测拼成 API 响应。
// previous 是上一次采样的累计量，用于给出区间口径的时延/速率/错误率；
// 第二个返回值是下一次调用应当使用的基线。
func buildSystemStats(
	ctx context.Context,
	collector *sysstat.Collector,
	sources hostMetricSources,
	db *sql.DB,
	previous *httpTotals,
) (*dto.SystemStatsResponse, *httpTotals, error) {
	host := collector.Sample()

	resp := &dto.SystemStatsResponse{
		Uptime:        host.UptimeSeconds,
		CPUUsage:      host.CPUUsagePercent,
		MemoryUsage:   host.MemoryUsagePercent,
		DiskUsage:     host.DiskUsagePercent,
		Cores:         host.Cores,
		Goroutines:    runtime.NumGoroutine(),
		SampledAt:     host.SampledAt,
		DBConnections: databaseConnectionCount(db),
		Availability: dto.SystemMetricAvailability{
			CPU:    availabilityOf(host.CPUAvailable),
			Memory: availabilityOf(host.MemoryAvailable),
			Disk:   availabilityOf(host.DiskAvailable),
		},
	}

	totals, totalsErr := aggregateHTTPTotals(sources.gatherer)
	if totalsErr != nil {
		return nil, previous, totalsErr
	}
	current := totals
	baseline := &current
	if current.requests > 0 || current.durationCount > 0 {
		resp.Availability.HTTP = metricMeasured
		baseline = applyHTTPRates(resp, previous, &current)
	} else {
		// 注册表里一个请求样本都没有：说明中间件尚未产生数据，不能显示 0ms/0%。
		resp.Availability.HTTP = metricUnavailable
	}

	if size, ok := queryDatabaseSize(ctx, db); ok {
		resp.DBSize = size
		resp.Availability.DBSize = metricMeasured
	} else {
		resp.Availability.DBSize = metricUnavailable
	}

	if sources.redis == nil {
		resp.Availability.Cache = metricUnconfigured
	} else if rate, ok := queryRedisHitRate(ctx, sources.redis); ok {
		resp.CacheHitRate = rate
		resp.Availability.Cache = metricMeasured
	} else {
		resp.Availability.Cache = metricUnavailable
	}

	return resp, baseline, nil
}

// applyHTTPRates 用「距上一次采样的增量」计算时延、QPS 与错误率。
// 首次采样、或区间不足 minHTTPSampleWindowSeconds 时退回启动以来的累计口径，
// 并通过 metricsWindowSeconds 告诉前端当前数值覆盖的时间窗，
// 避免把区间值和累计值混成同一个口径。
//
// 返回值是本次采样之后应当保留的基线：区间不足时沿用 previous，让窗口继续累积。
// 并发轮询（健康卡与系统配置页共用同一个基线）否则会把区间压到毫秒级，
// 单条请求就能算出几十 QPS 的尖峰。
func applyHTTPRates(resp *dto.SystemStatsResponse, previous, current *httpTotals) *httpTotals {
	elapsed := 0.0
	if previous != nil && previous.at.Before(current.at) {
		elapsed = current.at.Sub(previous.at).Seconds()
	}
	if elapsed < minHTTPSampleWindowSeconds {
		resp.RequestsPerSecond = perSecond(current.requests, resp.Uptime)
		resp.AvgResponseTime = averageMillis(current.durationSum, current.durationCount)
		resp.ErrorRate = errorPercent(current.serverErrors, current.requests)
		resp.MetricsWindowSeconds = resp.Uptime
		if previous != nil {
			return previous
		}
		return current
	}

	deltaRequests := max(current.requests-previous.requests, 0)
	deltaErrors := max(current.serverErrors-previous.serverErrors, 0)
	deltaDurationCount := max(current.durationCount-previous.durationCount, 0)
	deltaDurationSum := max(current.durationSum-previous.durationSum, 0)

	resp.MetricsWindowSeconds = elapsed
	resp.RequestsPerSecond = perSecond(deltaRequests, elapsed)
	if deltaDurationCount > 0 {
		resp.AvgResponseTime = averageMillis(deltaDurationSum, deltaDurationCount)
	} else {
		resp.AvgResponseTime = averageMillis(current.durationSum, current.durationCount)
	}
	if deltaRequests > 0 {
		resp.ErrorRate = errorPercent(deltaErrors, deltaRequests)
	} else {
		resp.ErrorRate = errorPercent(current.serverErrors, current.requests)
	}
	return current
}

func perSecond(count, seconds float64) float64 {
	if seconds <= 0 {
		return 0
	}
	return count / seconds
}

func averageMillis(sumSeconds, count float64) float64 {
	if count <= 0 {
		return 0
	}
	return sumSeconds / count * 1000
}

func errorPercent(errors, total float64) float64 {
	if total <= 0 {
		return 0
	}
	return errors / total * 100
}

// SetRedisClient 注入 Redis 客户端用于采集缓存命中率。
// 未注入（未配置 Redis 或连接不可用）时缓存指标返回 unconfigured，而不是 0%。
func (s *DashboardService) SetRedisClient(client *redis.Client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.metricSources.redis = client
}

// aggregateHTTPTotals 从注册表读取请求计数与耗时直方图。错误率只算 5xx：
// 4xx 是调用方参数/权限问题，不代表服务本身不健康。
func aggregateHTTPTotals(gatherer prometheus.Gatherer) (httpTotals, error) {
	if gatherer == nil {
		return httpTotals{}, errors.New("metrics gatherer is not configured")
	}
	families, err := gatherer.Gather()
	if err != nil {
		return httpTotals{}, err
	}

	totals := httpTotals{at: time.Now()}
	for _, family := range families {
		switch family.GetName() {
		case "itsm_http_requests_total":
			for _, metric := range family.GetMetric() {
				value := metric.GetCounter().GetValue()
				totals.requests += value
				if strings.HasPrefix(metricLabel(metric, "status"), "5") {
					totals.serverErrors += value
				}
			}
		case "itsm_http_request_duration_seconds":
			for _, metric := range family.GetMetric() {
				histogram := metric.GetHistogram()
				totals.durationCount += float64(histogram.GetSampleCount())
				totals.durationSum += histogram.GetSampleSum()
			}
		}
	}
	return totals, nil
}

func metricLabel(metric *dto_metric.Metric, name string) string {
	for _, pair := range metric.GetLabel() {
		if pair.GetName() == name {
			return pair.GetValue()
		}
	}
	return ""
}

func databaseConnectionCount(db *sql.DB) int {
	if db == nil {
		return 0
	}
	stats := db.Stats()
	return stats.InUse + stats.Idle
}

// queryDatabaseSize 先按 PostgreSQL 探测（生产口径），再退回 SQLite 页数乘页大小
// （本地开发与测试库）。两者都测不到时返回 false，由可用性字段标注未接入。
func queryDatabaseSize(ctx context.Context, db *sql.DB) (int64, bool) {
	if db == nil {
		return 0, false
	}
	probeCtx, cancel := context.WithTimeout(ctx, systemStatsProbeTimeout)
	defer cancel()

	var pgSize sql.NullInt64
	if err := db.QueryRowContext(probeCtx, "SELECT pg_database_size(current_database())").Scan(&pgSize); err == nil && pgSize.Valid && pgSize.Int64 > 0 {
		return pgSize.Int64, true
	}

	var sqliteSize sql.NullInt64
	err := db.QueryRowContext(probeCtx,
		"SELECT (SELECT page_count FROM pragma_page_count()) * (SELECT page_size FROM pragma_page_size())").Scan(&sqliteSize)
	if err == nil && sqliteSize.Valid && sqliteSize.Int64 > 0 {
		return sqliteSize.Int64, true
	}
	return 0, false
}

// queryRedisHitRate 解析 INFO stats 的 keyspace_hits/keyspace_misses。
// Redis 没有任何命中记录时返回 false，避免把「无流量」渲染成 0% 命中率告警。
func queryRedisHitRate(ctx context.Context, client *redis.Client) (float64, bool) {
	if client == nil {
		return 0, false
	}
	probeCtx, cancel := context.WithTimeout(ctx, systemStatsProbeTimeout)
	defer cancel()

	info, err := client.Info(probeCtx, "stats").Result()
	if err != nil {
		return 0, false
	}
	hits, hasHits := redisInfoFloat(info, "keyspace_hits")
	misses, hasMisses := redisInfoFloat(info, "keyspace_misses")
	if !hasHits || !hasMisses {
		return 0, false
	}
	total := hits + misses
	if total <= 0 {
		return 0, false
	}
	return hits / total * 100, true
}

func redisInfoFloat(info, field string) (float64, bool) {
	for _, line := range strings.Split(info, "\n") {
		key, value, found := strings.Cut(strings.TrimSpace(line), ":")
		if !found || key != field {
			continue
		}
		parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		if err != nil {
			return 0, false
		}
		return parsed, true
	}
	return 0, false
}

func availabilityOf(measured bool) string {
	if measured {
		return metricMeasured
	}
	return metricUnavailable
}
