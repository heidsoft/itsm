"use client";

import React, { useMemo, useCallback } from 'react';
import { Spin, Empty, Alert, Button, Card, Space, Typography } from 'antd';
import { RefreshCw, Wifi, WifiOff, AlertCircle, Inbox } from 'lucide-react';

const { Text, Title } = Typography;

interface LoadingStateManagerProps {
  loading?: boolean;
  error?: string | null;
  empty?: boolean;
  data?: unknown;
  children: React.ReactNode;
  
  // 自定义配置
  loadingText?: string;
  emptyText?: string;
  emptyDescription?: string;
  errorTitle?: string;
  showRetry?: boolean;
  onRetry?: () => void;
  
  // 样式配置
  minHeight?: number | string;
  className?: string;
  cardWrapper?: boolean;
  
  // 高级配置
  retryCount?: number;
  maxRetries?: number;
  showErrorDetails?: boolean;
}

const LoadingStateManagerComponent: React.FC<LoadingStateManagerProps> = ({
  loading = false,
  error = null,
  empty = false,
  data,
  children,
  loadingText = '加载中...',
  emptyText = '暂无数据',
  emptyDescription = '当前没有可显示的内容',
  errorTitle = '加载失败',
  showRetry = true,
  onRetry,
  minHeight = 200,
  className = '',
  cardWrapper = false,
  retryCount = 0,
  maxRetries = 3,
  showErrorDetails = false,
}) => {
  // 使用useMemo优化计算
  const isEmpty = useMemo(() => {
    return empty || (
      data !== undefined && (
        data === null ||
        (Array.isArray(data) && data.length === 0) ||
        (typeof data === 'object' && Object.keys(data).length === 0)
      )
    );
  }, [empty, data]);

  // 使用useMemo优化网络错误判断
  const isNetworkError = useMemo(() => {
    return error && (
      error.includes('网络') ||
      error.includes('连接') ||
      error.includes('timeout') ||
      error.includes('fetch')
    );
  }, [error]);

  // 使用useMemo优化容器样式
  const containerStyle = useMemo(() => ({
    minHeight: typeof minHeight === 'number' ? `${minHeight}px` : minHeight,
  }), [minHeight]);

  // 使用useCallback优化重试处理
  const handleRetry = useCallback(() => {
    if (onRetry) {
      onRetry();
    }
  }, [onRetry]);

  // 使用useCallback优化页面刷新
  const handleRefresh = useCallback(() => {
    window.location.reload();
  }, []);

  // 渲染加载状态
  const renderLoading = () => (
    <div 
      className={`flex flex-col items-center justify-center ${className}`}
      style={containerStyle}
    >
      <Spin size="large" />
      <Text className="mt-4 text-gray-500">{loadingText}</Text>
    </div>
  );

  // 渲染错误状态
  const renderError = () => {
    const isMaxRetriesReached = retryCount >= maxRetries;
    
    return (
      <div 
        className={`flex flex-col items-center justify-center p-6 ${className}`}
        style={containerStyle}
      >
        <div className="text-center max-w-md">
          {isNetworkError ? (
            <WifiOff className="text-red-500 mx-auto mb-4" size={48} />
          ) : (
            <AlertCircle className="text-red-500 mx-auto mb-4" size={48} />
          )}
          
          <Title level={4} className="text-gray-800 mb-2">
            {errorTitle}
          </Title>
          
          <Text className="text-gray-600 block mb-4">
            {isNetworkError ? '网络连接异常，请检查网络设置' : error}
          </Text>
          
          {showErrorDetails && retryCount > 0 && (
            <Text className="text-orange-600 text-sm block mb-4">
              已重试 {retryCount} 次
              {isMaxRetriesReached && ' (已达到最大重试次数)'}
            </Text>
          )}
          
          {showRetry && onRetry && !isMaxRetriesReached && (
            <Space>
              <Button 
                type="primary" 
                icon={<RefreshCw size={16} />}
                onClick={handleRetry}
              >
                重试
              </Button>
              {isNetworkError && (
                <Button 
                  icon={<Wifi size={16} />}
                  onClick={handleRefresh}
                >
                  刷新页面
                </Button>
              )}
            </Space>
          )}
          
          {isMaxRetriesReached && (
            <Alert
              message="多次重试失败"
              description="请稍后再试或联系技术支持"
              type="warning"
              showIcon
              className="mt-4 text-left"
            />
          )}
        </div>
      </div>
    );
  };

  // 渲染空状态
  const renderEmpty = () => (
    <div 
      className={`flex flex-col items-center justify-center ${className}`}
      style={containerStyle}
    >
      <Empty
        image={<Inbox className="text-gray-400" size={64} />}
        description={
          <Space direction="vertical" size="small">
            <Text className="text-gray-600">{emptyText}</Text>
            <Text className="text-gray-400 text-sm">{emptyDescription}</Text>
          </Space>
        }
      />
    </div>
  );

  // 渲染内容
  const renderContent = () => {
    if (loading) {
      return renderLoading();
    }
    
    if (error) {
      return renderError();
    }
    
    if (isEmpty) {
      return renderEmpty();
    }
    
    return children;
  };

  // 如果需要卡片包装
  if (cardWrapper) {
    return (
      <Card className={className}>
        {renderContent()}
      </Card>
    );
  }

  return <>{renderContent()}</>;
};

