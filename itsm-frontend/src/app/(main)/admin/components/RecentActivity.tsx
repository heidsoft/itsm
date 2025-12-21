'use client';

import React from 'react';
import { Card, List, Space, Typography, Avatar, Button, theme } from 'antd';
import {
  Activity,
  Users,
  Workflow,
  Shield,
  BookOpen,
  Bell,
  Clock,
} from 'lucide-react';
import { useI18n } from '@/lib/i18n';

const { Text } = Typography;

const recentActivities = [
  {
    id: 1,
    type: 'user_created',
    title: '新用户注册',
    description: '张三 加入了系统',
    time: '2分钟前',
    icon: Users,
    color: 'bg-blue-100 text-blue-600',
  },
  {
    id: 2,
    type: 'workflow_updated',
    title: '工作流更新',
    description: '事件管理流程已更新',
    time: '1小时前',
    icon: Workflow,
    color: 'bg-green-100 text-green-600',
  },
  {
    id: 3,
    type: 'role_assigned',
    title: '角色分配',
    description: '为李四分配了管理员角色',
    time: '3小时前',
    icon: Shield,
    color: 'bg-purple-100 text-purple-600',
  },
  {
    id: 4,
    type: 'service_added',
    title: '服务目录更新',
    description: '新增云存储服务项',
    time: '5小时前',
    icon: BookOpen,
    color: 'bg-orange-100 text-orange-600',
  },
  {
    id: 5,
    type: 'notification_sent',
    title: '通知发送',
    description: '系统维护通知已发送',
    time: '1天前',
    icon: Bell,
    color: 'bg-yellow-100 text-yellow-600',
  },
];

export const RecentActivity: React.FC = () => {
  const { token } = theme.useToken();
  const { t } = useI18n();

  return (
    <Card
      title={
        <Space>
          <Activity className='w-5 h-5' />
          {t('admin.recentActivity')}
        </Space>
      }
      extra={
        <Button type='link' size='small'>
          {t('admin.viewAll')}
        </Button>
      }
      style={{ height: '100%' }}
    >
      <List
        dataSource={recentActivities}
        renderItem={activity => {
          const Icon = activity.icon;
          const getActivityColor = (colorClass: string) => {
            if (colorClass.includes('blue')) return token.colorPrimary;
            if (colorClass.includes('green')) return token.colorSuccess;
            if (colorClass.includes('purple')) return '#722ed1';
            if (colorClass.includes('orange')) return '#f97316';
            if (colorClass.includes('yellow')) return token.colorWarning;
            return token.colorPrimary;
          };

          return (
            <List.Item
              style={{
                padding: `${token.paddingSM}px 0`,
                borderBottom: `1px solid ${token.colorBorder}`,
              }}
            >
              <List.Item.Meta
                avatar={
                  <Avatar
                    style={{
                      backgroundColor: getActivityColor(activity.color),
                    }}
                    icon={<Icon className='w-4 h-4' />}
                  />
                }
                title={
                  <div
                    style={{
                      display: 'flex',
                      justifyContent: 'space-between',
                      alignItems: 'center',
                    }}
                  >
                    <Text strong>{activity.title}</Text>
                    <Space
                      align='center'
                      style={{
                        color: token.colorTextSecondary,
                        fontSize: token.fontSizeSM,
                      }}
                    >
                      <Clock className='w-3 h-3' />
                      {activity.time}
                    </Space>
                  </div>
                }
                description={activity.description}
              />
            </List.Item>
          );
        }}
      />
    </Card>
  );
};
