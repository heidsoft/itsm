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
      title="CI change history"
      size="small"
      extra={
        <Button size="small" onClick={onLoad} loading={historyLoading} aria-label="Refresh change history">

          Refresh
        </Button>
      }
    >
      {historyError ? (
        <Alert
          type="error"
          showIcon
          title="Change history loading failed"
          description="Loading failed, please try again or confirm you have cmdb:read permissions"
        />
      ) : items.length === 0 ? (
        <Empty description={historyLoading ? 'Loading...' : 'No historical audit record yet'} />
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
                    {log.operatorName && <Text type="secondary">Operator: {log.operatorName}</Text>}
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
                      <Text type="secondary">Change fields: {log.changedFields.join(', ')}</Text>
                    </div>
                  )}
                </div>
              ),
            }))}
          />
          <Space size="small">
            <Text type="secondary">

              Total {total}  Article, Article {page}/{Math.max(totalPages, 1)}  Page
            </Text>
            <Button
              size="small"
              onClick={() => onPageChange(page - 1)}
              disabled={page <= 1 || historyLoading}
              aria-label="Change history previous page"
            >

              Previous page
            </Button>
            <Button
              size="small"
              onClick={() => onPageChange(page + 1)}
              disabled={page >= totalPages || historyLoading}
              aria-label="Change history next page"
            >

              Next page
            </Button>
          </Space>
        </>
      )}
    </Card>
  );
};

CIChangeHistoryTab.displayName = 'CIChangeHistoryTab';
