/**
 * ITSM前端性能优化 - 骨架屏和加载状态
 *
 * 提供各种骨架屏和加载状态组件，提升用户体验
 */

import React, { useState, useEffect } from 'react';
import { Skeleton, Spin, Card, Button, Progress, Alert } from 'antd';
import {
  LoadingOutlined,
  ReloadOutlined,
  CheckCircleOutlined,
  ExclamationCircleOutlined,
} from '@lucide-react';

// ==================== 基础骨架屏组件 ====================

/**
 * 文本骨架屏
 */
export const TextSkeleton: React.FC<{
  lines?: number;
  width?: string | string[];
  active?: boolean;
}> = ({ lines = 3, width = '100%', active = true }) => (
  <Skeleton active={active} paragraph={{ rows: lines, width }} title={false} />
);

/**
 * 标题骨架屏
 */
export const TitleSkeleton: React.FC<{
  width?: string;
  active?: boolean;
}> = ({ width = '60%', active = true }) => (
  <Skeleton active={active} title={{ width }} paragraph={false} />
);

/**
 * 头像骨架屏
 */
export const AvatarSkeleton: React.FC<{
  size?: number;
  active?: boolean;
}> = ({ size = 40, active = true }) => (
  <Skeleton active={active} avatar={{ size }} title={false} paragraph={false} />
);

/**
 * 卡片骨架屏
 */
export const CardSkeleton: React.FC<{
  showAvatar?: boolean;
  lines?: number;
  active?: boolean;
}> = ({ showAvatar = true, lines = 2, active = true }) => (
  <Card>
    <Skeleton
      active={active}
      avatar={showAvatar}
      paragraph={{ rows: lines }}
      title={{ width: '70%' }}
    />
  </Card>
);

// ==================== 表格骨架屏组件 ====================

/**
 * 表格骨架屏
 */
export const TableSkeleton: React.FC<{
  columns?: number;
  rows?: number;
  showHeader?: boolean;
  active?: boolean;
}> = ({ columns = 5, rows = 5, showHeader = true, active = true }) => (
  <div>
    {showHeader && (
      <div
        style={{
          display: 'flex',
          padding: '12px 16px',
          backgroundColor: '#fafafa',
          borderBottom: '1px solid #d9d9d9',
          marginBottom: '8px',
        }}
      >
        {Array.from({ length: columns }).map((_, index) => (
          <Skeleton
            key={index}
            active={active}
            title={false}
            paragraph={false}
            style={{
              width: `${100 / columns}%`,
              marginRight: index < columns - 1 ? '16px' : 0,
            }}
          />
        ))}
      </div>
    )}
    {Array.from({ length: rows }).map((_, rowIndex) => (
      <div
        key={rowIndex}
        style={{
          display: 'flex',
          padding: '8px 16px',
          borderBottom: '1px solid #f0f0f0',
        }}
      >
        {Array.from({ length: columns }).map((_, colIndex) => (
          <Skeleton
            key={colIndex}
            active={active}
            title={false}
            paragraph={false}
            style={{
              width: `${100 / columns}%`,
              marginRight: colIndex < columns - 1 ? '16px' : 0,
            }}
          />
        ))}
      </div>
    ))}
  </div>
);

/**
 * 列表骨架屏
 */
export const ListSkeleton: React.FC<{
  items?: number;
  showAvatar?: boolean;
  showActions?: boolean;
  active?: boolean;
}> = ({ items = 5, showAvatar = true, showActions = true, active = true }) => (
  <div>
    {Array.from({ length: items }).map((_, index) => (
      <div
        key={index}
        style={{
          display: 'flex',
          alignItems: 'center',
          padding: '12px 0',
          borderBottom: index < items - 1 ? '1px solid #f0f0f0' : 'none',
        }}
      >
        {showAvatar && (
          <Skeleton
            active={active}
            avatar={{ size: 40 }}
            title={false}
            paragraph={false}
            style={{ marginRight: '12px' }}
          />
        )}
        <div style={{ flex: 1 }}>
          <Skeleton
            active={active}
            title={{ width: '60%' }}
            paragraph={{ rows: 1, width: '80%' }}
          />
        </div>
        {showActions && (
          <Skeleton
            active={active}
            title={false}
            paragraph={false}
            style={{ width: '80px', height: '32px' }}
          />
        )}
      </div>
    ))}
  </div>
);

