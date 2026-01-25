'use client';

import { App } from 'antd';
import { CheckCircle, AlertCircle, Info, AlertTriangle } from 'lucide-react';

/**
 * 通知容器组件
 * 使用 Ant Design 的 notification API
 */
export function NotificationContainer() {
  const { notification } = App.useApp();
  return null;
}

// 导出通知工具函数，保持 API 兼容
export const notifications = {
  success: (title: string, message?: string) => {
    notification.success({
      message: title,
      description: message,
      placement: 'topRight',
      icon: <CheckCircle className="text-green-500" />,
    });
  },
  error: (title: string, message?: string) => {
    notification.error({
      message: title,
      description: message,
      placement: 'topRight',
      duration: 8,
      icon: <AlertCircle className="text-red-500" />,
    });
  },
  warning: (title: string, message?: string) => {
    notification.warning({
      message: title,
      description: message,
      placement: 'topRight',
      icon: <AlertTriangle className="text-yellow-500" />,
    });
  },
  info: (title: string, message?: string) => {
    notification.info({
      message: title,
      description: message,
      placement: 'topRight',
      icon: <Info className="text-blue-500" />,
    });
  },
};