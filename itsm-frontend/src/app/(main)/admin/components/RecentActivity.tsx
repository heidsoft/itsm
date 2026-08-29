'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { Card, List, Space, Typography, Avatar, Button, Tag, theme, Empty, Skeleton } from 'antd';
import {
  Activity,
  Clock,
  Ticket,
  AlertCircle,
  RefreshCw,
  Wrench,
} from 'lucide-react';
import { useI18n } from '@/lib/i18n';
import { DashboardAPI } from '@/lib/api/dashboard-api';
import type { RecentActivity as RecentActivityData } from '@/app/(main)/dashboard/types/dashboard.types';

const { Text } = Typography;

// 将 ISO 时间格式化为相对时间（例如：2分钟前）
const formatRelativeTime = (iso: string): string => {
  const ts = new Date(iso).getTime();
  if (Number.isNaN(ts)) return iso;
  const diffSec = Math.floor((Date.now() - ts) / 1000);
  if (diffSec < 60) return '刚刚';
  const diffMin = Math.floor(diffSec / 60);
  if (diffMin < 60) return `${diffMin}分钟前`;
  const diffHour = Math.floor(diffMin / 60);
  if (diffHour < 24) return `${diffHour}小时前`;
  const diffDay = Math.floor(diffHour / 24);
  if (diffDay < 30) return `${diffDay}天前`;
  return new Date(ts).toLocaleDateString('zh-CN');
};

// 活动类型 → 图标与配色
const ACTIVITY_META: Record<
  string,
  { icon: React.ComponentType<{ className?: string; size?: number }>; color: string; tokenColor: string }
> = {
  ticket: { icon: Ticket, color: 'bg-blue-100 text-blue-600', tokenColor: 'colorPrimary' },
  incident: { icon: AlertCircle, color: 'bg-red-100 text-red-600', tokenColor: 'colorError' },
  change: { icon: RefreshCw, color: 'bg-green-100 text-green-600', tokenColor: 'colorSuccess' },
  problem: { icon: Wrench, color: 'bg-purple-100 text-purple-600', tokenColor: '#722ed1' },
  default: { icon: Activity, color: 'bg-gray-100 text-gray-600', tokenColor: 'colorTextSecondary' },
};

export const RecentActivity: React.FC = () => {
  const { token } = theme.useToken();
  const { t } = useI18n();
  const [loading, setLoading] = useState(true);
  const [activities, setActivities] = useState<RecentActivityData[]>([]);

  const loadActivities = useCallback(async () => {
    setLoading(true);
    try {
      const data = await DashboardAPI.getRecentActivities(8);
      setActivities(Array.isArray(data) ? data : []);
    } catch (error) {
      console.error('Failed to load recent activities:', error);
      setActivities([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadActivities();
  }, [loadActivities]);

  return (
    <Card
      title={
        <Space>
          <Activity className="w-5 h-5" />
          {t('admin.recentActivity')}
          <Tag color="blue">实时</Tag>
        </Space>
      }
      extra={
        <Button type="link" size="small" onClick={() => void loadActivities()} loading={loading}>
          {t('admin.refresh')}
        </Button>
      }
      style={{ height: '100%' }}
    >
      {loading ? (
        <Skeleton active paragraph={{ rows: 5 }} />
      ) : activities.length === 0 ? (
        <Empty
          image={Empty.PRESENTED_IMAGE_SIMPLE}
          description={t('admin.noRecentActivity')}
          style={{ padding: '24px 0' }}
        />
      ) : (
        <List
          dataSource={activities}
          renderItem={activity => {
            const meta = ACTIVITY_META[activity.type] || ACTIVITY_META.default;
            const Icon = meta.icon;
            const color =
              meta.tokenColor === 'colorPrimary'
                ? token.colorPrimary
                : meta.tokenColor === 'colorSuccess'
                  ? token.colorSuccess
                  : meta.tokenColor === 'colorError'
                    ? token.colorError
                    : meta.tokenColor === 'colorTextSecondary'
                      ? token.colorTextSecondary
                      : meta.tokenColor;

            return (
              <List.Item
                style={{
                  padding: `${token.paddingSM}px 0`,
                  borderBottom: `1px solid ${token.colorBorder}`,
                }}
              >
                <List.Item.Meta
                  avatar={
                    <Avatar style={{ backgroundColor: color }}>
                      <Icon className="w-4 h-4" />
                    </Avatar>
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
                        align="center"
                        style={{ color: token.colorTextSecondary, fontSize: token.fontSizeSM }}
                      >
                        <Clock className="w-3 h-3" />
                        {formatRelativeTime(activity.timestamp)}
                      </Space>
                    </div>
                  }
                  description={
                    <span>
                      {activity.description}
                      {activity.user ? ` · ${activity.user}` : ''}
                      {activity.status ? (
                        <Tag
                          color="default"
                          style={{ marginLeft: 8, fontSize: 12 }}
                        >
                          {activity.status}
                        </Tag>
                      ) : null}
                    </span>
                  }
                />
              </List.Item>
            );
          }}
        />
      )}
    </Card>
  );
};
