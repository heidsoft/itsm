'use client';

import React, { useState, useEffect } from 'react';
import { Calendar, Badge, Card, Typography, Tag, Modal, List, App } from 'antd';
import { Clock, User, AlertCircle, CheckCircle } from 'lucide-react';
import type { Dayjs } from 'dayjs';
import dayjs from 'dayjs';
import { ChangeApi } from '@/lib/api/change-api';
import type { Change } from '@/types/biz/change';
import { ChangeStatus, ChangeStatusLabels } from '@/constants/change';

const { Title, Text } = Typography;

const statusColors: Record<string, string> = {
  [ChangeStatus.DRAFT]: 'default',
  [ChangeStatus.PENDING]: 'orange',
  [ChangeStatus.APPROVED]: 'cyan',
  [ChangeStatus.IN_PROGRESS]: 'blue',
  [ChangeStatus.COMPLETED]: 'green',
  [ChangeStatus.REJECTED]: 'red',
  [ChangeStatus.ROLLED_BACK]: 'magenta',
  [ChangeStatus.CANCELLED]: 'default',
};

interface ChangeCalendarProps {
  onDateSelect?: (date: Dayjs) => void;
}

export const ChangeCalendar: React.FC<ChangeCalendarProps> = ({ onDateSelect }) => {
  const { message } = App.useApp();
  const [changes, setChanges] = useState<Change[]>([]);
  const [loading, setLoading] = useState(false);
  const [selectedDate, setSelectedDate] = useState<Dayjs>(dayjs());
  const [modalVisible, setModalVisible] = useState(false);
  const [selectedChanges, setSelectedChanges] = useState<Change[]>([]);

  useEffect(() => {
    loadChanges();
  }, []);

  const loadChanges = async () => {
    try {
      setLoading(true);
      const response = await ChangeApi.getChanges({ page: 1, page_size: 100 });
      setChanges((response?.changes ?? []) as Change[]);
    } catch (error) {
      message.error('加载变更数据失败');
    } finally {
      setLoading(false);
    }
  };

  // 获取某一天的变更列表
  const getChangesForDate = (date: Dayjs): Change[] => {
    return changes.filter(change => {
      const startDate = change.plannedStartDate ? dayjs(change.plannedStartDate) : null;
      const endDate = change.plannedEndDate ? dayjs(change.plannedEndDate) : null;

      if (!startDate && !endDate) return false;

      // 变更开始或结束在这一天
      if (startDate && startDate.isSame(date, 'day')) return true;
      if (endDate && endDate.isSame(date, 'day')) return true;

      // 变更跨越了这一天
      if (startDate && endDate && date.isAfter(startDate, 'day') && date.isBefore(endDate, 'day')) {
        return true;
      }

      return false;
    });
  };

  // 渲染日历单元格
  const dateCellRender = (value: Dayjs) => {
    const dateChanges = getChangesForDate(value);
    if (dateChanges.length === 0) return null;

    return (
      <ul className="list-none p-0 m-0">
        {dateChanges.slice(0, 2).map((change, index) => (
          <li key={change.id || index} className="mb-1">
            <Badge
              color={statusColors[change.status] || 'default'}
              text={
                <span className="text-xs truncate block" style={{ maxWidth: '80px' }}>
                  {change.title}
                </span>
              }
            />
          </li>
        ))}
        {dateChanges.length > 2 && (
          <li>
            <Text type="secondary" className="text-xs">
              +{dateChanges.length - 2} 更多
            </Text>
          </li>
        )}
      </ul>
    );
  };

  // 处理日期选择
  const handleDateSelect = (date: Dayjs) => {
    setSelectedDate(date);
    const dateChanges = getChangesForDate(date);
    if (dateChanges.length > 0) {
      setSelectedChanges(dateChanges);
      setModalVisible(true);
    }
    onDateSelect?.(date);
  };

  return (
    <div>
      <Card className="shadow-sm">
        <Calendar
          dateCellRender={dateCellRender}
          onSelect={handleDateSelect}
          value={selectedDate}
        />
      </Card>

      <Modal
        title={
          <div className="flex items-center">
            <Clock className="w-5 h-5 mr-2 text-blue-500" />
            <span>{selectedDate.format('YYYY年MM月DD日')} 的变更</span>
          </div>
        }
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        footer={null}
        width={600}
      >
        <List
          dataSource={selectedChanges}
          renderItem={change => (
            <List.Item key={change.id}>
              <List.Item.Meta
                avatar={
                  <div
                    className="w-10 h-10 rounded-full flex items-center justify-center"
                    style={{ backgroundColor: `${statusColors[change.status]}20` }}
                  >
                    {change.status === ChangeStatus.COMPLETED ? (
                      <CheckCircle
                        className="w-5 h-5"
                        style={{ color: statusColors[change.status] }}
                      />
                    ) : (
                      <AlertCircle
                        className="w-5 h-5"
                        style={{ color: statusColors[change.status] }}
                      />
                    )}
                  </div>
                }
                title={
                  <div className="flex items-center justify-between">
                    <Text strong>{change.title}</Text>
                    <Tag color={statusColors[change.status]}>
                      {ChangeStatusLabels[change.status] || change.status}
                    </Tag>
                  </div>
                }
                description={
                  <div className="space-y-1">
                    <Text type="secondary" className="text-sm">
                      {change.description}
                    </Text>
                    <div className="flex items-center text-xs text-gray-500">
                      <Clock className="w-3 h-3 mr-1" />
                      {change.plannedStartDate && dayjs(change.plannedStartDate).format('HH:mm')}
                      {change.plannedEndDate &&
                        ` - ${dayjs(change.plannedEndDate).format('HH:mm')}`}
                    </div>
                  </div>
                }
              />
            </List.Item>
          )}
        />
      </Modal>
    </div>
  );
};

export default ChangeCalendar;
