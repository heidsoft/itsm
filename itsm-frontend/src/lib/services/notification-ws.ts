/**
 * WebSocket 通知服务
 * 提供实时通知推送功能，带自动重连
 */

import { logger } from '@/lib/env';
import { httpClient } from '@/lib/api/http-client';
import type { TicketNotification } from '@/lib/api/ticket-notification-api';
import type { UserNotification } from '@/lib/api/ticket-notification-api';
export type { TicketNotification } from '@/lib/api/ticket-notification-api';
export type { UserNotification } from '@/lib/api/ticket-notification-api';

export interface NotificationWSMessage {
  type: 'notification' | 'heartbeat' | 'error';
  /** 旧版前端读 data；后端 WebSocketMessage 实际下发的是 payload */
  data?: UserNotification;
  payload?: UserNotification;
  message?: string;
}

export type NotificationCallback = (notification: UserNotification) => void;
export type ConnectionCallback = (connected: boolean) => void;
export type ReconnectCallback = (attempt: number, maxAttempts: number) => void;
export type MaxAttemptsCallback = () => void;

export interface NotificationWSConfig {
  maxReconnectAttempts?: number;
  initialReconnectDelay?: number;
  maxReconnectDelay?: number;
  heartbeatInterval?: number;
}

const DEFAULT_CONFIG: Required<NotificationWSConfig> = {
  maxReconnectAttempts: 5,
  initialReconnectDelay: 1000,
  maxReconnectDelay: 30000,
  heartbeatInterval: 30000,
};

/** 去重窗口上限，防止长期会话中 Set 无界增长。 */
const MAX_SEEN_NOTIFICATIONS = 500;

class NotificationWSService {
  private ws: WebSocket | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts: number;
  private initialReconnectDelay: number;
  private maxReconnectDelay: number;
  private heartbeatIntervalMs: number;
  private reconnectDelay = 0;
  private heartbeatInterval: ReturnType<typeof setInterval> | null = null;
  private reconnectTimeout: ReturnType<typeof setTimeout> | null = null;
  private notificationCallbacks: Set<NotificationCallback> = new Set();
  private connectionCallbacks: Set<ConnectionCallback> = new Set();
  private reconnectCallbacks: Set<ReconnectCallback> = new Set();
  private maxAttemptsCallbacks: Set<MaxAttemptsCallback> = new Set();
  private userId: number | null = null;
  private shouldReconnect = true;
  private isManualDisconnect = false;

  /** Header 与通知页共享单例：只有最后一个使用者释放时才真正断开。 */
  private refCount = 0;
  /** 取票/建连进行中，避免两个组件同时 acquire 开出两条连接。 */
  private connectionPromise: Promise<void> | null = null;
  /** 重连或服务端重复投递时，同一条通知只回调一次。 */
  private seenNotificationIds: Set<number> = new Set();
  /** 指数退避耗尽后只通知一次，之后转入 keep-alive 低频重试。 */
  private maxAttemptsNotified = false;

  constructor(config: NotificationWSConfig = {}) {
    const cfg = { ...DEFAULT_CONFIG, ...config };
    this.maxReconnectAttempts = cfg.maxReconnectAttempts;
    this.initialReconnectDelay = cfg.initialReconnectDelay;
    this.maxReconnectDelay = cfg.maxReconnectDelay;
    this.heartbeatIntervalMs = cfg.heartbeatInterval;
  }

  /**
   * 连接 WebSocket
   * 流程：先 POST /api/v1/ws/ticket 获取短期票据，再用 ?ticket= 建立连接
   * 认证走 httpOnly cookie，因此这里不接收任何 token 形参。
   */
  connect(userId: number): Promise<void> {
    this.userId = userId;
    this.shouldReconnect = true;
    this.isManualDisconnect = false;

    if (this.connectionPromise) return this.connectionPromise;

    const pending = this.openConnection(userId);
    this.connectionPromise = pending;
    const clear = () => {
      if (this.connectionPromise === pending) this.connectionPromise = null;
    };
    // 取票失败时 this.ws 尚未创建，onclose 永远不会触发。
    // 必须由这里自行排队重试，否则后端一次滚动重启就让实时通道永久静默。
    pending.then(clear, () => {
      clear();
      this.handleReconnect();
    });
    return pending;
  }

