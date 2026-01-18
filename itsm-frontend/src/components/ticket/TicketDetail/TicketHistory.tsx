'use client';

import React from 'react';
import {
  History,
} from 'lucide-react';
import {
  Typography,
  Timeline,
} from 'antd';

const { Title, Text } = Typography;

interface HistoryRecord {
  id: number;
  user?: {
    name?: string;
  };
  field_name: string;
  change_reason?: string;
  old_value: string;
  new_value: string;
  changed_at: string;
}

interface TicketHistoryProps {
  history: HistoryRecord[];
  formatDateTime: (dateString: string) => string;
}

export const TicketHistory: React.FC<TicketHistoryProps> = ({
  history,
  formatDateTime,
}) => {
  return (
    <div className='p-6'>
      <Title level={4}>操作历史</Title>
      {history.length > 0 ? (
        <Timeline>
          {history.map((record) => (
            <Timeline.Item key={record.id}>
              <div className='flex items-center justify-between'>
                <div>
                  <Text strong>{record.user?.name || '系统'}</Text>
                  <div className='text-sm text-gray-600'>
                    修改了 {record.field_name}
                  </div>
                  {record.change_reason && (
                    <div className='text-sm text-gray-500'>
                      原因: {record.change_reason}
                    </div>
                  )}
                </div>
                <div className='text-right'>
                  <div className='text-sm text-gray-500'>
                    {formatDateTime(record.changed_at)}
                  </div>
                  <div className='text-xs text-gray-400'>
                    {record.old_value} → {record.new_value}
                  </div>
                </div>
              </div>
            </Timeline.Item>
          ))}
        </Timeline>
      ) : (
        <div className='text-center py-8 text-gray-500'>
          <History className='w-16 h-16 mx-auto mb-4 text-gray-300' />
          <Text>暂无历史记录</Text>
        </div>
      )}
    </div>
  );
};
