'use client';

import React from 'react';
import { Button, Result, Spin, Typography } from 'antd';
import { RotateCcw, Plus, FileText, AlertTriangle, User, Database, Settings } from 'lucide-react';
import { useI18n } from '@/lib/i18n/useI18n';

const { Text, Title } = Typography;

export type LoadingEmptyErrorState = 'loading' | 'empty' | 'error' | 'success';

interface LoadingEmptyErrorProps {
  state: LoadingEmptyErrorState;
  loadingText?: string;
  empty?: {
    title?: string;
    description?: string;
    actionText?: string;
    onAction?: () => void;
    icon?: React.ReactNode;
    showAction?: boolean;
  };
  error?: {
    title?: string;
    description?: string;
    actionText?: string;
    onAction?: () => void;
    showRetry?: boolean;
    showAction?: boolean;
  };
  success?: {
    title?: string;
    description?: string;
    actionText?: string;
    onAction?: () => void;
    showAction?: boolean;
  };
  minHeight?: number;
  className?: string;
  children?: React.ReactNode;
}

// 预定义的空状态配置
const getDefaultEmptyConfig = (context?: string, t?: (key: string) => string) => {
  const defaultTitle = t ? t('common.noData') : '暂无数据';
  const defaultDesc = t ? t('common.noData') : '暂无数据';
  const defaultAction = t ? t('common.create') : '创建';

  switch (context) {
    case 'tickets':
      return {
        title: defaultTitle,
        description: t ? t('common.noData') : '当前没有工单数据，点击下方按钮创建第一个工单',
        actionText: t ? t('common.create') : '创建工单',
        icon: <FileText size={48} />,
      };
    case 'incidents':
      return {
        title: defaultTitle,
        description: t ? t('common.noData') : '当前没有事件数据，点击下方按钮创建第一个事件',
        actionText: t ? t('common.create') : '创建事件',
        icon: <AlertTriangle size={48} />,
      };
    case 'problems':
      return {
        title: defaultTitle,
        description: t ? t('common.noData') : '当前没有问题数据，点击下方按钮创建第一个问题',
        actionText: t ? t('common.create') : '创建问题',
        icon: <AlertTriangle size={48} />,
      };
    case 'changes':
      return {
        title: defaultTitle,
        description: t ? t('common.noData') : '当前没有变更数据，点击下方按钮创建第一个变更',
        actionText: t ? t('common.create') : '创建变更',
        icon: <Settings size={48} />,
      };
    case 'cmdb':
      return {
        title: defaultTitle,
        description: t ? t('common.noData') : '当前没有配置项数据，点击下方按钮创建第一个配置项',
        actionText: t ? t('common.create') : '创建配置项',
        icon: <Database size={48} />,
      };
    case 'users':
      return {
        title: defaultTitle,
        description: t ? t('common.noData') : '当前没有用户数据，点击下方按钮创建第一个用户',
        actionText: t ? t('common.create') : '创建用户',
        icon: <User size={48} />,
      };
    case 'workflows':
      return {
        title: defaultTitle,
        description: t ? t('common.noData') : '当前没有工作流数据，点击下方按钮创建第一个工作流',
        actionText: t ? t('common.create') : '创建工作流',
        icon: <Settings size={48} />,
      };
    default:
      return {
        title: defaultTitle,
        description: defaultDesc,
        actionText: defaultAction,
        icon: <FileText size={48} />,
      };
  }
};