  private async openConnection(userId: number): Promise<void> {
    // 1. 获取 WebSocket 短期票据（认证走 httpOnly cookie，CSRF 由 httpClient 自动处理）
    let ticket: string;
    try {
      const data = await httpClient.request<{ ticket: string }>('/api/v1/ws/ticket', {
        method: 'POST',
      });
      if (!data?.ticket) {
        throw new Error('Failed to obtain WebSocket ticket');
      }
      ticket = data.ticket;
    } catch (err) {
      logger.error('[NotificationWS] Failed to get WS ticket:', err);
      throw err;
    }

    // 2. 使用票据建立 WebSocket 连接
    const wsUrl = process.env.NEXT_PUBLIC_WS_URL || `ws://${window.location.host}/api/v1/ws/notifications`;
    const url = `${wsUrl}?ticket=${encodeURIComponent(ticket)}`;

    this.cleanup();

    return new Promise((resolve, reject) => {
      try {
        this.ws = new WebSocket(url);

        this.ws.onopen = () => {
          logger.info('[NotificationWS] Connected');
          this.reconnectAttempts = 0;
          this.reconnectDelay = 0;
          this.maxAttemptsNotified = false;
          this.startHeartbeat();
          this.notifyConnectionStatus(true);
          resolve();
        };

        this.ws.onmessage = event => {
          try {
            const message: NotificationWSMessage = JSON.parse(event.data);
            this.handleMessage(message);
          } catch (error) {
            logger.error('[NotificationWS] Failed to parse message:', error);
          }
        };

        this.ws.onerror = error => {
          logger.error('[NotificationWS] Error:', error);
          reject(error);
        };

        this.ws.onclose = event => {
          logger.info('[NotificationWS] Disconnected:', event.code, event.reason);
          this.stopHeartbeat();
          this.notifyConnectionStatus(false);

          if (!this.isManualDisconnect && this.shouldReconnect) {
            this.handleReconnect();
          }
        };
      } catch (error) {
        reject(error);
      }
    });
  }

  /**
   * 申请使用共享连接。多个组件各自 acquire/release，
   * 任一组件卸载不再触发全局 disconnect（N3）。
   */
  acquire(userId: number): Promise<void> {
    this.refCount += 1;
    this.shouldReconnect = true;
    this.isManualDisconnect = false;

    if (this.connectionPromise) return this.connectionPromise;
    if (this.isConnected() && this.userId === userId) return Promise.resolve();
    return this.connect(userId);
  }

  /** 释放共享连接；引用计数归零才真正断开。 */
  release(): void {
    if (this.refCount <= 1) {
      this.refCount = 0;
      this.disconnect();
      return;
    }
    this.refCount -= 1;
  }

  private getApiBase(): string {
    if (typeof window === 'undefined') return '';
    const proto = window.location.protocol === 'https:' ? 'https:' : 'http:';
    return `${proto}//${window.location.host}`;
  }

  /**
   * 清理资源。注意不清 reconnectTimeout：
   * 建连过程中 cleanup 会打断 connect 内部刚排队的重试链路。
   * 重连定时器只由 disconnect() 显式取消。
   */
  private cleanup(): void {
    // 停止心跳
    this.stopHeartbeat();

    // 关闭旧连接
    if (this.ws) {
      this.ws.onopen = null;
      this.ws.onmessage = null;
      this.ws.onerror = null;
      this.ws.onclose = null;

      if (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING) {
        this.ws.close();
      }
      this.ws = null;
    }
  }

  /**
   * 处理 WebSocket 消息。
   * 后端 WebSocketMessage 是 {type, payload}，前端历史契约读 data，两者都兼容。
   */
  private handleMessage(message: NotificationWSMessage): void {
    switch (message.type) {
      case 'notification': {
        const notification = message.data ?? message.payload;
        if (!notification) break;
        if (typeof notification.id === 'number') {
          if (this.seenNotificationIds.has(notification.id)) break;
          this.seenNotificationIds.add(notification.id);
          if (this.seenNotificationIds.size > MAX_SEEN_NOTIFICATIONS) {
            this.seenNotificationIds.clear();
          }
        }
        this.notificationCallbacks.forEach(cb => cb(notification));
        break;
      }
      case 'heartbeat':
        // 心跳响应
        break;
      case 'error':
        logger.error('[NotificationWS] Server error:', message.message);
        break;
    }
  }

  /**
   * 启动心跳
   */
  private startHeartbeat(): void {
    this.heartbeatInterval = setInterval(() => {
      this.send({ type: 'heartbeat' });
    }, this.heartbeatIntervalMs);
  }

  /**
   * 停止心跳
   */
  private stopHeartbeat(): void {
    if (this.heartbeatInterval) {
      clearInterval(this.heartbeatInterval);
      this.heartbeatInterval = null;
    }
  }

