'use client';

import React, { useState } from 'react';
import { Modal, Popconfirm, PopconfirmProps, message } from 'antd';
import {
  ExclamationCircleOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  InfoCircleOutlined,
} from '@ant-design/icons';

/**
 * 增强的确认对话框配置
 */
export interface EnhancedConfirmOptions {
  title: string;
  content?: React.ReactNode;
  type?: 'info' | 'success' | 'warning' | 'error';
  okText?: string;
  cancelText?: string;
  okType?: 'primary' | 'danger' | 'default';
  onOk?: () => void | Promise<void>;
  onCancel?: () => void;
  showIcon?: boolean;
  width?: number;
}

/**
 * 增强的确认对话框
 * 
 * 提供更友好的确认交互
 */
export const showEnhancedConfirm = (options: EnhancedConfirmOptions) => {
  const {
    title,
    content,
    type = 'warning',
    okText = '确定',
    cancelText = '取消',
    okType = type === 'error' ? 'danger' : 'primary',
    onOk,
    onCancel,
    showIcon = true,
    width = 416,
  } = options;

  const iconMap = {
    info: <InfoCircleOutlined style={{ color: '#1890ff' }} />,
    success: <CheckCircleOutlined style={{ color: '#52c41a' }} />,
    warning: <ExclamationCircleOutlined style={{ color: '#faad14' }} />,
    error: <CloseCircleOutlined style={{ color: '#ff4d4f' }} />,
  };

  Modal.confirm({
    title,
    content,
    icon: showIcon ? iconMap[type] : null,
    okText,
    cancelText,
    okType,
    onOk,
    onCancel,
    width,
    centered: true,
    maskClosable: false,
    className: 'enhanced-confirm-modal',
  });
};

/**
 * 乐观更新Hook
 * 
 * 用于实现乐观UI更新模式
 */
export interface OptimisticUpdateOptions<T> {
  /**
   * 当前数据
   */
  data: T;
  
  /**
   * 更新数据的函数
   */
  setData: (data: T) => void;
  
  /**
   * 执行更新的异步函数
   */
  updateFn: (newData: T) => Promise<void>;
  
  /**
   * 成功消息
   */
  successMessage?: string;
  
  /**
   * 失败消息
   */
  errorMessage?: string;
}

export const useOptimisticUpdate = <T,>() => {
  const [isUpdating, setIsUpdating] = useState(false);

  const performUpdate = async ({
    data,
    setData,
    updateFn,
    successMessage = '更新成功',
    errorMessage = '更新失败，请重试',
  }: OptimisticUpdateOptions<T>) => {
    const originalData = data;
    setIsUpdating(true);

    try {
      await updateFn(data);
      message.success(successMessage);
    } catch (error) {
      // 回滚到原始数据
      setData(originalData);
      message.error(errorMessage);
      console.error('更新失败:', error);
    } finally {
      setIsUpdating(false);
    }
  };

  return { performUpdate, isUpdating };
};

/**
 * 增强的Popconfirm组件
 * 
 * 带有更好的样式和交互
 */
export interface EnhancedPopconfirmProps extends PopconfirmProps {
  type?: 'danger' | 'warning' | 'info';
  children: React.ReactElement;
}

export const EnhancedPopconfirm: React.FC<EnhancedPopconfirmProps> = ({
  type = 'warning',
  title,
  description,
  okText = '确定',
  cancelText = '取消',
  okButtonProps,
  children,
  ...props
}) => {
  const iconMap = {
    danger: <ExclamationCircleOutlined style={{ color: '#ff4d4f' }} />,
    warning: <ExclamationCircleOutlined style={{ color: '#faad14' }} />,
    info: <InfoCircleOutlined style={{ color: '#1890ff' }} />,
  };

  return (
    <Popconfirm
      title={title}
      description={description}
      icon={iconMap[type]}
      okText={okText}
      cancelText={cancelText}
      okButtonProps={{
        danger: type === 'danger',
        ...okButtonProps,
      }}
      {...props}
    >
      {children}
    </Popconfirm>
  );
};

/**
 * 批量操作确认
 * 
 * 用于批量操作的确认对话框
 */
export interface BatchOperationConfirmOptions {
  selectedCount: number;
  operation: string;
  onConfirm: () => void | Promise<void>;
  onCancel?: () => void;
  warningMessage?: string;
}

