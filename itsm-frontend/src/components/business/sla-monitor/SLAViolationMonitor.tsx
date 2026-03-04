/**
 * 重构后的 SLAViolationMonitor 组件
 * 使用拆分的子组件和自定义 hooks
 */

import React, { useEffect } from 'react';
import { Card, Button, message, Space } from 'antd';
import { AlertTriangle, Settings, RefreshCw } from 'lucide-react';
import dayjs from 'dayjs';

import { useSLAViolations } from './hooks/useSLAViolations';
import { useSLAStatisticsFrom } from './hooks/useSLAStatistics';
import { useSLARefresh } from './hooks/useSLARefresh';
import { SLAStatisticsCards } from './components/SLAStatisticsCards';
import { SLAFilterPanel } from './components/SLAFilterPanel';
import { SLATable } from './components/SLATable';
import { SLAChartPanel } from './components/SLAChartPanel';
import { SLAViolationDetailModal } from './components/SLAViolationDetailModal';
import type { SLAViolationMonitorProps, SLAViolation, SLAFilters } from './types';

export const SLAViolationMonitor: React.FC<SLAViolationMonitorProps> = ({
  autoRefresh = true,
  refreshInterval = 30000,
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
  } = useSLAViolations({}, autoRefresh);

  const { stats } = useSLAStatisticsFrom(filteredViolations);

  const { isRefreshing, startAutoRefresh, refreshNow } = useSLARefresh();

  const [selectedViolation, setSelectedViolation] = React.useState<SLAViolation | null>(null);
  const [detailVisible, setDetailVisible] = React.useState(false);

  // 自动刷新
  useEffect(() => {
    if (autoRefresh) {
      startAutoRefresh(refreshInterval, refresh);
    }
    return () => {
      // cleanup handled by hook
    };
  }, [autoRefresh, refreshInterval, refresh, startAutoRefresh]);

  const handleView = (violation: SLAViolation) => {
    setSelectedViolation(violation);
    setDetailVisible(true);
  };

  const handleResolve = async () => {
    // 刷新列表
    await refresh();
    setDetailVisible(false);
    message.success('操作成功');
  };

  const handleAcknowledge = async () => {
    await refresh();
    setDetailVisible(false);
    message.success('已确认');
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
            onResolve={handleResolve}
            onAcknowledge={handleAcknowledge}
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
      />
    </div>
  );
};

export default SLAViolationMonitor;
