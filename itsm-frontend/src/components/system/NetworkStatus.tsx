'use client';

/**
 * NetworkStatus · 全局网络状态提示
 *
 * 功能：
 * - 每 30s 心跳 GET /api/v1/health
 * - 离线 → 黄色 banner + 禁用「提交工单」按钮
 * - 5xx 错误累积 → 红色 banner，提示稍后重试
 *
 * 集成方式：在 (main)/layout.tsx 中嵌入 <NetworkStatus />
 * 或在 Header.tsx 中作为 Banner 渲染
 */

import React, { useEffect, useRef, useState, useCallback } from 'react';
import { Alert, Space, Button, Tag } from 'antd';
import { WifiOutlined, DisconnectOutlined, WarningOutlined } from '@ant-design/icons';

const HEALTH_CHECK_INTERVAL_MS = 30_000;
const ERROR_THRESHOLD = 3;
const HEALTH_URL = '/api/v1/health';

type Status = 'checking' | 'online' | 'offline' | 'degraded';

interface NetworkStatusProps {
  /** 是否在 Header 中以紧凑模式渲染 */
  compact?: boolean;
  /** 禁用表单提交通知（通过 CustomEvent 通知全局） */
  enableFormLock?: boolean;
}

interface HealthResponse {
  status?: string;
  uptime?: number;
  version?: string;
}

export function NetworkStatus({ compact = false, enableFormLock = true }: NetworkStatusProps) {
  const [status, setStatus] = useState<Status>('checking');
  const [consecutiveErrors, setConsecutiveErrors] = useState(0);
  const [lastChecked, setLastChecked] = useState<Date | null>(null);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const checkHealth = useCallback(async () => {
    try {
      const controller = new AbortController();
      const timeout = setTimeout(() => controller.abort(), 5000);

      const res = await fetch(HEALTH_URL, {
        method: 'GET',
        signal: controller.signal,
        cache: 'no-store',
      });
      clearTimeout(timeout);

      if (res.ok) {
        setStatus('online');
        setConsecutiveErrors(0);
      } else if (res.status >= 500) {
        setConsecutiveErrors((c) => {
          const next = c + 1;
          if (next >= ERROR_THRESHOLD) setStatus('degraded');
          return next;
        });
      } else {
        // 4xx 通常是配置问题，不算网络异常
        setStatus('online');
        setConsecutiveErrors(0);
      }
    } catch (err) {
      setConsecutiveErrors((c) => c + 1);
      if (consecutiveErrors + 1 >= ERROR_THRESHOLD) {
        setStatus('degraded');
      } else {
        setStatus('offline');
      }
    } finally {
      setLastChecked(new Date());
    }
  }, [consecutiveErrors]);

  useEffect(() => {
    checkHealth();
    intervalRef.current = setInterval(checkHealth, HEALTH_CHECK_INTERVAL_MS);
    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current);
    };
  }, [checkHealth]);

  // 离线时通过 CustomEvent 通知全局锁定表单
  useEffect(() => {
    if (!enableFormLock) return;
    if (typeof window === 'undefined') return;

    const event = new CustomEvent('itsm:network-status', {
      detail: { status, canSubmit: status === 'online' },
    });
    window.dispatchEvent(event);
  }, [status, enableFormLock]);

  if (status === 'online' || status === 'checking') {
    if (compact) {
      return (
        <Tag color="green" icon={<WifiOutlined />}>
          在线
        </Tag>
      );
    }
    return null;
  }

  if (compact) {
    return (
      <Tag color={status === 'degraded' ? 'red' : 'orange'} icon={status === 'degraded' ? <WarningOutlined /> : <DisconnectOutlined />}>
        {status === 'degraded' ? '降级' : '离线'}
      </Tag>
    );
  }

  return (
    <Alert
      banner
      type={status === 'degraded' ? 'error' : 'warning'}
      showIcon
      icon={status === 'degraded' ? <WarningOutlined /> : <DisconnectOutlined />}
      message={
        <Space>
          <span>
            {status === 'offline' && '网络连接已断开，部分功能不可用'}
            {status === 'degraded' && '后端服务异常，请稍后重试'}
          </span>
          {lastChecked && (
            <span style={{ fontSize: 12, color: '#999' }}>
              上次检测：{lastChecked.toLocaleTimeString()}
            </span>
          )}
        </Space>
      }
      action={
        <Button size="small" onClick={checkHealth}>
          立即重试
        </Button>
      }
    />
  );
}

export default NetworkStatus;