// ==================== 表单骨架屏组件 ====================

/**
 * 表单骨架屏
 */
export const FormSkeleton: React.FC<{
  fields?: number;
  showSubmit?: boolean;
  active?: boolean;
}> = ({ fields = 4, showSubmit = true, active = true }) => (
  <div>
    {Array.from({ length: fields }).map((_, index) => (
      <div key={index} style={{ marginBottom: '24px' }}>
        <Skeleton
          active={active}
          title={{ width: '30%' }}
          paragraph={false}
          style={{ marginBottom: '8px' }}
        />
        <Skeleton active={active} title={false} paragraph={false} style={{ height: '32px' }} />
      </div>
    ))}
    {showSubmit && (
      <div style={{ marginTop: '32px' }}>
        <Skeleton
          active={active}
          title={false}
          paragraph={false}
          style={{ width: '120px', height: '40px' }}
        />
      </div>
    )}
  </div>
);

// ==================== 加载状态组件 ====================

/**
 * 页面加载状态
 */
export const PageLoading: React.FC<{
  message?: string;
  size?: 'small' | 'default' | 'large';
}> = ({ message = '加载中...', size = 'large' }) => (
  <div
    style={{
      display: 'flex',
      flexDirection: 'column',
      justifyContent: 'center',
      alignItems: 'center',
      height: '400px',
      gap: '16px',
    }}
  >
    <Spin
      size={size}
      indicator={<LoadingOutlined style={{ fontSize: size === 'large' ? 48 : 32 }} spin />}
    />
    <div style={{ color: '#666', fontSize: '16px' }}>{message}</div>
  </div>
);

/**
 * 内联加载状态
 */
export const InlineLoading: React.FC<{
  message?: string;
  size?: 'small' | 'default' | 'large';
}> = ({ message, size = 'default' }) => (
  <div
    style={{
      display: 'flex',
      alignItems: 'center',
      gap: '8px',
      padding: '8px 0',
    }}
  >
    <Spin
      size={size}
      indicator={<LoadingOutlined style={{ fontSize: size === 'large' ? 24 : 16 }} spin />}
    />
    {message && <span style={{ color: '#666' }}>{message}</span>}
  </div>
);

/**
 * 按钮加载状态
 */
export const ButtonLoading: React.FC<{
  loading?: boolean;
  children: React.ReactNode;
  onClick?: () => void;
  disabled?: boolean;
}> = ({ loading = false, children, onClick, disabled }) => (
  <Button
    loading={loading}
    onClick={onClick}
    disabled={disabled || loading}
    icon={loading ? <LoadingOutlined /> : undefined}
  >
    {children}
  </Button>
);

// ==================== 进度加载组件 ====================

/**
 * 进度加载器
 */
export const ProgressLoader: React.FC<{
  percent?: number;
  status?: 'active' | 'success' | 'exception';
  message?: string;
  showInfo?: boolean;
}> = ({ percent = 0, status = 'active', message, showInfo = true }) => (
  <div style={{ padding: '20px' }}>
    <Progress
      percent={percent}
      status={status}
      showInfo={showInfo}
      strokeColor={status === 'success' ? '#52c41a' : undefined}
    />
    {message && (
      <div
        style={{
          textAlign: 'center',
          marginTop: '8px',
          color: '#666',
        }}
      >
        {message}
      </div>
    )}
  </div>
);

/**
 * 步骤加载器
 */