// 高阶组件包装器
export const withLoadingState = <P extends object>(
  Component: React.ComponentType<P>
) => {
  const WrappedComponent = (props: P & LoadingStateManagerProps) => {
    const { 
      loading, 
      error, 
      empty, 
      data, 
      loadingText,
      emptyText,
      emptyDescription,
      errorTitle,
      showRetry,
      onRetry,
      minHeight,
      className,
      cardWrapper,
      retryCount,
      maxRetries,
      showErrorDetails,
      ...componentProps 
    } = props;
    
    return (
      <LoadingStateManagerComponent
        loading={loading}
        error={error}
        empty={empty}
        data={data}
        loadingText={loadingText}
        emptyText={emptyText}
        emptyDescription={emptyDescription}
        errorTitle={errorTitle}
        showRetry={showRetry}
        onRetry={onRetry}
        minHeight={minHeight}
        className={className}
        cardWrapper={cardWrapper}
        retryCount={retryCount}
        maxRetries={maxRetries}
        showErrorDetails={showErrorDetails}
      >
        <Component {...(componentProps as P)} />
      </LoadingStateManagerComponent>
    );
  };
  
  WrappedComponent.displayName = `withLoadingState(${Component.displayName || Component.name})`;
  
  return WrappedComponent;
};

// 简化的加载组件
const SimpleLoadingComponent: React.FC<{
  text?: string;
  size?: 'small' | 'default' | 'large';
}> = ({ text = '加载中...', size = 'default' }) => (
  <div className="flex flex-col items-center justify-center p-8">
    <Spin size={size} />
    <Text className="mt-2 text-gray-500">{text}</Text>
  </div>
);

export const SimpleLoading = React.memo(SimpleLoadingComponent);

// 简化的错误组件
const SimpleErrorComponent: React.FC<{
  message?: string;
  onRetry?: () => void;
}> = ({ message = '加载失败', onRetry }) => (
  <div className="flex flex-col items-center justify-center p-8">
    <AlertCircle className="text-red-500 mb-2" size={32} />
    <Text className="text-gray-600 mb-4">{message}</Text>
    {onRetry && (
      <Button 
        type="primary" 
        size="small"
        icon={<RefreshCw size={14} />}
        onClick={onRetry}
      >
        重试
      </Button>
    )}
  </div>
);

export const SimpleError = React.memo(SimpleErrorComponent);

// 使用React.memo优化性能
export const LoadingStateManager = React.memo(LoadingStateManagerComponent, (prevProps, nextProps) => {
  // 自定义比较函数，避免不必要的重渲染
  return (
    prevProps.loading === nextProps.loading &&
    prevProps.error === nextProps.error &&
    prevProps.empty === nextProps.empty &&
    prevProps.data === nextProps.data &&
    prevProps.loadingText === nextProps.loadingText &&
    prevProps.emptyText === nextProps.emptyText &&
    prevProps.emptyDescription === nextProps.emptyDescription &&
    prevProps.errorTitle === nextProps.errorTitle &&
    prevProps.showRetry === nextProps.showRetry &&
    prevProps.onRetry === nextProps.onRetry &&
    prevProps.minHeight === nextProps.minHeight &&
    prevProps.className === nextProps.className &&
    prevProps.cardWrapper === nextProps.cardWrapper &&
    prevProps.retryCount === nextProps.retryCount &&
    prevProps.maxRetries === nextProps.maxRetries &&
    prevProps.showErrorDetails === nextProps.showErrorDetails
  );
});

export default LoadingStateManager;