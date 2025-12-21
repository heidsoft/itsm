'use client';

import React from 'react';
import { Card, Space, Typography, List, Tag, Avatar, theme, Button } from 'antd';
import {
  Activity,
  AlertCircle,
  CheckCircle,
  RefreshCw,
  XCircle,
} from 'lucide-react';
import { useI18n } from '@/lib/i18n';

const { Title, Text } = Typography;

const systemHealth = {
  overall: 'excellent', // excellent, good, warning, critical
  uptime: '99.98%',
  lastUpdate: '2024-01-15 14:30:25',
  services: {
    database: 'healthy',
    api: 'healthy',
    cache: 'healthy',
    queue: 'warning',
  },
};

export const SystemHealth: React.FC = () => {
  const { token } = theme.useToken();
  const { t } = useI18n();

  const getHealthStatus = (status: string) => {
    switch (status) {
      case 'excellent':
        return {
          type: 'success' as const,
          text: t('admin.excellent'),
          color: token.colorSuccess,
        };
      case 'good':
        return { type: 'info' as const, text: t('admin.good'), color: token.colorInfo };
      case 'warning':
        return {
          type: 'warning' as const,
          text: t('admin.warning'),
          color: token.colorWarning,
        };
      case 'critical':
        return {
          type: 'error' as const,
          text: t('admin.critical'),
          color: token.colorError,
        };
      default:
        return {
          type: 'default' as const,
          text: t('admin.unknown'),
          color: token.colorTextSecondary,
        };
    }
  };

  const getHealthIcon = (status: string) => {
    switch (status) {
      case 'excellent':
      case 'good':
        return CheckCircle;
      case 'warning':
        return AlertCircle;
      case 'critical':
        return XCircle;
      default:
        return Activity;
    }
  };

  const HealthIcon = getHealthIcon(systemHealth.overall);
  const healthStatus = getHealthStatus(systemHealth.overall);

  const serviceList = Object.entries(systemHealth.services).map(
    ([service, status]) => ({
      name:
        service === 'database'
          ? t('admin.database')
          : service === 'api'
          ? t('admin.apiService')
          : service === 'cache'
          ? t('admin.cache')
          : t('admin.messageQueue'),
      status,
      icon: getHealthIcon(status),
    })
  );

  return (
    <Card
      title={
        <Space>
          <Activity className='w-5 h-5' />
          {t('admin.systemHealth')}
        </Space>
      }
      extra={
        <Button
          type='text'
          icon={<RefreshCw className='w-4 h-4' />}
          size='small'
        />
      }
    >
      <div style={{ marginBottom: token.marginLG }}>
        <Space align='center' size='large'>
          <Avatar
            size={48}
            style={{ backgroundColor: healthStatus.color, border: 'none' }}
            icon={<HealthIcon className='w-6 h-6' />}
          />
          <div>
            <Title level={3} style={{ margin: 0, color: healthStatus.color }}>
              {healthStatus.text}
            </Title>
            <Text type='secondary'>{t('admin.uptime')}: {systemHealth.uptime}</Text>
          </div>
        </Space>
      </div>

      <List
        grid={{ column: 2, gutter: 16 }}
        dataSource={serviceList}
        renderItem={item => {
          const ServiceIcon = item.icon;
          const serviceStatus = getHealthStatus(item.status);
          return (
            <List.Item>
              <Space align='center'>
                <ServiceIcon
                  className='w-4 h-4'
                  style={{ color: serviceStatus.color }}
                />
                <Text>{item.name}</Text>
                <Tag color={serviceStatus.type}>{serviceStatus.text}</Tag>
              </Space>
            </List.Item>
          );
        }}
      />

      <div
        style={{
          marginTop: token.marginLG,
          paddingTop: token.paddingSM,
          borderTop: `1px solid ${token.colorBorder}`,
        }}
      >
        <Text type='secondary' style={{ fontSize: token.fontSizeSM }}>
          {t('admin.lastUpdate')}: {systemHealth.lastUpdate}
        </Text>
      </div>
    </Card>
  );
};