  /**
   * 发送消息
   */
  private send(message: object): void {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message));
    }
  }

  /**
   * 计算重连延迟（指数退避）
   */
  private calculateReconnectDelay(): number {
    const delay = this.initialReconnectDelay * Math.pow(2, this.reconnectAttempts - 1);
    return Math.min(delay, this.maxReconnectDelay);
  }

  /**
   * 处理重连。指数退避耗尽后不再彻底放弃，而是转入低频 keep-alive 重试，
   * 保证后端滚动重启后通道能自愈；同时防重避免 onerror + onclose 双调度。
   */
  private handleReconnect(): void {
    if (!this.shouldReconnect || this.isManualDisconnect || this.reconnectTimeout) return;

    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      if (!this.maxAttemptsNotified) {
        this.maxAttemptsNotified = true;
        this.notifyMaxAttemptsReached();
      }
      this.reconnectDelay = this.maxReconnectDelay;
      logger.warn('[NotificationWS] Backoff exhausted, keeping low-frequency retry alive');
    } else {
      this.reconnectAttempts++;
      this.reconnectDelay = this.calculateReconnectDelay();
      logger.info(
        `[NotificationWS] Reconnecting in ${this.reconnectDelay}ms... (${this.reconnectAttempts}/${this.maxReconnectAttempts})`
      );
      this.reconnectCallbacks.forEach(cb => cb(this.reconnectAttempts, this.maxReconnectAttempts));
    }

    this.reconnectTimeout = setTimeout(() => {
      this.reconnectTimeout = null;
      if (this.shouldReconnect && this.userId) {
        this.connect(this.userId).catch(() => {
          // connect 的 rejection 分支已排队下一次重试，避免重复调度
        });
      }
    }, this.reconnectDelay);
  }

  /**
   * 通知连接状态变化
   */
  private notifyConnectionStatus(connected: boolean): void {
    this.connectionCallbacks.forEach(cb => cb(connected));
  }

  /**
   * 通知达到最大重连次数（进入 keep-alive 前只触发一次）
   */
  private notifyMaxAttemptsReached(): void {
    this.maxAttemptsCallbacks.forEach(cb => cb());
  }

  /**
   * 订阅新通知
   */
  onNotification(callback: NotificationCallback): () => void {
    this.notificationCallbacks.add(callback);
    return () => this.notificationCallbacks.delete(callback);
  }

  /**
   * 订阅连接状态变化
   */
  onConnectionChange(callback: ConnectionCallback): () => void {
    this.connectionCallbacks.add(callback);
    return () => this.connectionCallbacks.delete(callback);
  }

  /**
   * 订阅重连尝试
   */
  onReconnect(callback: ReconnectCallback): () => void {
    this.reconnectCallbacks.add(callback);
    return () => this.reconnectCallbacks.delete(callback);
  }

  /**
   * 订阅最大重连次数达到
   */
  onMaxAttemptsReached(callback: MaxAttemptsCallback): () => void {
    this.maxAttemptsCallbacks.add(callback);
    return () => this.maxAttemptsCallbacks.delete(callback);
  }

  /**
   * 断开连接
   */
  disconnect(): void {
    this.isManualDisconnect = true;
    this.shouldReconnect = false;
    this.refCount = 0;
    if (this.reconnectTimeout) {
      clearTimeout(this.reconnectTimeout);
      this.reconnectTimeout = null;
    }
    this.cleanup();
    this.reconnectAttempts = 0;
    this.reconnectDelay = 0;
    this.notifyConnectionStatus(false);
  }

  /**
   * 手动触发重连
   */
  reconnect(): Promise<void> {
    this.shouldReconnect = true;
    this.isManualDisconnect = false;
    this.reconnectAttempts = 0;
    this.reconnectDelay = 0;
    this.maxAttemptsNotified = false;
    if (this.reconnectTimeout) {
      clearTimeout(this.reconnectTimeout);
      this.reconnectTimeout = null;
    }

    if (this.userId) {
      return this.connect(this.userId);
    }

    return Promise.reject(new Error('No userId available'));
  }

  /**
   * 获取连接状态
   */
  isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN;
  }

  /**
   * 获取重连状态
   */
  isReconnecting(): boolean {
    return this.reconnectAttempts > 0 && this.shouldReconnect;
  }

  /**
   * 获取当前重连尝试次数
   */
  getReconnectAttempts(): number {
    return this.reconnectAttempts;
  }
}

// 导出单例（使用默认配置）
export const notificationWS = new NotificationWSService();

// 导出类型
export { NotificationWSService };
