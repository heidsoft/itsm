'use client';

import React, { createContext, useContext, useState, useCallback, useRef, useEffect } from 'react';
import { message, Modal } from 'antd';
import type { MessageType } from 'antd/es/message/interface';

// 反馈类型定义
export interface FeedbackConfig {
  /** 消息类型 */
  type?: 'success' | 'error' | 'warning' | 'info';
  /** 消息内容 */
  content: string;
  /** 持续时间(秒) */
  duration?: number;
  /** 关闭回调 */
  onClose?: () => void;
  /** 点击回调 */
  onClick?: () => void;
  /** 自定义图标 */
  icon?: React.ReactNode;
  /** 自定义类名 */
  className?: string;
  /** 位置 */
  placement?: 'top' | 'topLeft' | 'topRight' | 'bottom' | 'bottomLeft' | 'bottomRight';
}

export interface ConfirmConfig {
  /** 标题 */
  title?: string;
  /** 内容 */
  content: React.ReactNode;
  /** 确认按钮文字 */
  okText?: string;
  /** 取消按钮文字 */
  cancelText?: string;
  /** 确认回调 */
  onOk?: () => Promise<void> | void;
  /** 取消回调 */
  onCancel?: () => void;
  /** 确认按钮类型 */
  okType?: 'primary' | 'default' | 'danger';
  /** 图标类型 */
  type?: 'info' | 'success' | 'error' | 'warning' | 'confirm';
  /** 宽度 */
  width?: number | string;
  /** 是否居中 */
  centered?: boolean;
  /** 自定义类名 */
  className?: string;
}

export interface LoadingConfig {
  /** 加载内容 */
  content?: string;
  /** 延迟显示时间(ms) */
  delay?: number;
  /** 遮罩层 */
  mask?: boolean;
  /** 可关闭 */
  closable?: boolean;
  /** 自定义类名 */
  className?: string;
}

// 交互反馈上下文
interface InteractionFeedbackContextType {
  showSuccess: (content: string, config?: Omit<FeedbackConfig, 'content' | 'type'>) => MessageType;
  showError: (content: string, config?: Omit<FeedbackConfig, 'content' | 'type'>) => MessageType;
  showWarning: (content: string, config?: Omit<FeedbackConfig, 'content' | 'type'>) => MessageType;
  showInfo: (content: string, config?: Omit<FeedbackConfig, 'content' | 'type'>) => MessageType;
  showFeedback: (config: FeedbackConfig) => MessageType;
  showConfirm: (config: ConfirmConfig) => Promise<boolean>;
  showLoading: (config?: LoadingConfig) => () => void;
  clearAll: () => void;
}

const InteractionFeedbackContext = createContext<InteractionFeedbackContextType | null>(null);

/**
 * 交互反馈Provider组件
 */
export const InteractionFeedbackProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [loadingStates, setLoadingStates] = useState<Map<string, any>>(new Map());
  const loadingTimers = useRef<Map<string, NodeJS.Timeout>>(new Map());

  // 统一的消息显示方法
  const showMessage = useCallback((type: FeedbackConfig['type'], content: string, config: Omit<FeedbackConfig, 'content' | 'type'> = {}) => {
    const { duration = 3, onClose, onClick, icon, className, placement = 'top' } = config;
    
    return message.open({
      type,
      content,
      duration,
      onClose,
      onClick,
      icon,
      className,
      // 全局样式优化
      style: {
        marginTop: '24px',
        borderRadius: '8px',
        boxShadow: '0 4px 12px rgba(0, 0, 0, 0.15)',
      },
    });
  }, []);

  // 快捷方法
  const showSuccess = useCallback((content: string, config: Omit<FeedbackConfig, 'content' | 'type'> = {}) => {
    return showMessage('success', content, { ...config, duration: config.duration || 2.5 });
  }, [showMessage]);

  const showError = useCallback((content: string, config: Omit<FeedbackConfig, 'content' | 'type'> = {}) => {
    return showMessage('error', content, { ...config, duration: config.duration || 4 });
  }, [showMessage]);

  const showWarning = useCallback((content: string, config: Omit<FeedbackConfig, 'content' | 'type'> = {}) => {
    return showMessage('warning', content, { ...config, duration: config.duration || 3 });
  }, [showMessage]);

  const showInfo = useCallback((content: string, config: Omit<FeedbackConfig, 'content' | 'type'> = {}) => {
    return showMessage('info', content, config);
  }, [showMessage]);

  // 通用反馈方法
  const showFeedback = useCallback((config: FeedbackConfig) => {
    const { type = 'info', content, duration = 3, onClose, onClick, icon, className } = config;
    
    return message.open({
      type,
      content,
      duration,
      onClose,
      onClick,
      icon,
      className,
    });
  }, []);

  // 确认对话框
  const showConfirm = useCallback((config: ConfirmConfig): Promise<boolean> => {
    const {
      title = '确认操作',
      content,
      okText = '确认',
      cancelText = '取消',
      onOk,
      onCancel,
      okType = 'primary',
      type = 'confirm',
      width = 416,
      centered = true,
      className,
    } = config;

    return new Promise((resolve) => {
      Modal.confirm({
        title,
        content,
        okText,
        cancelText,
        okType,
        centered,
        width,
        className,
        // 全局样式优化
        styles: {
          body: { padding: '20px 24px' },
        },
        onOk: async () => {
          try {
            await onOk?.();
            resolve(true);
          } catch (error) {
            console.error('确认操作失败:', error);
            resolve(false);
          }
        },
        onCancel: () => {
          onCancel?.();
          resolve(false);
        },
      });
    });
  }, []);

  // 加载状态管理
  const showLoading = useCallback((config: LoadingConfig = {}) => {
    const { 
      content = '加载中...', 
      delay = 0, 
      mask = true, 
      closable = false,
      className 
    } = config;
    
    const loadingId = Date.now().toString();
    
    const hideLoading = () => {
      const timer = loadingTimers.current.get(loadingId);
      if (timer) {
        clearTimeout(timer);
        loadingTimers.current.delete(loadingId);
      }
      
      const loading = loadingStates.get(loadingId);
      if (loading) {
        loading();
        loadingStates.delete(loadingId);
      }
    };
    
    if (delay > 0) {
      const timer = setTimeout(() => {
        const hide = message.loading({
          content,
          duration: 0,
          className,
          style: {
            marginTop: '24px',
            borderRadius: '8px',
          },
        });
        loadingStates.set(loadingId, hide);
      }, delay);
      
      loadingTimers.current.set(loadingId, timer);
    } else {
      const hide = message.loading({
        content,
        duration: 0,
        className,
        style: {
          marginTop: '24px',
          borderRadius: '8px',
        },
      });
      loadingStates.set(loadingId, hide);
    }
    
    return hideLoading;
  }, []);

  // 清除所有消息
  const clearAll = useCallback(() => {
    message.destroy();
    Modal.destroyAll();
    
    // 清除所有加载状态
    loadingStates.forEach((hide) => hide());
    loadingStates.clear();
    
    // 清除所有定时器
    loadingTimers.current.forEach((timer) => clearTimeout(timer));
    loadingTimers.current.clear();
  }, []);

  useEffect(() => {
    return () => {
      // 清理定时器
      loadingTimers.current.forEach((timer) => clearTimeout(timer));
    };
  }, []);

  const contextValue: InteractionFeedbackContextType = {
    showSuccess,
    showError,
    showWarning,
    showInfo,
    showFeedback,
    showConfirm,
    showLoading,
    clearAll,
  };

  return (
    <InteractionFeedbackContext.Provider value={contextValue}>
      {children}
    </InteractionFeedbackContext.Provider>
  );
};

