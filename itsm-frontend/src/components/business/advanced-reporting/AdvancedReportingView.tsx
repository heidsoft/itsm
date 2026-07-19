'use client';

import { Alert, Button, Card, Empty, Skeleton, Space, Typography } from 'antd';
import { BarChart3, RefreshCw } from 'lucide-react';
import type { AdvancedReportingViewProps } from './types';

export function AdvancedReportingView({
  reports,
  loading,
  error,
  onReload,
  onOpenReport,
}: AdvancedReportingViewProps) {
  return (
    <Card
      title={
        <Space>
          <BarChart3 size={18} />
          <span>报表中心</span>
        </Space>
      }
      extra={
        <Button icon={<RefreshCw size={16} />} onClick={onReload} loading={loading}>
          刷新
        </Button>
      }
    >
      {error && (
        <Alert
          type="error"
          showIcon
          title="无法加载报表目录"
          description={error}
          action={<Button onClick={onReload}>重试</Button>}
          style={{ marginBottom: 16 }}
        />
      )}
      {loading ? (
        <Skeleton active />
      ) : reports.length === 0 ? (
        <Empty description="暂无可用报表" />
      ) : (
        <Space orientation="vertical" size={0} style={{ width: '100%' }}>
          {reports.map(report => (
            <div
              key={report.id}
              className="flex items-center justify-between border-b border-gray-100 py-3 last:border-b-0"
            >
              <div>
                <Typography.Text strong>{report.name}</Typography.Text>
                <br />
                <Typography.Text type="secondary">{report.path}</Typography.Text>
              </div>
              {onOpenReport && (
                <Button type="link" onClick={() => onOpenReport(report)}>
                  查看
                </Button>
              )}
            </div>
          ))}
        </Space>
      )}
    </Card>
  );
}
