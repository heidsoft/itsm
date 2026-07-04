'use client';

import React, { useState, useEffect, useRef, useCallback } from 'react';
import { Alert } from 'antd';
import { Wifi, Server } from 'lucide-react';

const RESTORED_DISMISS_MS = 3000;

/**
 * 网络状态监听组件
 * 监听 online/offline 事件，在页面顶部显示网络状态横幅
 * - 断网时显示红色横幅
 * - 恢复时显示绿色提示，3秒后自动消失
 */
export const NetworkStatus: React.FC = () => {
  const [isOnline, setIsOnline] = useState(true);
  const [showRestored, setShowRestored] = useState(false);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // 处理网络恢复
  const handleOnline = useCallback(() => {
    setIsOnline(true);
    setShowRestored(true);
    // 清除之前的定时器，避免重复
    if (timerRef.current) {
      clearTimeout(timerRef.current);
    }
    timerRef.current = setTimeout(() => {
      setShowRestored(false);
    }, RESTORED_DISMISS_MS);
  }, []);

  // 处理网络断开
  const handleOffline = useCallback(() => {
    setIsOnline(false);
    setShowRestored(false);
    if (timerRef.current) {
      clearTimeout(timerRef.current);
      timerRef.current = null;
    }
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
      if (timerRef.current) {
        clearTimeout(timerRef.current);
      }
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
          icon={<Wifi />}
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
          icon={<Server />}
          banner
        />
      </div>
    );
  }

  return null;
};

export default NetworkStatus;