/**
 * 使用交互反馈的Hook
 */
export const useInteractionFeedback = (): InteractionFeedbackContextType => {
  const context = useContext(InteractionFeedbackContext);
  
  if (!context) {
    throw new Error('useInteractionFeedback must be used within InteractionFeedbackProvider');
  }
  
  return context;
};

/**
 * 操作确认Hook
 * 提供常用操作的确认逻辑
 */
export const useConfirmActions = () => {
  const { showConfirm, showSuccess, showError } = useInteractionFeedback();
  
  const confirmDelete = useCallback(async (itemName: string, onConfirm: () => Promise<void>) => {
    const confirmed = await showConfirm({
      title: '确认删除',
      content: `确定要删除 "${itemName}" 吗？此操作不可恢复。`,
      okText: '删除',
      okType: 'danger',
      type: 'warning',
    });
    
    if (confirmed) {
      try {
        await onConfirm();
        showSuccess(`删除成功`);
        return true;
      } catch (error) {
        showError('删除失败，请重试');
        return false;
      }
    }
    
    return false;
  }, [showConfirm, showSuccess, showError]);

  const confirmSave = useCallback(async (itemName: string, onConfirm: () => Promise<void>) => {
    const confirmed = await showConfirm({
      title: '确认保存',
      content: `确定要保存对 "${itemName}" 的更改吗？`,
      okText: '保存',
      type: 'info',
    });
    
    if (confirmed) {
      try {
        await onConfirm();
        showSuccess('保存成功');
        return true;
      } catch (error) {
        showError('保存失败，请重试');
        return false;
      }
    }
    
    return false;
  }, [showConfirm, showSuccess, showError]);

  const confirmCancel = useCallback(async (itemName: string, hasUnsavedChanges = false, onConfirm: () => void) => {
    const content = hasUnsavedChanges 
      ? `确定要取消编辑 "${itemName}" 吗？未保存的更改将会丢失。`
      : `确定要取消操作吗？`;
      
    const confirmed = await showConfirm({
      title: '确认取消',
      content,
      okText: '确定',
      type: hasUnsavedChanges ? 'warning' : 'info',
    });
    
    if (confirmed) {
      onConfirm();
      return true;
    }
    
    return false;
  }, [showConfirm]);

  return {
    confirmDelete,
    confirmSave,
    confirmCancel,
  };
};

/**
 * 操作反馈Hook
 * 提供操作成功/失败的反馈逻辑
 */
export const useOperationFeedback = () => {
  const { showSuccess, showError, showWarning, showInfo } = useInteractionFeedback();
  
  const handleOperation = useCallback(
    async <T>(
      operation: () => Promise<T>,
      options: {
        successMessage?: string;
        errorMessage?: string;
        warningMessage?: string;
        infoMessage?: string;
        onSuccess?: (result: T) => void;
        onError?: (error: Error) => void;
        showLoading?: boolean;
      } = {}
    ): Promise<T | null> => {
      const {
        successMessage = '操作成功',
        errorMessage = '操作失败，请重试',
        warningMessage,
        infoMessage,
        onSuccess,
        onError,
        showLoading = true,
      } = options;

      if (showLoading) {
        infoMessage && showInfo(infoMessage);
      }

      try {
        const result = await operation();
        if (successMessage) showSuccess(successMessage);
        onSuccess?.(result);
        return result;
      } catch (error) {
        const err = error as Error;
        if (warningMessage) {
          showWarning(warningMessage);
        } else {
          showError(`${errorMessage}: ${err.message}`);
        }
        onError?.(err);
        return null;
      }
    },
    [showSuccess, showError, showWarning, showInfo]
  );

  return {
    handleOperation,
  };
};

export default InteractionFeedbackProvider;