export const LoadingEmptyError: React.FC<LoadingEmptyErrorProps> = ({
  state,
  loadingText,
  empty,
  error,
  success,
  minHeight = 200,
  className = '',
  children,
}) => {
  const { t } = useI18n();
  const defaultLoadingText = t('common.loading');
  // 成功状态直接显示内容
  if (state === 'success') {
    return (
      <div className={className}>
        {success?.title && (
          <div className="mb-4 text-center">
            <Title level={4} className="text-gray-700">
              {success.title}
            </Title>
            {success.description && (
              <Text type="secondary" className="text-sm">
                {success.description}
              </Text>
            )}
            {success.showAction && success.actionText && success.onAction && (
              <div className="mt-3">
                <Button type="primary" onClick={success.onAction}>
                  {success.actionText}
                </Button>
              </div>
            )}
          </div>
        )}
        {children}
      </div>
    );
  }

  // 加载状态
  if (state === 'loading') {
    return (
      <div
        className={`flex flex-col items-center justify-center ${className}`}
        style={{ minHeight }}
      >
        <Spin size="large" />
        <Text className="mt-4 text-gray-500">{loadingText || defaultLoadingText}</Text>
      </div>
    );
  }

  // 空状态
  if (state === 'empty') {
    const defaultConfig = getDefaultEmptyConfig(undefined, t);
    const config = {
      ...defaultConfig,
      ...empty,
      icon: empty?.icon || defaultConfig.icon,
    };

    return (
      <div
        className={`flex flex-col items-center justify-center ${className}`}
        style={{ minHeight }}
      >
        <div className="text-gray-300 mb-4">{config.icon}</div>
        <Title level={4} className="text-gray-600 mb-2">
          {config.title}
        </Title>
        <Text type="secondary" className="text-center mb-6 max-w-md">
          {config.description}
        </Text>
        {config.showAction !== false && config.actionText && config.onAction && (
          <Button type="primary" icon={<Plus size={16} />} onClick={config.onAction}>
            {config.actionText}
          </Button>
        )}
      </div>
    );
  }

  // 错误状态
  if (state === 'error') {
    const defaultErrorConfig = {
      title: t('common.loadFailed'),
      description: t('common.checkNetwork'),
      actionText: t('common.retry'),
      showRetry: true,
      showAction: false,
    };
    const config = { ...defaultErrorConfig, ...error };

    return (
      <div
        className={`flex flex-col items-center justify-center ${className}`}
        style={{ minHeight }}
      >
        <Result
          status="error"
          icon={<AlertTriangle style={{ color: '#ff4d4f' }} />}
          title={config.title}
          subTitle={config.description}
          extra={
            <div className="flex gap-2">
              {config.showRetry && (
                <Button type="primary" icon={<RotateCcw size={16} />} onClick={config.onAction}>
                  {config.actionText}
                </Button>
              )}
              {config.showAction && config.actionText && config.onAction && (
                <Button onClick={config.onAction}>{config.actionText}</Button>
              )}
            </div>
          }
        />
      </div>
    );
  }

  return null;
};

// 预定义的上下文组件
export const TicketsLoadingEmptyError = (props: Omit<LoadingEmptyErrorProps, 'empty'>) => (
  <LoadingEmptyError {...props} empty={getDefaultEmptyConfig('tickets')} />
);

export const IncidentsLoadingEmptyError = (props: Omit<LoadingEmptyErrorProps, 'empty'>) => (
  <LoadingEmptyError {...props} empty={getDefaultEmptyConfig('incidents')} />
);

export const ProblemsLoadingEmptyError = (props: Omit<LoadingEmptyErrorProps, 'empty'>) => (
  <LoadingEmptyError {...props} empty={getDefaultEmptyConfig('problems')} />
);

export const ChangesLoadingEmptyError = (props: Omit<LoadingEmptyErrorProps, 'empty'>) => (
  <LoadingEmptyError {...props} empty={getDefaultEmptyConfig('changes')} />
);

export const CMDBLoadingEmptyError = (props: Omit<LoadingEmptyErrorProps, 'empty'>) => (
  <LoadingEmptyError {...props} empty={getDefaultEmptyConfig('cmdb')} />
);

export const UsersLoadingEmptyError = (props: Omit<LoadingEmptyErrorProps, 'empty'>) => (
  <LoadingEmptyError {...props} empty={getDefaultEmptyConfig('users')} />
);

export const WorkflowsLoadingEmptyError = (props: Omit<LoadingEmptyErrorProps, 'empty'>) => (
  <LoadingEmptyError {...props} empty={getDefaultEmptyConfig('workflows')} />
);

// 简化的状态组件
export const Loading = ({ text, className = '' }: { text?: string; className?: string }) => (
  <LoadingEmptyError state="loading" loadingText={text} className={className} />
);

export const Empty = ({
  title,
  description,
  actionText,
  onAction,
  className = '',
}: {
  title?: string;
  description?: string;
  actionText?: string;
  onAction?: () => void;
  className?: string;
}) => (
  <LoadingEmptyError
    state="empty"
    empty={{ title, description, actionText, onAction }}
    className={className}
  />
);

export const Error = ({
  title,
  description,
  actionText,
  onAction,
  className = '',
}: {
  title?: string;
  description?: string;
  actionText?: string;
  onAction?: () => void;
  className?: string;
}) => (
  <LoadingEmptyError
    state="error"
    error={{ title, description, actionText, onAction }}
    className={className}
  />
);

export default LoadingEmptyError;
