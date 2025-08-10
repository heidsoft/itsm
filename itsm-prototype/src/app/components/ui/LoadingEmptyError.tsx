import React from 'react';
import { Spin, Empty, Alert, Button, Space } from 'antd';
import { 
  LoadingOutlined, 
  ExclamationCircleOutlined, 
  ReloadOutlined,
  PlusOutlined,
  SearchOutlined,
  FileText,
  AlertTriangle,
  HelpCircle,
  Database,
  Workflow,
  BarChart3,
  Settings
} from 'lucide-react';

export interface LoadingEmptyErrorProps {
  /** 当前状态 */
  state: 'loading' | 'empty' | 'error' | 'success';
  /** 加载状态文本 */
  loadingText?: string;
  /** 空状态配置 */
  empty?: {
    /** 空状态图标 */
    icon?: React.ReactNode;
    /** 空状态标题 */
    title?: string;
    /** 空状态描述 */
    description?: string;
    /** 空状态操作按钮 */
    actions?: Array<{
      text: string;
      icon?: React.ReactNode;
      onClick: () => void;
      type?: 'primary' | 'default' | 'dashed' | 'link' | 'text';
    }>;
  };
  /** 错误状态配置 */
  error?: {
    /** 错误标题 */
    title?: string;
    /** 错误描述 */
    description?: string;
    /** 错误详情 */
    details?: string;
    /** 重试函数 */
    onRetry?: () => void;
    /** 自定义操作按钮 */
    actions?: Array<{
      text: string;
      icon?: React.ReactNode;
      onClick: () => void;
      type?: 'primary' | 'default' | 'dashed' | 'link' | 'text';
    }>;
  };
  /** 成功状态内容 */
  children?: React.ReactNode;
  /** 自定义样式类名 */
  className?: string;
  /** 容器高度 */
  height?: number | string;
  /** 是否显示边框 */
  bordered?: boolean;
  /** 内容区域最小高度 */
  minHeight?: number | string;
}

// 默认空状态图标映射
const getDefaultEmptyIcon = (type?: string) => {
  switch (type) {
    case 'tickets':
    case 'incidents':
    case 'problems':
    case 'changes':
      return <FileText className="w-16 h-16 text-gray-300" />;
    case 'workflow':
      return <Workflow className="w-16 h-16 text-gray-300" />;
    case 'cmdb':
      return <Database className="w-16 h-16 text-gray-300" />;
    case 'reports':
      return <BarChart3 className="w-16 h-16 text-gray-300" />;
    case 'settings':
      return <Settings className="w-16 h-16 text-gray-300" />;
    default:
      return <FileText className="w-16 h-16 text-gray-300" />;
  }
};

// 默认空状态标题映射
const getDefaultEmptyTitle = (type?: string) => {
  switch (type) {
    case 'tickets':
      return '暂无工单';
    case 'incidents':
      return '暂无事件';
    case 'problems':
      return '暂无问题';
    case 'changes':
      return '暂无变更';
    case 'workflow':
      return '暂无工作流';
    case 'cmdb':
      return '暂无配置项';
    case 'reports':
      return '暂无报表数据';
    case 'settings':
      return '暂无配置';
    default:
      return '暂无数据';
  }
};

// 默认空状态描述映射
const getDefaultEmptyDescription = (type?: string) => {
  switch (type) {
    case 'tickets':
      return '当前没有工单记录，您可以创建新的工单';
    case 'incidents':
      return '当前没有事件记录，您可以创建新的事件';
    case 'problems':
      return '当前没有问题记录，您可以创建新的问题';
    case 'changes':
      return '当前没有变更记录，您可以创建新的变更';
    case 'workflow':
      return '当前没有工作流记录，您可以创建新的工作流';
    case 'cmdb':
      return '当前没有配置项记录，您可以创建新的配置项';
    case 'reports':
      return '当前没有报表数据，请稍后再试';
    case 'settings':
      return '当前没有配置项，您可以添加新的配置';
    default:
      return '当前没有相关数据';
  }
};

