'use client';

import React from 'react';
import { Card, List, Avatar, Tag, Button, Tooltip, Empty, Spin } from 'antd';
import {
  Clock,
  User,
  ArrowRight,
  FileText,
  AlertTriangle,
  BarChart3,
  HelpCircle,
} from 'lucide-react';
import { RecentActivity as RecentActivityType } from '../types/dashboard.types';

interface RecentActivityProps {
  activities: RecentActivityType[];
  loading?: boolean;
  onViewAll?: () => void;
}

// 活动项组件
const ActivityItem: React.FC<{ activity: RecentActivityType }> = React.memo(({ activity }) => {
  // 获取活动类型图标
  const getActivityIcon = () => {
    switch (activity.type) {
      case 'ticket':
        return <FileText size={16} className='text-blue-500' />;
      case 'incident':
        return <AlertTriangle size={16} className='text-red-500' />;
      case 'change':
        return <BarChart3 size={16} className='text-green-500' />;
      case 'problem':
        return <HelpCircle size={16} className='text-orange-500' />;
      default:
        return <FileText size={16} className='text-gray-500' />;
    }
  };

  // 获取优先级颜色
  const getPriorityColor = () => {
    switch (activity.priority) {
      case 'urgent':
        return 'red';
      case 'high':
        return 'orange';
      case 'medium':
        return 'blue';
      case 'low':
        return 'green';
      default:
        return 'default';
    }
  };

  // 获取状态颜色
  const getStatusColor = () => {
    const status = activity.status.toLowerCase();
    if (status.includes('resolved') || status.includes('closed')) return 'success';
    if (status.includes('progress') || status.includes('investigation')) return 'processing';
    if (status.includes('approved')) return 'success';
    return 'default';
  };

  // 格式化时间
  const formatTime = (timestamp: string) => {
    const date = new Date(timestamp);
    const now = new Date();
    const diff = now.getTime() - date.getTime();

    const minutes = Math.floor(diff / (1000 * 60));
    const hours = Math.floor(diff / (1000 * 60 * 60));
    const days = Math.floor(diff / (1000 * 60 * 60 * 24));

    if (minutes < 60) {
      return `${minutes}分钟前`;
    } else if (hours < 24) {
      return `${hours}小时前`;
    } else {
      return `${days}天前`;
    }
  };

  return (
    <List.Item className='px-0 py-3 border-b border-gray-100 last:border-b-0'>
      <div className='flex items-start gap-3 w-full'>
        {/* 活动图标 */}
        <div className='flex-shrink-0 mt-1'>
          <div className='w-8 h-8 bg-gray-50 rounded-full flex items-center justify-center'>
            {getActivityIcon()}
          </div>
        </div>

        {/* 活动内容 */}
        <div className='flex-1 min-w-0'>
          <div className='flex items-start justify-between mb-1'>
            <div className='flex-1 min-w-0'>
              <h4 className='text-sm font-medium text-gray-900 truncate'>{activity.title}</h4>
              <p className='text-xs text-gray-600 mt-1 line-clamp-2'>{activity.description}</p>
            </div>

            {/* 优先级标签 */}
            {activity.priority && (
              <Tag color={getPriorityColor()} className='ml-2 flex-shrink-0'>
                {activity.priority.toUpperCase()}
              </Tag>
            )}
          </div>

          <div className='flex items-center justify-between mt-2'>
            <div className='flex items-center gap-3 text-xs text-gray-500'>
              <div className='flex items-center gap-1'>
                <User size={12} />
                <span>{activity.user}</span>
              </div>
              <div className='flex items-center gap-1'>
                <Clock size={12} />
                <span>{formatTime(activity.timestamp)}</span>
              </div>
            </div>

            <Tag color={getStatusColor()}>{activity.status}</Tag>
          </div>
        </div>
      </div>
    </List.Item>
  );
});

ActivityItem.displayName = 'ActivityItem';

// 最近活动主组件
export const RecentActivity: React.FC<RecentActivityProps> = React.memo(
  ({ activities, loading = false, onViewAll }) => {
    if (loading) {
      return (
        <Card
          title={
            <div className='flex items-center gap-2'>
              <Clock size={18} className='text-blue-500' />
              <span>最近活动</span>
            </div>
          }
          className='h-full'
          extra={
            <Button type='link' size='small' disabled>
              查看全部
            </Button>
          }
        >
          <div className='flex items-center justify-center h-64'>
            <Spin size='large' />
          </div>
        </Card>
      );
    }

    if (!activities || activities.length === 0) {
      return (
        <Card
          title={
            <div className='flex items-center gap-2'>
              <Clock size={18} className='text-blue-500' />
              <span>最近活动</span>
            </div>
          }
          className='h-full'
        >
          <div className='flex items-center justify-center h-64'>
            <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='暂无最近活动' />
          </div>
        </Card>
      );
    }

    return (
      <Card
        title={
          <div className='flex items-center gap-2'>
            <Clock size={18} className='text-blue-500' />
            <span>Recent Activity</span>
          </div>
        }
        className='h-full'
        extra={
          <Tooltip title='View all activities'>
            <Button
              type='link'
              size='small'
              onClick={onViewAll}
              className='flex items-center gap-1'
            >
              查看全部
              <ArrowRight size={12} />
            </Button>
          </Tooltip>
        }
      >
        <List
          dataSource={activities.slice(0, 5)}
          renderItem={activity => <ActivityItem activity={activity} />}
          className='activity-list'
        />

        {activities.length > 5 && (
          <div className='text-center mt-4 pt-4 border-t border-gray-100'>
            <Button type='link' size='small' onClick={onViewAll} className='text-blue-500'>
              显示 {activities.length - 5} 个更多活动
            </Button>
          </div>
        )}
      </Card>
    );
  }
);

RecentActivity.displayName = 'RecentActivity';
