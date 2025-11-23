'use client';

import React from 'react';
import { Button } from 'antd';
import { ArrowLeft } from 'lucide-react';
import { useRouter } from 'next/navigation';

interface PageHeaderProps {
  title?: string;
  description?: string;
  extra?: React.ReactNode;
  showBackButton?: boolean;
}

/**
 * 页面头部组件
 * 用于在页面顶部显示标题、描述和操作按钮
 * 不包含布局元素（Header/Sidebar），仅作为内容装饰
 */
export function PageHeader({ title, description, extra, showBackButton = false }: PageHeaderProps) {
  const router = useRouter();

  if (!title && !description && !extra && !showBackButton) {
    return null;
  }

  return (
    <div
      style={{
        marginBottom: 24,
        paddingBottom: 16,
        borderBottom: '1px solid #e5e7eb',
      }}
    >
      {/* 返回按钮 */}
      {showBackButton && (
        <div style={{ marginBottom: 16 }}>
          <Button icon={<ArrowLeft size={16} />} onClick={() => router.back()} size='small'>
            返回
          </Button>
        </div>
      )}

      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
        {/* 标题和描述 */}
        <div style={{ flex: 1 }}>
          {title && (
            <h1
              style={{
                fontSize: 20,
                fontWeight: '600',
                color: '#1f2937',
                margin: 0,
                marginBottom: description ? 8 : 0,
              }}
            >
              {title}
            </h1>
          )}
          {description && (
            <p
              style={{
                fontSize: 14,
                color: '#6b7280',
                margin: 0,
                lineHeight: '1.5',
              }}
            >
              {description}
            </p>
          )}
        </div>

        {/* 操作按钮区域 */}
        {extra && (
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              marginLeft: 16,
            }}
          >
            {extra}
          </div>
        )}
      </div>
    </div>
  );
}

export default PageHeader;
