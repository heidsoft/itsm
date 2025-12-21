'use client';

import React from 'react';
import { Card, Row, Col, Button, Empty, Spin } from 'antd';
import {
  PlusOutlined,
  ArrowRightOutlined,
  FileTextOutlined,
  WarningOutlined,
  FundOutlined,
  RiseOutlined,
  ThunderboltOutlined,
} from '@ant-design/icons';
import { useRouter } from 'next/navigation';
import { QuickAction } from '../types/dashboard.types';

interface QuickActionsProps {
  actions: QuickAction[];
  loading?: boolean;
  onActionClick?: (action: QuickAction) => void;
  showTitle?: boolean;
  compact?: boolean;
}

// 企业级快速操作卡片
const EnterpriseQuickActionCard: React.FC<{
  action: QuickAction;
  onClick?: (action: QuickAction) => void;
}> = React.memo(({ action, onClick }) => {
  const router = useRouter();

  const handleClick = () => {
    if (onClick) {
      onClick(action);
    } else {
      router.push(action.path);
    }
  };

  const getDefaultIcon = () => {
    const iconStyle = { fontSize: 28 };
    switch (action.id) {
      case 'create-ticket':
        return <FileTextOutlined style={iconStyle} />;
      case 'create-incident':
        return <WarningOutlined style={iconStyle} />;
      case 'create-change':
        return <FundOutlined style={iconStyle} />;
      case 'view-reports':
        return <RiseOutlined style={iconStyle} />;
      default:
        return <PlusOutlined style={iconStyle} />;
    }
  };

  return (
    <Col xs={24} sm={12} md={6} lg={6}>
      <Card
        className='enterprise-action-card h-full cursor-pointer border-0 hover:shadow-2xl transition-all duration-300 group overflow-hidden'
        style={{
          borderRadius: '12px',
          background: '#ffffff',
          boxShadow: '0 1px 3px rgba(0, 0, 0, 0.12), 0 1px 2px rgba(0, 0, 0, 0.08)',
        }}
        styles={{
          body: {
            padding: '24px',
            minHeight: '200px',
            display: 'flex',
            flexDirection: 'column',
            position: 'relative',
          },
        }}
        onClick={handleClick}
      >
        {/* 背景装饰 */}
        <div
          className='absolute inset-0 opacity-0 group-hover:opacity-100 transition-opacity duration-500'
          style={{
            background: `radial-gradient(circle at top right, ${action.color}20 0%, transparent 70%)`,
          }}
        />

        <div className='relative z-10 flex flex-col h-full'>
          {/* 图标区域 */}
          <div className='mb-4'>
            <div
              className='w-14 h-14 rounded-2xl flex items-center justify-center shadow-lg transition-all duration-300 group-hover:scale-110 group-hover:rotate-6'
              style={{
                background: `linear-gradient(135deg, ${action.color} 0%, ${action.color}dd 100%)`,
                boxShadow: `0 4px 16px ${action.color}40`,
              }}
            >
              <div style={{ color: '#ffffff' }}>{action.icon || getDefaultIcon()}</div>
            </div>
          </div>

          {/* 内容区域 */}
          <div className='flex-1 flex flex-col'>
            <h3 className='text-base font-bold text-gray-900 mb-2 group-hover:text-gray-700 transition-colors'>
              {action.title}
            </h3>
            <p className='text-sm text-gray-600 mb-4 line-clamp-2 flex-1'>{action.description}</p>
          </div>

          {/* 操作按钮 */}
          <div className='mt-auto pt-4 border-t border-gray-100'>
            <Button
              type='text'
              size='middle'
              className='w-full font-semibold group/btn'
              style={{
                color: action.color,
                borderColor: action.color,
              }}
              icon={<ArrowRightOutlined />}
              iconPosition='end'
            >
              <span className='group-hover/btn:mr-1 transition-all duration-300'>立即开始</span>
            </Button>
          </div>
        </div>

        {/* 装饰光晕 */}
        <div
          className='absolute -bottom-12 -right-12 w-32 h-32 rounded-full opacity-10 group-hover:opacity-20 transition-opacity duration-500 blur-3xl'
          style={{
            background: action.color,
          }}
        />
      </Card>
    </Col>
  );
});

EnterpriseQuickActionCard.displayName = 'EnterpriseQuickActionCard';

// 快速操作主组件
export const QuickActions: React.FC<QuickActionsProps> = React.memo(
  ({ actions, loading = false, onActionClick, showTitle = true, compact = false }) => {
    if (loading) {
      return (
        <div className={compact ? '' : 'mb-6'}>
          {showTitle && !compact && (
            <div className='mb-4'>
              <h2 className='text-lg font-semibold text-gray-900 mb-1 flex items-center gap-2'>
                <ThunderboltOutlined style={{ fontSize: 20, color: '#3b82f6' }} />
                快速操作
              </h2>
              <p className='text-sm text-gray-600'>常用任务和快捷方式</p>
            </div>
          )}

          <Row gutter={[16, 16]}>
            {Array.from({ length: 4 }).map((_, index) => (
              <Col key={index} xs={24} sm={12} md={6}>
                <Card
                  className='h-48 border-0'
                  style={{
                    borderRadius: '12px',
                    boxShadow: '0 1px 3px rgba(0, 0, 0, 0.12)',
                  }}
                >
                  <div className='flex items-center justify-center h-full'>
                    <Spin size='large' />
                  </div>
                </Card>
              </Col>
            ))}
          </Row>
        </div>
      );
    }

    if (!actions || actions.length === 0) {
      return (
        <div className={compact ? '' : 'mb-6'}>
          {showTitle && !compact && (
            <div className='mb-4'>
              <h2 className='text-lg font-semibold text-gray-900 mb-1'>快速操作</h2>
              <p className='text-sm text-gray-600'>常用任务和快捷方式</p>
            </div>
          )}

          <Card
            className='text-center py-12 border-0'
            style={{
              borderRadius: '12px',
              boxShadow: '0 1px 3px rgba(0, 0, 0, 0.12)',
            }}
          >
            <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='暂无快速操作' />
          </Card>
        </div>
      );
    }

    return (
      <div className={compact ? '' : 'mb-6'}>
        {showTitle && !compact && (
          <div className='mb-4'>
            <h2 className='text-lg font-semibold text-gray-900 mb-1 flex items-center gap-2'>
              <ThunderboltOutlined style={{ fontSize: 20, color: '#3b82f6' }} />
              快速操作
            </h2>
            <p className='text-sm text-gray-600'>常用任务和快捷方式</p>
          </div>
        )}

        <Row gutter={[16, 16]}>
          {actions.map(action => (
            <EnterpriseQuickActionCard key={action.id} action={action} onClick={onActionClick} />
          ))}
        </Row>
      </div>
    );
  }
);

QuickActions.displayName = 'QuickActions';