export const LoadingEmptyError: React.FC<LoadingEmptyErrorProps> = ({
  state,
  loadingText = '加载中...',
  empty,
  error,
  children,
  className = '',
  height,
  bordered = false,
  minHeight = 200,
  ...props
}) => {
  const containerStyle: React.CSSProperties = {
    height,
    minHeight,
    ...(bordered && {
      border: '1px solid #f0f0f0',
      borderRadius: '8px',
      padding: '24px',
    }),
  };

  // 加载状态
  if (state === 'loading') {
    return (
      <div 
        className={`flex items-center justify-center ${className}`}
        style={containerStyle}
      >
        <div className="text-center">
          <Spin 
            indicator={<LoadingOutlined style={{ fontSize: 24 }} spin />}
            size="large"
          />
          <div className="mt-4 text-gray-600">{loadingText}</div>
        </div>
      </div>
    );
  }

  // 错误状态
  if (state === 'error') {
    return (
      <div 
        className={`flex items-center justify-center ${className}`}
        style={containerStyle}
      >
        <Alert
          message={error?.title || '加载失败'}
          description={
            <div className="mt-2">
              <div className="text-gray-600 mb-3">
                {error?.description || '数据加载过程中发生错误，请稍后重试'}
              </div>
              {error?.details && (
                <div className="text-sm text-gray-500 mb-3">
                  {error.details}
                </div>
              )}
              <Space>
                {error?.onRetry && (
                  <Button 
                    icon={<ReloadOutlined />}
                    onClick={error.onRetry}
                    type="primary"
                  >
                    重试
                  </Button>
                )}
                {error?.actions?.map((action, index) => (
                  <Button
                    key={index}
                    icon={action.icon}
                    onClick={action.onClick}
                    type={action.type || 'default'}
                  >
                    {action.text}
                  </Button>
                ))}
              </Space>
            </div>
          }
          type="error"
          showIcon
          icon={<ExclamationCircleOutlined />}
          className="max-w-md"
        />
      </div>
    );
  }

  // 空状态
  if (state === 'empty') {
    const emptyIcon = empty?.icon || getDefaultEmptyIcon();
    const emptyTitle = empty?.title || getDefaultEmptyTitle();
    const emptyDescription = empty?.description || getDefaultEmptyDescription();

    return (
      <div 
        className={`flex items-center justify-center ${className}`}
        style={containerStyle}
      >
        <Empty
          image={emptyIcon}
          description={
            <div className="text-center">
              <div className="text-lg font-medium text-gray-900 mb-2">
                {emptyTitle}
              </div>
              <div className="text-gray-600 mb-6">
                {emptyDescription}
              </div>
              {empty?.actions && empty.actions.length > 0 && (
                <Space>
                  {empty.actions.map((action, index) => (
                    <Button
                      key={index}
                      icon={action.icon}
                      onClick={action.onClick}
                      type={action.type || 'primary'}
                    >
                      {action.text}
                    </Button>
                  ))}
                </Space>
              )}
            </div>
          }
          className="max-w-md"
        />
      </div>
    );
  }

  // 成功状态 - 显示子内容
  return (
    <div className={className} style={containerStyle}>
      {children}
    </div>
  );
};

// 便捷的状态管理Hook
export const useLoadingEmptyError = <T>(
  fetcher: () => Promise<T>,
  options: {
    autoFetch?: boolean;
    onSuccess?: (data: T) => void;
    onError?: (error: Error) => void;
  } = {}
) => {
  const [state, setState] = React.useState<'loading' | 'empty' | 'error' | 'success'>('loading');
  const [data, setData] = React.useState<T | null>(null);
  const [error, setError] = React.useState<Error | null>(null);

  const fetch = React.useCallback(async () => {
    setState('loading');
    setError(null);
    
    try {
      const result = await fetcher();
      setData(result);
      
      // 判断是否为空数据
      if (Array.isArray(result)) {
        setState(result.length === 0 ? 'empty' : 'success');
      } else if (result === null || result === undefined) {
        setState('empty');
      } else {
        setState('success');
      }
      
      options.onSuccess?.(result);
    } catch (err) {
      const error = err instanceof Error ? err : new Error('未知错误');
      setError(error);
      setState('error');
      options.onError?.(error);
    }
  }, [fetcher, options]);

  React.useEffect(() => {
    if (options.autoFetch !== false) {
      fetch();
    }
  }, [fetch, options.autoFetch]);

  return {
    state,
    data,
    error,
    refetch: fetch,
    setState,
  };
};

export default LoadingEmptyError;
