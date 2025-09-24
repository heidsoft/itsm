'use client';

import { useUIStore } from '@/lib/store/ui-store';
import { X, CheckCircle, AlertCircle, Info, AlertTriangle } from 'lucide-react';

/**
 * 通知容器组件
 * 显示全局通知消息
 */
export function NotificationContainer() {
  const { notifications, removeNotification } = useUIStore();

  if (notifications.length === 0) {
    return null;
  }

  return (
    <div className="fixed top-4 right-4 z-50 space-y-2">
      {notifications.map((notification) => (
        <NotificationItem
          key={notification.id}
          notification={notification}
          onClose={() => removeNotification(notification.id)}
        />
      ))}
    </div>
  );
}

/**
 * 单个通知项组件
 */
interface NotificationItemProps {
  notification: {
    id: string;
    type: 'success' | 'error' | 'warning' | 'info';
    title: string;
    message?: string;
    duration?: number;
  };
  onClose: () => void;
}

function NotificationItem({ notification, onClose }: NotificationItemProps) {
  const { type, title, message } = notification;

  // 获取通知样式
  const getNotificationStyles = () => {
    switch (type) {
      case 'success':
        return {
          container: 'bg-green-50 border-green-200 text-green-800',
          icon: <CheckCircle className="h-5 w-5 text-green-400" />,
        };
      case 'error':
        return {
          container: 'bg-red-50 border-red-200 text-red-800',
          icon: <AlertCircle className="h-5 w-5 text-red-400" />,
        };
      case 'warning':
        return {
          container: 'bg-yellow-50 border-yellow-200 text-yellow-800',
          icon: <AlertTriangle className="h-5 w-5 text-yellow-400" />,
        };
      case 'info':
      default:
        return {
          container: 'bg-blue-50 border-blue-200 text-blue-800',
          icon: <Info className="h-5 w-5 text-blue-400" />,
        };
    }
  };

  const styles = getNotificationStyles();

  return (
    <div
      className={`
        max-w-sm w-full shadow-lg rounded-lg pointer-events-auto
        border ${styles.container}
        transform transition-all duration-300 ease-in-out
        animate-in slide-in-from-right-full
      `}
    >
      <div className="p-4">
        <div className="flex items-start">
          <div className="flex-shrink-0">
            {styles.icon}
          </div>
          <div className="ml-3 w-0 flex-1">
            <p className="text-sm font-medium">
              {title}
            </p>
            {message && (
              <p className="mt-1 text-sm opacity-90">
                {message}
              </p>
            )}
          </div>
          <div className="ml-4 flex-shrink-0 flex">
            <button
              className="inline-flex text-gray-400 hover:text-gray-600 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
              onClick={onClose}
            >
              <span className="sr-only">关闭</span>
              <X className="h-4 w-4" />
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}