export const StepLoader: React.FC<{
  current?: number;
  steps: string[];
  status?: 'wait' | 'process' | 'finish' | 'error';
}> = ({ current = 0, steps, status = 'process' }) => (
  <div style={{ padding: '20px' }}>
    {steps.map((step, index) => (
      <div
        key={index}
        style={{
          display: 'flex',
          alignItems: 'center',
          marginBottom: '16px',
        }}
      >
        <div
          style={{
            width: '24px',
            height: '24px',
            borderRadius: '50%',
            backgroundColor:
              index < current ? '#52c41a' : index === current ? '#1890ff' : '#d9d9d9',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            marginRight: '12px',
            color: 'white',
            fontSize: '12px',
            fontWeight: 'bold',
          }}
        >
          {index < current ? <CheckCircleOutlined /> : index + 1}
        </div>
        <span
          style={{
            color: index <= current ? '#000' : '#999',
            fontWeight: index === current ? 'bold' : 'normal',
          }}
        >
          {step}
        </span>
        {index === current && status === 'error' && (
          <ExclamationCircleOutlined style={{ marginLeft: '8px', color: '#ff4d4f' }} />
        )}
      </div>
    ))}
  </div>
);

// ==================== 错误状态组件 ====================

/**
 * 错误状态
 */
export const ErrorState: React.FC<{
  title?: string;
  message?: string;
  onRetry?: () => void;
  showRetry?: boolean;
}> = ({
  title = '加载失败',
  message = '数据加载失败，请检查网络连接后重试',
  onRetry,
  showRetry = true,
}) => (
  <div
    style={{
      display: 'flex',
      flexDirection: 'column',
      justifyContent: 'center',
      alignItems: 'center',
      height: '300px',
      gap: '16px',
      padding: '20px',
    }}
  >
    <ExclamationCircleOutlined style={{ fontSize: '48px', color: '#ff4d4f' }} />
    <div style={{ textAlign: 'center' }}>
      <h3 style={{ margin: '0 0 8px 0', color: '#000' }}>{title}</h3>
      <p style={{ margin: 0, color: '#666' }}>{message}</p>
    </div>
    {showRetry && onRetry && (
      <Button type='primary' icon={<ReloadOutlined />} onClick={onRetry}>
        重试
      </Button>
    )}
  </div>
);

/**
 * 空状态
 */
export const EmptyState: React.FC<{
  title?: string;
  message?: string;
  action?: React.ReactNode;
}> = ({ title = '暂无数据', message = '当前没有数据，请稍后再试', action }) => (
  <div
    style={{
      display: 'flex',
      flexDirection: 'column',
      justifyContent: 'center',
      alignItems: 'center',
      height: '300px',
      gap: '16px',
      padding: '20px',
    }}
  >
    <div
      style={{
        width: '64px',
        height: '64px',
        borderRadius: '50%',
        backgroundColor: '#f5f5f5',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        fontSize: '24px',
        color: '#999',
      }}
    >
      📄
    </div>
    <div style={{ textAlign: 'center' }}>
      <h3 style={{ margin: '0 0 8px 0', color: '#000' }}>{title}</h3>
      <p style={{ margin: 0, color: '#666' }}>{message}</p>
    </div>
    {action}
  </div>
);

// ==================== 智能加载组件 ====================

/**
 * 智能加载状态管理器
 */
export class LoadingStateManager {
  private states = new Map<
    string,
    {
      loading: boolean;
      error: string | null;
      data: any;
      timestamp: number;
    }
  >();

  /**
   * 设置加载状态
   */
  setLoading(key: string, loading: boolean): void {
    const current = this.states.get(key) || {
      loading: false,
      error: null,
      data: null,
      timestamp: 0,
    };
    this.states.set(key, { ...current, loading, timestamp: Date.now() });
  }

  /**
   * 设置错误状态
   */
  setError(key: string, error: string): void {
    const current = this.states.get(key) || {
      loading: false,
      error: null,
      data: null,
      timestamp: 0,
    };
    this.states.set(key, { ...current, error, loading: false, timestamp: Date.now() });
  }

  /**
   * 设置数据
   */
  setData(key: string, data: any): void {
    const current = this.states.get(key) || {
      loading: false,
      error: null,
      data: null,
      timestamp: 0,
    };
    this.states.set(key, { ...current, data, loading: false, error: null, timestamp: Date.now() });
  }

