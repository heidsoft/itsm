'use client';

import React from 'react';
import { Card, Space } from 'antd';

/**
 * FilterToolbarCard - Combined filter and action toolbar card
 * @param filters - ReactNode(s) for filter controls (Select, DatePicker, Input, etc.)
 * @param actions - ReactNode(s) for action buttons (primary CTA, reset, export, etc.)
 * @param className - Optional additional CSS class names
 */
interface FilterToolbarCardProps {
  filters?: React.ReactNode;
  actions?: React.ReactNode;
  className?: string;
}

export function FilterToolbarCard({ filters, actions, className }: FilterToolbarCardProps) {
  return (
    <Card className={className}>
      <div className="flex flex-col gap-3 xl:flex-row xl:items-center xl:justify-between">
        {filters && (
          <Space wrap size="middle" className="min-w-0">
            {filters}
          </Space>
        )}
        {actions && (
          <Space wrap size="middle" className="xl:justify-end">
            {actions}
          </Space>
        )}
      </div>
    </Card>
  );
}
