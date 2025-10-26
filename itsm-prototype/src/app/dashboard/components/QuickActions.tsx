'use client';

import React from 'react';
import { Card, Row, Col, Button, Tooltip, Empty, Spin } from 'antd';
import { Plus, ArrowRight, FileText, AlertTriangle, BarChart3, TrendingUp } from 'lucide-react';
import { useRouter } from 'next/navigation';
import { QuickAction } from '../types/dashboard.types';

interface QuickActionsProps {
  actions: QuickAction[];
  loading?: boolean;
  onActionClick?: (action: QuickAction) => void;
}

// 快速操作卡片组件
const QuickActionCard: React.FC<{
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

  // 获取默认图标
  const getDefaultIcon = () => {
    switch (action.id) {
      case 'create-ticket':
        return <FileText size={24} className='text-blue-500' />;
      case 'create-incident':
        return <AlertTriangle size={24} className='text-red-500' />;
      case 'create-change':
        return <BarChart3 size={24} className='text-green-500' />;
      case 'view-reports':
        return <TrendingUp size={24} className='text-purple-500' />;
      default:
        return <Plus size={24} className='text-gray-500' />;
    }
  };

  return (
    <Col xs={24} sm={12} lg={6}>
      <Card
        className='h-full hover:shadow-lg transition-all duration-300 cursor-pointer border-0'
        style={{
          background: `linear-gradient(135deg, ${action.color}15 0%, ${action.color}05 100%)`,
          borderLeft: `4px solid ${action.color}`,
        }}
        bodyStyle={{ padding: '20px' }}
        onClick={handleClick}
      >
        <div className='text-center'>
          {/* 图标 */}
          <div className='mb-4'>
            <div
              className='w-16 h-16 mx-auto rounded-full flex items-center justify-center'
              style={{ backgroundColor: `${action.color}20` }}
            >
              {typeof action.icon === 'string' ? (
                <span className='text-2xl'>{action.icon}</span>
              ) : (
                action.icon || getDefaultIcon()
              )}
            </div>
          </div>

          {/* 标题和描述 */}
          <h3 className='text-lg font-semibold text-gray-900 mb-2'>{action.title}</h3>
          <p className='text-sm text-gray-600 mb-4 line-clamp-2'>{action.description}</p>

          {/* 操作按钮 */}
          <Button
            type='primary'
            size='small'
            className='w-full'
            style={{ backgroundColor: action.color, borderColor: action.color }}
            icon={<ArrowRight size={14} />}
          >
            开始使用
          </Button>
        </div>
      </Card>
    </Col>
  );
});

QuickActionCard.displayName = 'QuickActionCard';

// 快速操作主组件
export const QuickActions: React.FC<QuickActionsProps> = React.memo(
  ({ actions, loading = false, onActionClick }) => {
    if (loading) {
      return (
        <div className='mb-6'>
          <div className='mb-4'>
            <h2 className='text-lg font-semibold text-gray-900 mb-1'>快速操作</h2>
            <p className='text-sm text-gray-600'>常用任务和快捷方式</p>
          </div>

          <Row gutter={[16, 16]}>
            {Array.from({ length: 4 }).map((_, index) => (
              <Col key={index} xs={24} sm={12} lg={6}>
                <Card className='h-48'>
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
        <div className='mb-6'>
          <div className='mb-4'>
            <h2 className='text-lg font-semibold text-gray-900 mb-1'>快速操作</h2>
            <p className='text-sm text-gray-600'>常用任务和快捷方式</p>
          </div>

          <Card className='text-center py-12'>
            <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='暂无快速操作' />
          </Card>
        </div>
      );
    }

    return (
      <div className='mb-6'>
        <div className='mb-4'>
          <h2 className='text-lg font-semibold text-gray-900 mb-1'>快速操作</h2>
          <p className='text-sm text-gray-600'>常用任务和快捷方式</p>
        </div>

        <Row gutter={[16, 16]}>
          {actions.map(action => (
            <QuickActionCard key={action.id} action={action} onClick={onActionClick} />
          ))}
        </Row>
      </div>
    );
  }
);

QuickActions.displayName = 'QuickActions';
