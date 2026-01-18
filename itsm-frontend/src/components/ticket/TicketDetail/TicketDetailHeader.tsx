'use client';

import React from 'react';
import {
  User,
  Calendar,
  Clock,
  AlertTriangle,
  Edit,
  FileText,
} from 'lucide-react';
import {
  Typography,
  Badge,
  Tag as AntTag,
} from 'antd';
import { Ticket } from '@/app/lib/api-config';

const { Title, Text } = Typography;

// 优先级配置
export const getPriorityConfig = (priority: string) => {
  const configs: Record<string, { color: string; bgColor: string; textColor: string; borderColor: string }> = {
    紧急: { color: 'red', bgColor: 'bg-red-100', textColor: 'text-red-800', borderColor: 'border-red-300' },
    高: { color: 'orange', bgColor: 'bg-orange-100', textColor: 'text-orange-800', borderColor: 'border-orange-300' },
    中: { color: 'yellow', bgColor: 'bg-yellow-100', textColor: 'text-yellow-800', borderColor: 'border-yellow-300' },
    低: { color: 'blue', bgColor: 'bg-blue-100', textColor: 'text-blue-800', borderColor: 'border-blue-300' },
  };
  return configs[priority] || configs['中'];
};

// 状态配置
export const getStatusConfig = (status: string) => {
  const configs: Record<string, { bgColor: string; textColor: string }> = {
    处理中: { bgColor: 'bg-blue-100', textColor: 'text-blue-800' },
    已分配: { bgColor: 'bg-purple-100', textColor: 'text-purple-800' },
    已解决: { bgColor: 'bg-green-100', textColor: 'text-green-800' },
    已关闭: { bgColor: 'bg-gray-200', textColor: 'text-gray-800' },
    待审批: { bgColor: 'bg-yellow-100', textColor: 'text-yellow-800' },
    已批准: { bgColor: 'bg-green-100', textColor: 'text-green-800' },
  };
  return configs[status] || { bgColor: 'bg-gray-100', textColor: 'text-gray-800' };
};

// 获取类型图标
export const getTypeIcon = (type: string) => {
  const icons: Record<string, React.ComponentType<any>> = {
    Incident: AlertTriangle,
    Problem: AlertTriangle,
    Change: Edit,
    'Service Request': FileText,
  };
  return icons[type] || FileText;
};

// 格式化时间
const formatDateTime = (dateString: string) => {
  return new Date(dateString).toLocaleString('zh-CN');
};

interface TicketDetailHeaderProps {
  ticket: Ticket;
  isEditing: boolean;
  onEditToggle: () => void;
  canEdit: boolean;
}

export const TicketDetailHeader: React.FC<TicketDetailHeaderProps> = ({
  ticket,
  isEditing,
  onEditToggle,
  canEdit,
}) => {
  const TypeIcon = getTypeIcon(ticket.type || 'Service Request');
  const priorityConfig = getPriorityConfig(ticket.priority || '中');
  const statusConfig = getStatusConfig(ticket.status || '待处理');

  return (
    <div className='sticky top-0 z-10 bg-white shadow-sm border-b mb-6'>
      <div className='px-6 py-4'>
        <div className='flex items-start justify-between mb-4'>
          <div className='flex items-center space-x-3'>
            <TypeIcon className='w-8 h-8 text-blue-600' />
            <div>
              <Title level={2} className='mb-1'>
                {ticket.title}
              </Title>
              <Text type='secondary'>
                {ticket.type || 'Service Request'} #{ticket.ticket_number || ticket.id}
              </Text>
            </div>
          </div>
          {canEdit && (
            <button
              onClick={onEditToggle}
              className={`px-4 py-2 rounded-md text-sm font-medium transition-colors ${
                isEditing
                  ? 'bg-gray-100 text-gray-700 hover:bg-gray-200'
                  : 'bg-blue-50 text-blue-700 hover:bg-blue-100'
              }`}
            >
              {isEditing ? '取消编辑' : '编辑'}
            </button>
          )}
        </div>

        {/* 关键信息：状态、优先级、处理人 */}
        <div className='flex items-center flex-wrap gap-4 mb-4'>
          <Badge
            status={
              ticket.status === 'open'
                ? 'processing'
                : ticket.status === 'closed'
                ? 'success'
                : 'warning'
            }
            text={<Text strong>{ticket.status || '待处理'}</Text>}
          />
          <AntTag color={priorityConfig.color} style={{ fontSize: '14px', padding: '4px 12px' }}>
            优先级: {ticket.priority || '中'}
          </AntTag>
          <div className='flex items-center space-x-2'>
            <User className='w-4 h-4 text-gray-500' />
            <Text>
              <Text type='secondary' className='text-sm'>
                处理人：
              </Text>
              <Text strong>{ticket.assignee?.name || '未分配'}</Text>
            </Text>
          </div>
          {ticket.category && <AntTag color='blue'>{ticket.category}</AntTag>}
          {ticket.tags?.slice(0, 3).map((tag: string, index: number) => (
            <AntTag key={index} color='green'>
              {tag}
            </AntTag>
          ))}
          {ticket.tags && ticket.tags.length > 3 && (
            <AntTag>+{ticket.tags.length - 3}</AntTag>
          )}
        </div>

        {/* 次要信息：快速查看 */}
        <div className='grid grid-cols-2 md:grid-cols-4 gap-4 pt-3 border-t border-gray-100'>
          <div>
            <Text type='secondary' className='text-xs'>
              报告人
            </Text>
            <div className='flex items-center space-x-1 mt-1'>
              <User className='w-3 h-3 text-gray-400' />
              <Text className='text-sm'>{ticket.requester?.name || '未知'}</Text>
            </div>
          </div>
          <div>
            <Text type='secondary' className='text-xs'>
              创建时间
            </Text>
            <div className='flex items-center space-x-1 mt-1'>
              <Calendar className='w-3 h-3 text-gray-400' />
              <Text className='text-sm'>{formatDateTime(ticket.created_at)}</Text>
            </div>
          </div>
          <div>
            <Text type='secondary' className='text-xs'>
              最后更新
            </Text>
            <div className='flex items-center space-x-1 mt-1'>
              <Clock className='w-3 h-3 text-gray-400' />
              <Text className='text-sm'>{formatDateTime(ticket.updated_at)}</Text>
            </div>
          </div>
          <div>
            <Text type='secondary' className='text-xs'>
              工单类型
            </Text>
            <div className='mt-1'>
              <Text className='text-sm'>{ticket.type || 'Service Request'}</Text>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