  /**
   * 获取状态
   */
  getState(
    key: string
  ): { loading: boolean; error: string | null; data: any; timestamp: number } | null {
    return this.states.get(key) || null;
  }

  /**
   * 清除状态
   */
  clearState(key: string): void {
    this.states.delete(key);
  }

  /**
   * 清除所有状态
   */
  clearAllStates(): void {
    this.states.clear();
  }
}

export const loadingStateManager = new LoadingStateManager();

/**
 * 智能加载Hook
 */
export function useSmartLoading(key: string) {
  const [state, setState] = useState(
    () =>
      loadingStateManager.getState(key) || { loading: false, error: null, data: null, timestamp: 0 }
  );

  const setLoading = (loading: boolean) => {
    loadingStateManager.setLoading(key, loading);
    setState(loadingStateManager.getState(key)!);
  };

  const setError = (error: string) => {
    loadingStateManager.setError(key, error);
    setState(loadingStateManager.getState(key)!);
  };

  const setData = (data: any) => {
    loadingStateManager.setData(key, data);
    setState(loadingStateManager.getState(key)!);
  };

  const clearState = () => {
    loadingStateManager.clearState(key);
    setState({ loading: false, error: null, data: null, timestamp: 0 });
  };

  return {
    ...state,
    setLoading,
    setError,
    setData,
    clearState,
  };
}

// ==================== 加载状态包装器 ====================

/**
 * 加载状态包装器
 */
export const LoadingWrapper: React.FC<{
  loading: boolean;
  error?: string | null;
  data?: any;
  children: React.ReactNode;
  loadingComponent?: React.ReactNode;
  errorComponent?: React.ReactNode;
  emptyComponent?: React.ReactNode;
  onRetry?: () => void;
}> = ({
  loading,
  error,
  data,
  children,
  loadingComponent,
  errorComponent,
  emptyComponent,
  onRetry,
}) => {
  if (loading) {
    return <>{loadingComponent || <PageLoading />}</>;
  }

  if (error) {
    return <>{errorComponent || <ErrorState message={error} onRetry={onRetry} />}</>;
  }

  if (!data || (Array.isArray(data) && data.length === 0)) {
    return <>{emptyComponent || <EmptyState />}</>;
  }

  return <>{children}</>;
};

// ==================== 性能优化加载组件 ====================

/**
 * 延迟加载组件
 */
export const DelayedLoading: React.FC<{
  delay?: number;
  children: React.ReactNode;
  fallback?: React.ReactNode;
}> = ({ delay = 200, children, fallback }) => {
  const [show, setShow] = useState(false);

  useEffect(() => {
    const timer = setTimeout(() => {
      setShow(true);
    }, delay);

    return () => clearTimeout(timer);
  }, [delay]);

  if (!show) {
    return <>{fallback || <InlineLoading />}</>;
  }

  return <>{children}</>;
};

/**
 * 渐进式加载组件
 */
export const ProgressiveLoading: React.FC<{
  steps: Array<{
    delay: number;
    component: React.ReactNode;
  }>;
}> = ({ steps }) => {
  const [currentStep, setCurrentStep] = useState(0);

  useEffect(() => {
    if (currentStep < steps.length) {
      const timer = setTimeout(() => {
        setCurrentStep(prev => prev + 1);
      }, steps[currentStep].delay);

      return () => clearTimeout(timer);
    }
  }, [currentStep, steps]);

  return (
    <div>
      {steps.slice(0, currentStep + 1).map((step, index) => (
        <div key={index}>{step.component}</div>
      ))}
    </div>
  );
};

export default {
  TextSkeleton,
  TitleSkeleton,
  AvatarSkeleton,
  CardSkeleton,
  TableSkeleton,
  ListSkeleton,
  FormSkeleton,
  PageLoading,
  InlineLoading,
  ButtonLoading,
  ProgressLoader,
  StepLoader,
  ErrorState,
  EmptyState,
  LoadingStateManager,
  loadingStateManager,
  useSmartLoading,
  LoadingWrapper,
  DelayedLoading,
  ProgressiveLoading,
};
