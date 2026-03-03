/**
 * WebSocket 通知服务
 * 提供实时通知推送功能
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

class NotificationWSService {
  private ws: WebSocket | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectDelay = 3000;
  private heartbeatInterval: NodeJS.Timeout | null = null;
  private notificationCallbacks: Set<NotificationCallback> = new Set();
  private connectionCallbacks: Set<ConnectionCallback> = new Set();
  private userId: number | null = null;
  private token: string | null = null;

  /**
   * 连接 WebSocket
   */
  connect(userId: number, token: string): Promise<void> {
    return new Promise((resolve, reject) => {
      this.userId = userId;
      this.token = token;

      // 获取 WebSocket URL
      const wsUrl = process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:8080/ws/notifications';
      const url = `${wsUrl}?user_id=${userId}&token=${token}`;

      try {
        this.ws = new WebSocket(url);

        this.ws.onopen = () => {
          logger.info('[NotificationWS] Connected');
          this.reconnectAttempts = 0;
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
          this.handleReconnect();
        };
      } catch (error) {
        reject(error);
      }
    });
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
    }, 30000);
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
   * 处理重连
   */
  private handleReconnect(): void {
    if (this.reconnectAttempts < this.maxReconnectAttempts) {
      this.reconnectAttempts++;
      logger.info(
        `[NotificationWS] Reconnecting... (${this.reconnectAttempts}/${this.maxReconnectAttempts})`
      );
      setTimeout(() => {
        if (this.userId && this.token) {
          this.connect(this.userId, this.token).catch(() => {});
        }
      }, this.reconnectDelay);
    } else {
      logger.warn('[NotificationWS] Max reconnection attempts reached');
    }
  }

  /**
   * 通知连接状态变化
   */
  private notifyConnectionStatus(connected: boolean): void {
    this.connectionCallbacks.forEach(cb => cb(connected));
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
   * 断开连接
   */
  disconnect(): void {
    this.stopHeartbeat();
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
    this.userId = null;
    this.token = null;
    this.reconnectAttempts = 0;
  }

  /**
   * 获取连接状态
   */
  isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN;
  }
}

// 导出单例
export const notificationWS = new NotificationWSService();

// 导出类型
export { NotificationWSService };
