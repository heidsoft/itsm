'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { Alert } from 'antd';
import { WifiOutlined, CloudServerOutlined } from '@ant-design/icons';

/**
 * 网络状态监听组件
 * 监听 online/offline 事件，在页面顶部显示网络状态横幅
 * - 断网时显示红色横幅
 * - 恢复时显示绿色提示，3秒后自动消失
 */
export const NetworkStatus: React.FC = () => {
  const [isOnline, setIsOnline] = useState(true);
  const [showRestored, setShowRestored] = useState(false);

  // 处理网络恢复
  const handleOnline = useCallback(() => {
    setIsOnline(true);
    setShowRestored(true);
    // 3秒后自动消失
    const timer = setTimeout(() => {
      setShowRestored(false);
    }, 3000);
    return () => clearTimeout(timer);
  }, []);

  // 处理网络断开
  const handleOffline = useCallback(() => {
    setIsOnline(false);
    setShowRestored(false);
  }, []);

  useEffect(() => {
    // 初始化时获取当前网络状态
    if (typeof navigator !== 'undefined') {
      setIsOnline(navigator.onLine);
    }

    window.addEventListener('online', handleOnline);
    window.addEventListener('offline', handleOffline);

    return () => {
      window.removeEventListener('online', handleOnline);
      window.removeEventListener('offline', handleOffline);
    };
  }, [handleOnline, handleOffline]);

  // 网络断开 — 显示红色横幅
  if (!isOnline) {
    return (
      <div
        style={{
          position: 'fixed',
          top: 0,
          left: 0,
          right: 0,
          zIndex: 9999,
        }}
      >
        <Alert
          message="网络连接已断开"
          description="请检查您的网络连接，部分功能可能不可用。"
          type="error"
          showIcon
          icon={<WifiOutlined />}
          banner
        />
      </div>
    );
  }

  // 网络恢复 — 显示绿色提示（3秒后消失）
  if (showRestored) {
    return (
      <div
        style={{
          position: 'fixed',
          top: 0,
          left: 0,
          right: 0,
          zIndex: 9999,
          transition: 'opacity 0.3s ease',
        }}
      >
        <Alert
          message="网络已恢复"
          description="网络连接已恢复正常。"
          type="success"
          showIcon
          icon={<CloudServerOutlined />}
          banner
        />
      </div>
    );
  }

  return null;
};

export default NetworkStatus;
