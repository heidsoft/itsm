// 运行时指标展示口径：系统健康卡与系统配置页共用，避免同一数值出现两套读法。

export const formatUptime = (seconds: number): string => {
  if (!Number.isFinite(seconds) || seconds <= 0) return '—';
  const d = Math.floor(seconds / 86400);
  const h = Math.floor((seconds % 86400) / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  if (d > 0) return `${d}天 ${h}小时`;
  if (h > 0) return `${h}小时 ${m}分钟`;
  if (m > 0) return `${m}分钟`;
  // 刚重启的后端不足 1 分钟，取整会显示成「0分钟」，看起来像没有运行时长。
  return `${Math.floor(seconds)}秒`;
};

// 采样区间由后端决定，可能只有几毫秒；取整会显示成「0s」，看起来像没有数据。
export const formatMetricWindow = (seconds: number): string => {
  if (!Number.isFinite(seconds) || seconds <= 0) return '—';
  if (seconds < 1) return `${Math.round(seconds * 1000)}ms`;
  if (seconds < 10) return `${seconds.toFixed(1)}s`;
  if (seconds < 60) return `${seconds.toFixed(0)}s`;
  if (seconds < 3600) return `${(seconds / 60).toFixed(1)}min`;
  return `${(seconds / 3600).toFixed(1)}h`;
};
