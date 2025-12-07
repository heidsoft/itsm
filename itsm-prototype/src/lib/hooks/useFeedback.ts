/**
 * 统一的操作反馈Hook
 * 提供统一的成功、错误、警告、信息提示
 */

import { App } from 'antd';
import { useCallback } from 'react';

export const useFeedback = () => {
  const { message, notification } = App.useApp();

  // 成功提示
  const showSuccess = useCallback(
    (content: string, duration: number = 2) => {
      message.success({
        content,
        duration,
      });
    },
    [message]
  );

  // 错误提示
  const showError = useCallback(
    (content: string, duration: number = 3) => {
      message.error({
        content,
        duration,
      });
    },
    [message]
  );

  // 警告提示
  const showWarning = useCallback(
    (content: string, duration: number = 2) => {
      message.warning({
        content,
        duration,
      });
    },
    [message]
  );

  // 信息提示
  const showInfo = useCallback(
    (content: string, duration: number = 2) => {
      message.info({
        content,
        duration,
      });
    },
    [message]
  );

  // 加载提示
  const showLoading = useCallback(
    (content: string, key: string) => {
      message.loading({
        content,
        key,
        duration: 0,
      });
    },
    [message]
  );

  // 隐藏加载提示
  const hideLoading = useCallback(
    (key: string) => {
      message.destroy(key);
    },
    [message]
  );

  // 成功通知（带详细信息）
  const showSuccessNotification = useCallback(
    (title: string, description?: string, duration: number = 4.5) => {
      notification.success({
        message: title,
        description,
        duration,
        placement: 'topRight',
      });
    },
    [notification]
  );

  // 错误通知（带详细信息）
  const showErrorNotification = useCallback(
    (title: string, description?: string, duration: number = 4.5) => {
      notification.error({
        message: title,
        description,
        duration,
        placement: 'topRight',
      });
    },
    [notification]
  );

  // 警告通知（带详细信息）
  const showWarningNotification = useCallback(
    (title: string, description?: string, duration: number = 4.5) => {
      notification.warning({
        message: title,
        description,
        duration,
        placement: 'topRight',
      });
    },
    [notification]
  );

  return {
    showSuccess,
    showError,
    showWarning,
    showInfo,
    showLoading,
    hideLoading,
    showSuccessNotification,
    showErrorNotification,
    showWarningNotification,
  };
};

