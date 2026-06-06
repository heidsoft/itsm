'use client';

import React from 'react';
import { Alert, Space, Typography } from 'antd';

const { Title, Text } = Typography;

interface ManagementPageHeaderProps {
  title: string;
  description?: string;
  actions?: React.ReactNode;
  notice?: React.ReactNode;
  className?: string;
}

export function ManagementPageHeader({
  title,
  description,
  actions,
  notice,
  className,
}: ManagementPageHeaderProps) {
  return (
    <div className={className}>
      <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
        <div className="min-w-0">
          <Title level={2} className="!mb-1">
            {title}
          </Title>
          {description && <Text type="secondary">{description}</Text>}
        </div>
        {actions && (
          <Space wrap className="lg:justify-end">
            {actions}
          </Space>
        )}
      </div>
      {notice && <div className="mt-4">{notice}</div>}
    </div>
  );
}

interface ManagementNoticeProps {
  message: React.ReactNode;
  description?: React.ReactNode;
  type?: 'info' | 'warning' | 'error' | 'success';
}

export function ManagementNotice({
  message,
  description,
  type = 'info',
}: ManagementNoticeProps) {
  return <Alert showIcon type={type} message={message} description={description} />;
}

