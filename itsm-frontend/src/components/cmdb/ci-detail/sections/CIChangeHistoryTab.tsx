/**
 * CI 变更历史标签页组件
 */

import React from 'react';
import { Card, Button, Timeline, Empty, Tag, Space, Typography } from 'antd';
import { HistoryOutlined } from '@ant-design/icons';
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
            children: (
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
                <div>
                  <Text type="secondary">
                    {log.path} - {log.Method} - {log.StatusCode}
                  </Text>
                </div>
                <div>
                  <Text type="secondary">
                    {dayjs(log.created_at || log.updated_at).format('YYYY-MM-DD HH:mm:ss')}
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
