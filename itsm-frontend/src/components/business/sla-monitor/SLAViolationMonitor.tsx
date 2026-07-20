/**
 * 重构后的 SLAViolationMonitor 组件
 * 使用拆分的子组件和自定义 hooks
 */

import React, { useEffect } from 'react';
import { Card, Button, message, Space } from 'antd';
import { AlertTriangle, RefreshCw } from 'lucide-react';

import { useSLAViolations } from './hooks/useSLAViolations';
import { useSLAStatisticsFrom } from './hooks/useSLAStatistics';
import { useSLARefresh } from './hooks/useSLARefresh';
import { SLAStatisticsCards } from './components/SLAStatisticsCards';
import { SLAFilterPanel } from './components/SLAFilterPanel';
import { SLATable } from './components/SLATable';
import { SLAChartPanel } from './components/SLAChartPanel';
import { SLAViolationDetailModal } from './components/SLAViolationDetailModal';
import type { SLAViolationMonitorProps, SLAViolation } from './types';
import { acknowledgeSLAViolation, resolveSLAViolation } from './services/sla-monitor-service';

export const SLAViolationMonitor: React.FC<SLAViolationMonitorProps> = ({
  autoRefresh = true,
  refreshInterval = 30000,
  canManage = false,
  onViolationUpdate,
}) => {
  const {
    filteredViolations,
    filters,
    loading,
    setFilters,
    refresh,
    selectedRowKeys,
    selectRow,
  } = useSLAViolations({}, false);

  const { stats } = useSLAStatisticsFrom(filteredViolations);

  const { isRefreshing, startAutoRefresh, refreshNow } = useSLARefresh();

  const [selectedViolation, setSelectedViolation] = React.useState<SLAViolation | null>(null);
  const [detailVisible, setDetailVisible] = React.useState(false);
  const [actionLoading, setActionLoading] = React.useState(false);

  // 自动刷新
  useEffect(() => {
    if (autoRefresh) {
      startAutoRefresh(refreshInterval, refresh);
    } else {
      void refresh();
    }
  }, [autoRefresh, refreshInterval, refresh, startAutoRefresh]);

  const handleView = (violation: SLAViolation) => {
    setSelectedViolation(violation);
    setDetailVisible(true);
  };

  const handleResolve = async (violation: SLAViolation) => {
    setActionLoading(true);
    try {
      await resolveSLAViolation(violation.id);
      await refresh();
      setDetailVisible(false);
      message.success('已标记为已解决');
      onViolationUpdate?.(violation);
    } catch (cause) {
      message.error(cause instanceof Error ? cause.message : '解决 SLA 违规失败');
    } finally {
      setActionLoading(false);
    }
  };

  const handleAcknowledge = async (violation: SLAViolation) => {
    setActionLoading(true);
    try {
      await acknowledgeSLAViolation(violation.id);
      await refresh();
      setDetailVisible(false);
      message.success('已确认');
      onViolationUpdate?.(violation);
    } catch (cause) {
      message.error(cause instanceof Error ? cause.message : '确认 SLA 违规失败');
    } finally {
      setActionLoading(false);
    }
  };

  const handleRefresh = async () => {
    await refreshNow(refresh);
  };

  return (
    <div>
      <Card
        title={
          <Space>
            <AlertTriangle size={18} className="text-orange-500" />
            <span>SLA 违规监控</span>
          </Space>
        }
        extra={
          <Button
            icon={<RefreshCw size={14} />}
            onClick={handleRefresh}
            loading={isRefreshing}
          >
            刷新
          </Button>
        }
      >
        <SLAStatisticsCards stats={stats} loading={loading} />

        <div style={{ marginTop: 24 }}>
          <SLAFilterPanel
            filters={filters}
            onFiltersChange={setFilters}
            onRefresh={handleRefresh}
            loading={loading}
          />

          <SLATable
            violations={filteredViolations}
            loading={loading}
            selectedRowKeys={selectedRowKeys}
            onRowSelect={selectRow}
            onView={handleView}
            onResolve={violation => void handleResolve(violation)}
            onAcknowledge={violation => void handleAcknowledge(violation)}
          />

          <div style={{ marginTop: 16 }}>
            <SLAChartPanel violations={filteredViolations} />
          </div>
        </div>
      </Card>

      <SLAViolationDetailModal
        violation={selectedViolation}
        visible={detailVisible}
        onClose={() => setDetailVisible(false)}
        onResolve={handleResolve}
        onAcknowledge={handleAcknowledge}
        canManage={canManage}
        actionLoading={actionLoading}
      />
    </div>
  );
};

export default SLAViolationMonitor;