export const showBatchOperationConfirm = ({
  selectedCount,
  operation,
  onConfirm,
  onCancel,
  warningMessage,
}: BatchOperationConfirmOptions) => {
  Modal.confirm({
    title: `确认${operation}`,
    content: (
      <div>
        <p>您将要{operation} <strong>{selectedCount}</strong> 条记录。</p>
        {warningMessage && (
          <p className="text-red-500 mt-2">⚠️ {warningMessage}</p>
        )}
        <p className="mt-2">此操作不可撤销，确定要继续吗？</p>
      </div>
    ),
    icon: <ExclamationCircleOutlined style={{ color: '#faad14' }} />,
    okText: `确定${operation}`,
    cancelText: '取消',
    okType: 'danger',
    onOk: onConfirm,
    onCancel,
    width: 480,
    centered: true,
  });
};

/**
 * 操作进度反馈
 * 
 * 用于显示长时间操作的进度
 */
export interface OperationProgressOptions {
  title: string;
  total: number;
  successCount: number;
  failedCount: number;
  currentItem?: string;
}

export const showOperationProgress = (options: OperationProgressOptions) => {
  const { title, total, successCount, failedCount, currentItem } = options;
  const progress = ((successCount + failedCount) / total) * 100;

  const modal = Modal.info({
    title,
    content: (
      <div className="space-y-4">
        {/* 进度条 */}
        <div className="w-full bg-gray-200 rounded-full h-2">
          <div
            className="bg-blue-600 h-2 rounded-full transition-all duration-300"
            style={{ width: `${progress}%` }}
          />
        </div>
        
        {/* 进度信息 */}
        <div className="flex justify-between text-sm text-gray-600">
          <span>进度: {successCount + failedCount} / {total}</span>
          <span>{progress.toFixed(0)}%</span>
        </div>
        
        {/* 成功/失败统计 */}
        <div className="flex gap-4 text-sm">
          <span className="text-green-600">✓ 成功: {successCount}</span>
          {failedCount > 0 && (
            <span className="text-red-600">✗ 失败: {failedCount}</span>
          )}
        </div>
        
        {/* 当前项 */}
        {currentItem && (
          <div className="text-sm text-gray-500">
            正在处理: {currentItem}
          </div>
        )}
      </div>
    ),
    icon: null,
    okText: '后台运行',
    centered: true,
  });

  return modal;
};

/**
 * 操作成功反馈
 * 
 * 显示操作成功的消息和后续操作
 */
export interface OperationSuccessOptions {
  title: string;
  content?: React.ReactNode;
  actions?: Array<{
    label: string;
    onClick: () => void;
    type?: 'primary' | 'default';
  }>;
  autoClose?: boolean;
  autoCloseDelay?: number;
}

export const showOperationSuccess = ({
  title,
  content,
  actions = [],
  autoClose = true,
  autoCloseDelay = 3000,
}: OperationSuccessOptions) => {
  const modal = Modal.success({
    title,
    content: (
      <div className="space-y-4">
        {content}
        {actions.length > 0 && (
          <div className="flex gap-2 mt-4">
            {actions.map((action, index) => (
              <button
                key={index}
                onClick={() => {
                  action.onClick();
                  modal.destroy();
                }}
                className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
                  action.type === 'primary'
                    ? 'bg-blue-500 text-white hover:bg-blue-600'
                    : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
                }`}
              >
                {action.label}
              </button>
            ))}
          </div>
        )}
      </div>
    ),
    centered: true,
  });

  if (autoClose) {
    setTimeout(() => {
      modal.destroy();
    }, autoCloseDelay);
  }

  return modal;
};

/**
 * 数据加载错误反馈
 * 
 * 显示数据加载失败的错误信息和重试选项
 */
export interface DataLoadErrorOptions {
  title?: string;
  error: Error | string;
  onRetry?: () => void;
  showDetails?: boolean;
}

export const showDataLoadError = ({
  title = '数据加载失败',
  error,
  onRetry,
  showDetails = false,
}: DataLoadErrorOptions) => {
  const errorMessage = typeof error === 'string' ? error : error.message;
  const errorStack = typeof error === 'string' ? undefined : error.stack;

  Modal.error({
    title,
    content: (
      <div className="space-y-3">
        <p className="text-gray-700">{errorMessage}</p>
        {showDetails && errorStack && (
          <details className="text-xs text-gray-500">
            <summary className="cursor-pointer hover:text-gray-700">
              查看详细信息
            </summary>
            <pre className="mt-2 p-2 bg-gray-100 rounded overflow-auto max-h-48">
              {errorStack}
            </pre>
          </details>
        )}
        {onRetry && (
          <button
            onClick={() => {
              onRetry();
              Modal.destroyAll();
            }}
            className="mt-4 px-4 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 transition-colors"
          >
            重试
          </button>
        )}
      </div>
    ),
    centered: true,
    width: 480,
  });
};

export default {
  showEnhancedConfirm,
  EnhancedPopconfirm,
  showBatchOperationConfirm,
  showOperationProgress,
  showOperationSuccess,
  showDataLoadError,
  useOptimisticUpdate,
};

