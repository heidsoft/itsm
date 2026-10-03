/**
 * CI 变更历史标签页组件
 *
 * 数据契约：GET /api/v1/cmdb/cis/:id/history 返回平台五键信封
 * `{items,total,page,pageSize,totalPages}`，条目字段是 `dto.CIHistoryResponse`。
 */

import React from 'react';
import { Alert, Button, Card, Empty, Space, Tag, Timeline, Typography } from 'antd';
import dayjs from 'dayjs';
import type { ChangeHistoryData } from '../types';
import type { CIHistoryItem } from '@/types/biz/cmdb';
import { ciOperationColor, ciOperationLabel } from '../constants';

const { Text } = Typography;

interface CIChangeHistoryTabProps {
  changeHistory: ChangeHistoryData | null;
  historyLoading: boolean;
  historyError: boolean;
  onLoad: () => void;
  onPageChange: (page: number) => void;
}

export const CIChangeHistoryTab: React.FC<CIChangeHistoryTabProps> = ({
  changeHistory,
  historyLoading,
  historyError,
  onLoad,
  onPageChange,
}) => {
  const items = changeHistory?.items ?? [];
  const total = changeHistory?.total ?? 0;
  const page = changeHistory?.page ?? 1;
  const totalPages = changeHistory?.totalPages ?? 0;

  return (
    <Card
      title="CI变更历史"
      size="small"
      extra={
        <Button size="small" onClick={onLoad} loading={historyLoading} aria-label="刷新变更历史">
          刷新
        </Button>
      }
    >
      {historyError ? (
        <Alert
          type="error"
          showIcon
          title="变更历史加载失败"
          description="后端未能返回该配置项的历史记录，请重试；若持续失败请确认当前账号具备 cmdb:read 权限。"
        />
      ) : items.length === 0 ? (
        <Empty description={historyLoading ? '加载中...' : '暂无历史审计记录'} />
      ) : (
        <>
          <Timeline
            items={items.map((log: CIHistoryItem) => ({
              color: ciOperationColor(log.operation),
              content: (
                <div>
                  <Space size="small" wrap>
                    <Tag color={ciOperationColor(log.operation)}>{ciOperationLabel(log.operation)}</Tag>
                    <Text type="secondary">v{log.version}</Text>
                    {log.operatorName && <Text type="secondary">操作人: {log.operatorName}</Text>}
                    <Text type="secondary">
                      {dayjs(log.createdAt).format('YYYY-MM-DD HH:mm:ss')}
                    </Text>
                  </Space>
                  {log.remark && (
                    <div>
                      <Text>{log.remark}</Text>
                    </div>
                  )}
                  {log.changedFields && log.changedFields.length > 0 && (
                    <div>
                      <Text type="secondary">变更字段: {log.changedFields.join(', ')}</Text>
                    </div>
                  )}
                </div>
              ),
            }))}
          />
          <Space size="small">
            <Text type="secondary">
              共 {total} 条，第 {page}/{Math.max(totalPages, 1)} 页
            </Text>
            <Button
              size="small"
              onClick={() => onPageChange(page - 1)}
              disabled={page <= 1 || historyLoading}
              aria-label="变更历史上一页"
            >
              上一页
            </Button>
            <Button
              size="small"
              onClick={() => onPageChange(page + 1)}
              disabled={page >= totalPages || historyLoading}
              aria-label="变更历史下一页"
            >
              下一页
            </Button>
          </Space>
        </>
      )}
    </Card>
  );
};

CIChangeHistoryTab.displayName = 'CIChangeHistoryTab';
