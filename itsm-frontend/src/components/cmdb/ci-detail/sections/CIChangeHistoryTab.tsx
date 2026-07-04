/**
 * CI 变更历史标签页组件
 */

import React from 'react';
import { Card, Button, Timeline, Empty, Tag, Space, Typography } from 'antd';
import { History } from 'lucide-react';
import dayjs from 'dayjs';
import type { ChangeHistoryData, ChangeLog } from '../types';
import { ACTION_COLORS } from '../constants';

const { Text } = Typography;

interface CIChangeHistoryTabProps {
  changeHistory: ChangeHistoryData | null;
  historyLoading: boolean;
  onLoad: () => void;
}

export const CIChangeHistoryTab: React.FC<CIChangeHistoryTabProps> = ({
  changeHistory,
  historyLoading,
  onLoad,
}) => {
  return (
    <Card title="CI变更历史" size="small">
      <Button onClick={onLoad} loading={historyLoading} style={{ marginBottom: 16 }}>
        加载变更历史
      </Button>

      {changeHistory && changeHistory.logs && changeHistory.logs.length > 0 ? (
        <Timeline
          items={changeHistory.logs.map((log: ChangeLog) => ({
            color:
              log.action === 'create'
                ? 'green'
                : log.action === 'update'
                  ? 'blue'
                  : log.action === 'delete'
                    ? 'red'
                    : 'gray',
            content: (
              <div>
                <Space>
                  <Tag
                    color={
                      log.action === 'create'
                        ? 'green'
                        : log.action === 'update'
                          ? 'blue'
                          : log.action === 'delete'
                            ? 'red'
                            : 'default'
                    }
                  >
                    {log.action || log.Method}
                  </Tag>
                  <Text>{log.resource}</Text>
                </Space>
                {log.description && (
                  <div>
                    <Text>{log.description}</Text>
                  </div>
                )}
                <div>
                  <Text type="secondary">
                    {log.path} - {log.Method} - {log.StatusCode}
                  </Text>
                </div>
                <div>
                  <Text type="secondary">
                    {log.updated_by && `操作人: ${log.updated_by} - `}
                    {dayjs(log.createdAt || log.created_at || log.updatedAt || log.updated_at || null).format('YYYY-MM-DD HH:mm:ss')}
                  </Text>
                </div>
              </div>
            ),
          }))}
        />
      ) : (
        <Empty description={historyLoading ? '加载中...' : '暂无历史审计记录'} />
      )}
    </Card>
  );
};

CIChangeHistoryTab.displayName = 'CIChangeHistoryTab';
