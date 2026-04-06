/**
 * WebSocket 通知服务
 * 提供实时通知推送功能，带自动重连
 */

import { logger } from '@/lib/env';
import { TicketNotification } from '@/lib/api/ticket-notification-api';

export interface NotificationWSMessage {
  type: 'notification' | 'heartbeat' | 'error';
  data?: TicketNotification;
  message?: string;
}

export type NotificationCallback = (notification: TicketNotification) => void;
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
  private token: string | null = null;
  private shouldReconnect = true;
  private isManualDisconnect = false;

  constructor(config: NotificationWSConfig = {}) {
    const cfg = { ...DEFAULT_CONFIG, ...config };
    this.maxReconnectAttempts = cfg.maxReconnectAttempts;
    this.initialReconnectDelay = cfg.initialReconnectDelay;
    this.maxReconnectDelay = cfg.maxReconnectDelay;
    this.heartbeatIntervalMs = cfg.heartbeatInterval;
  }

  /**
   * 连接 WebSocket
   */
  connect(userId: number, token: string): Promise<void> {
    return new Promise((resolve, reject) => {
      this.userId = userId;
      this.token = token;
      this.shouldReconnect = true;
      this.isManualDisconnect = false;

      // 获取 WebSocket URL
      const wsUrl = process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:8080/ws/notifications';
      const url = `${wsUrl}?user_id=${userId}&token=${token}`;

      // 清理旧连接
      this.cleanup();

      try {
        this.ws = new WebSocket(url);

        this.ws.onopen = () => {
          logger.info('[NotificationWS] Connected');
          this.reconnectAttempts = 0;
          this.reconnectDelay = 0;
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
   * 清理资源
   */
  private cleanup(): void {
    // 停止心跳
    this.stopHeartbeat();

    // 清除重连定时器
    if (this.reconnectTimeout) {
      clearTimeout(this.reconnectTimeout);
      this.reconnectTimeout = null;
    }

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
   * 处理 WebSocket 消息
   */
  private handleMessage(message: NotificationWSMessage): void {
    switch (message.type) {
      case 'notification':
        if (message.data) {
          this.notificationCallbacks.forEach(cb => cb(message.data!));
        }
        break;
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
    const delay = this.initialReconnectDelay * Math.pow(2, this.reconnectAttempts);
    return Math.min(delay, this.maxReconnectDelay);
  }

  /**
   * 处理重连
   */
  private handleReconnect(): void {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      logger.warn('[NotificationWS] Max reconnection attempts reached');
      this.notifyMaxAttemptsReached();
      return;
    }

    this.reconnectAttempts++;
    this.reconnectDelay = this.calculateReconnectDelay();

    logger.info(
      `[NotificationWS] Reconnecting in ${this.reconnectDelay}ms... (${this.reconnectAttempts}/${this.maxReconnectAttempts})`
    );

    // 通知重连尝试
    this.reconnectCallbacks.forEach(cb => cb(this.reconnectAttempts, this.maxReconnectAttempts));

    this.reconnectTimeout = setTimeout(() => {
      if (this.shouldReconnect && this.userId && this.token) {
        this.connect(this.userId, this.token).catch(() => {
          // 连接失败由 onclose 处理，会触发重连
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
   * 通知达到最大重连次数
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

    if (this.userId && this.token) {
      return this.connect(this.userId, this.token);
    }

    return Promise.reject(new Error('No userId or token available'));
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
