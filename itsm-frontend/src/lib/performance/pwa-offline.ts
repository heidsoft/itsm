// @ts-nocheck
/**
 * ITSM前端性能优化 - PWA和离线支持
 * 
 * 实现Service Worker、离线缓存、推送通知等PWA功能
 */

import { useState, useEffect, useCallback } from 'react';

// ==================== Service Worker 管理 ====================

/**
 * Service Worker 管理器
 */
export class ServiceWorkerManager {
  private registration: ServiceWorkerRegistration | null = null;
  private isSupported: boolean;

  constructor() {
    this.isSupported = 'serviceWorker' in navigator;
  }

  /**
   * 注册 Service Worker
   */
  async register(swPath: string = '/sw.js'): Promise<ServiceWorkerRegistration | null> {
    if (!this.isSupported) {
      console.warn('Service Worker not supported');
      return null;
    }

    try {
      this.registration = await navigator.serviceWorker.register(swPath);
      console.log('Service Worker registered:', this.registration);
      
      // 监听更新
      this.registration.addEventListener('updatefound', () => {
        const newWorker = this.registration!.installing;
        if (newWorker) {
          newWorker.addEventListener('statechange', () => {
            if (newWorker.state === 'installed' && navigator.serviceWorker.controller) {
              // 新版本可用
              this.notifyUpdateAvailable();
            }
          });
        }
      });

      return this.registration;
    } catch (error) {
      console.error('Service Worker registration failed:', error);
      return null;
    }
  }

  /**
   * 检查更新
   */
  async checkForUpdate(): Promise<void> {
    if (this.registration) {
      await this.registration.update();
    }
  }

  /**
   * 跳过等待并激活新版本
   */
  async skipWaiting(): Promise<void> {
    if (this.registration && this.registration.waiting) {
      this.registration.waiting.postMessage({ type: 'SKIP_WAITING' });
    }
  }

  /**
   * 通知更新可用
   */
  private notifyUpdateAvailable(): void {
    // 这里可以触发UI更新通知
    window.dispatchEvent(new CustomEvent('sw-update-available'));
  }

  /**
   * 获取注册信息
   */
  getRegistration(): ServiceWorkerRegistration | null {
    return this.registration;
  }

  /**
   * 检查是否支持
   */
  isServiceWorkerSupported(): boolean {
    return this.isSupported;
  }
}

export const serviceWorkerManager = new ServiceWorkerManager();

// ==================== 缓存管理 ====================

/**
 * 缓存管理器
 */
export class CacheManager {
  private cacheName: string;
  private maxAge: number;

  constructor(cacheName: string = 'itsm-cache-v1', maxAge: number = 24 * 60 * 60 * 1000) {
    this.cacheName = cacheName;
    this.maxAge = maxAge;
  }

  /**
   * 缓存资源
   */
  async cacheResource(url: string, response?: Response): Promise<void> {
    if (!('caches' in window)) {
      console.warn('Cache API not supported');
      return;
    }

    try {
      const cache = await caches.open(this.cacheName);
      const res = response || await fetch(url);
      
      // 添加缓存时间戳
      const headers = new Headers(res.headers);
      headers.set('sw-cache-time', Date.now().toString());
      
      const cachedResponse = new Response(res.body, {
        status: res.status,
        statusText: res.statusText,
        headers: headers,
      });

      await cache.put(url, cachedResponse);
    } catch (error) {
      console.error('Failed to cache resource:', error);
    }
  }

  /**
   * 获取缓存的资源
   */
  async getCachedResource(url: string): Promise<Response | null> {
    if (!('caches' in window)) {
      return null;
    }

    try {
      const cache = await caches.open(this.cacheName);
      const response = await cache.match(url);
      
      if (response) {
        const cacheTime = response.headers.get('sw-cache-time');
        if (cacheTime) {
          const age = Date.now() - parseInt(cacheTime);
          if (age > this.maxAge) {
            // 缓存过期，删除
            await cache.delete(url);
            return null;
          }
        }
      }
      
      return response;
    } catch (error) {
      console.error('Failed to get cached resource:', error);
      return null;
    }
  }

  /**
   * 删除缓存
   */
  async deleteCache(url: string): Promise<void> {
    if (!('caches' in window)) {
      return;
    }

    try {
      const cache = await caches.open(this.cacheName);
      await cache.delete(url);
    } catch (error) {
      console.error('Failed to delete cache:', error);
    }
  }

  /**
   * 清理过期缓存
   */
  async cleanExpiredCache(): Promise<void> {
    if (!('caches' in window)) {
      return;
    }

    try {
      const cache = await caches.open(this.cacheName);
      const requests = await cache.keys();
      
      for (const request of requests) {
        const response = await cache.match(request);
        if (response) {
          const cacheTime = response.headers.get('sw-cache-time');
          if (cacheTime) {
            const age = Date.now() - parseInt(cacheTime);
            if (age > this.maxAge) {
              await cache.delete(request);
            }
          }
        }
      }
    } catch (error) {
      console.error('Failed to clean expired cache:', error);
    }
  }

  /**
   * 获取缓存大小
   */
  async getCacheSize(): Promise<number> {
    if (!('caches' in window)) {
      return 0;
    }

    try {
      const cache = await caches.open(this.cacheName);
      const requests = await cache.keys();
      let totalSize = 0;

      for (const request of requests) {
        const response = await cache.match(request);
        if (response) {
          const blob = await response.blob();
          totalSize += blob.size;
        }
      }

      return totalSize;
    } catch (error) {
      console.error('Failed to get cache size:', error);
      return 0;
    }
  }
}

export const cacheManager = new CacheManager();

// ==================== 离线检测 ====================

/**
 * 网络状态管理器
 */
export class NetworkManager {
  private isOnline: boolean = navigator.onLine;
  private listeners: Set<(isOnline: boolean) => void> = new Set();

  constructor() {
    this.setupEventListeners();
  }

  private setupEventListeners(): void {
    window.addEventListener('online', () => {
      this.isOnline = true;
      this.notifyListeners(true);
    });

    window.addEventListener('offline', () => {
      this.isOnline = false;
      this.notifyListeners(false);
    });
  }

  private notifyListeners(isOnline: boolean): void {
    this.listeners.forEach(listener => listener(isOnline));
  }

  /**
   * 添加网络状态监听器
   */
  addListener(listener: (isOnline: boolean) => void): () => void {
    this.listeners.add(listener);
    return () => this.listeners.delete(listener);
  }

  /**
   * 检查是否在线
   */
  isOnline(): boolean {
    return this.isOnline;
  }

  /**
   * 检查网络连接质量
   */
  async checkConnectionQuality(): Promise<'slow' | 'fast' | 'unknown'> {
    if (!('connection' in navigator)) {
      return 'unknown';
    }

    const connection = (navigator as any).connection;
    if (connection.effectiveType) {
      return connection.effectiveType === 'slow-2g' || connection.effectiveType === '2g' ? 'slow' : 'fast';
    }

    return 'unknown';
  }
}

export const networkManager = new NetworkManager();

// ==================== 推送通知 ====================

/**
 * 推送通知管理器
 */
export class PushNotificationManager {
  private registration: ServiceWorkerRegistration | null = null;
  private isSupported: boolean;

  constructor() {
    this.isSupported = 'Notification' in window && 'serviceWorker' in navigator;
  }

  /**
   * 请求通知权限
   */
  async requestPermission(): Promise<NotificationPermission> {
    if (!this.isSupported) {
      return 'denied';
    }

    return await Notification.requestPermission();
  }

  /**
   * 检查权限状态
   */
  getPermissionStatus(): NotificationPermission {
    if (!this.isSupported) {
      return 'denied';
    }

    return Notification.permission;
  }

  /**
   * 订阅推送
   */
  async subscribe(userVisibleOnly: boolean = true): Promise<PushSubscription | null> {
    if (!this.isSupported || this.getPermissionStatus() !== 'granted') {
      return null;
    }

    try {
      this.registration = await navigator.serviceWorker.ready;
      const subscription = await this.registration.pushManager.subscribe({
        userVisibleOnly,
        applicationServerKey: this.urlBase64ToUint8Array(process.env.NEXT_PUBLIC_VAPID_PUBLIC_KEY || ''),
      });

      return subscription;
    } catch (error) {
      console.error('Failed to subscribe to push notifications:', error);
      return null;
    }
  }

  /**
   * 取消订阅
   */
  async unsubscribe(): Promise<boolean> {
    if (!this.registration) {
      return false;
    }

    try {
      const subscription = await this.registration.pushManager.getSubscription();
      if (subscription) {
        await subscription.unsubscribe();
        return true;
      }
      return false;
    } catch (error) {
      console.error('Failed to unsubscribe from push notifications:', error);
      return false;
    }
  }

  /**
   * 发送本地通知
   */
  async sendLocalNotification(title: string, options?: NotificationOptions): Promise<void> {
    if (!this.isSupported || this.getPermissionStatus() !== 'granted') {
      return;
    }

    try {
      this.registration = await navigator.serviceWorker.ready;
      await this.registration.showNotification(title, {
        icon: '/icon-192x192.png',
        badge: '/badge-72x72.png',
        ...options,
      });
    } catch (error) {
      console.error('Failed to send local notification:', error);
    }
  }

  /**
   * 转换VAPID密钥
   */
  private urlBase64ToUint8Array(base64String: string): Uint8Array {
    const padding = '='.repeat((4 - base64String.length % 4) % 4);
    const base64 = (base64String + padding)
      .replace(/-/g, '+')
      .replace(/_/g, '/');

    const rawData = window.atob(base64);
    const outputArray = new Uint8Array(rawData.length);

    for (let i = 0; i < rawData.length; ++i) {
      outputArray[i] = rawData.charCodeAt(i);
    }
    return outputArray;
  }
}

export const pushNotificationManager = new PushNotificationManager();

// ==================== PWA Hook ====================

/**
 * 使用PWA功能
 */
export function usePWA() {
  const [isOnline, setIsOnline] = useState(navigator.onLine);
  const [swRegistration, setSwRegistration] = useState<ServiceWorkerRegistration | null>(null);
  const [updateAvailable, setUpdateAvailable] = useState(false);
  const [notificationPermission, setNotificationPermission] = useState<NotificationPermission>('default');

  useEffect(() => {
    // 监听网络状态
    const removeOnlineListener = networkManager.addListener(setIsOnline);

    // 监听Service Worker更新
    const handleUpdateAvailable = () => {
      setUpdateAvailable(true);
    };

    window.addEventListener('sw-update-available', handleUpdateAvailable);

    // 检查通知权限
    setNotificationPermission(pushNotificationManager.getPermissionStatus());

    // 注册Service Worker
    serviceWorkerManager.register().then(registration => {
      setSwRegistration(registration);
    });

    return () => {
      removeOnlineListener();
      window.removeEventListener('sw-update-available', handleUpdateAvailable);
    };
  }, []);

  const requestNotificationPermission = useCallback(async () => {
    const permission = await pushNotificationManager.requestPermission();
    setNotificationPermission(permission);
    return permission;
  }, []);

  const subscribeToPush = useCallback(async () => {
    if (notificationPermission === 'granted') {
      return await pushNotificationManager.subscribe();
    }
    return null;
  }, [notificationPermission]);

  const sendNotification = useCallback(async (title: string, options?: NotificationOptions) => {
    await pushNotificationManager.sendLocalNotification(title, options);
  }, []);

  const updateApp = useCallback(async () => {
    await serviceWorkerManager.skipWaiting();
    setUpdateAvailable(false);
  }, []);

  const checkForUpdate = useCallback(async () => {
    await serviceWorkerManager.checkForUpdate();
  }, []);

  return {
    isOnline,
    swRegistration,
    updateAvailable,
    notificationPermission,
    requestNotificationPermission,
    subscribeToPush,
    sendNotification,
    updateApp,
    checkForUpdate,
  };
}

// ==================== 离线存储 ====================

/**
 * 离线存储管理器
 */
export class OfflineStorageManager {
  private dbName: string;
  private version: number;
  private db: IDBDatabase | null = null;

  constructor(dbName: string = 'itsm-offline', version: number = 1) {
    this.dbName = dbName;
    this.version = version;
  }

  /**
   * 初始化数据库
   */
  async init(): Promise<void> {
    return new Promise((resolve, reject) => {
      const request = indexedDB.open(this.dbName, this.version);

      request.onerror = () => reject(request.error);
      request.onsuccess = () => {
        this.db = request.result;
        resolve();
      };

      request.onupgradeneeded = (event) => {
        const db = (event.target as IDBOpenDBRequest).result;
        
        // 创建工单存储
        if (!db.objectStoreNames.contains('tickets')) {
          const ticketStore = db.createObjectStore('tickets', { keyPath: 'id' });
          ticketStore.createIndex('status', 'status', { unique: false });
          ticketStore.createIndex('assignee', 'assignee_id', { unique: false });
        }

        // 创建用户存储
        if (!db.objectStoreNames.contains('users')) {
          db.createObjectStore('users', { keyPath: 'id' });
        }

        // 创建缓存存储
        if (!db.objectStoreNames.contains('cache')) {
          db.createObjectStore('cache', { keyPath: 'key' });
        }
      };
    });
  }

  /**
   * 存储数据
   */
  async store(storeName: string, data: any): Promise<void> {
    if (!this.db) {
      await this.init();
    }

    return new Promise((resolve, reject) => {
      const transaction = this.db!.transaction([storeName], 'readwrite');
      const store = transaction.objectStore(storeName);
      const request = store.put(data);

      request.onsuccess = () => resolve();
      request.onerror = () => reject(request.error);
    });
  }

  /**
   * 获取数据
   */
  async get(storeName: string, key: any): Promise<any> {
    if (!this.db) {
      await this.init();
    }

    return new Promise((resolve, reject) => {
      const transaction = this.db!.transaction([storeName], 'readonly');
      const store = transaction.objectStore(storeName);
      const request = store.get(key);

      request.onsuccess = () => resolve(request.result);
      request.onerror = () => reject(request.error);
    });
  }

  /**
   * 获取所有数据
   */
  async getAll(storeName: string): Promise<any[]> {
    if (!this.db) {
      await this.init();
    }

    return new Promise((resolve, reject) => {
      const transaction = this.db!.transaction([storeName], 'readonly');
      const store = transaction.objectStore(storeName);
      const request = store.getAll();

      request.onsuccess = () => resolve(request.result);
      request.onerror = () => reject(request.error);
    });
  }

  /**
   * 删除数据
   */
  async delete(storeName: string, key: any): Promise<void> {
    if (!this.db) {
      await this.init();
    }

    return new Promise((resolve, reject) => {
      const transaction = this.db!.transaction([storeName], 'readwrite');
      const store = transaction.objectStore(storeName);
      const request = store.delete(key);

      request.onsuccess = () => resolve();
      request.onerror = () => reject(request.error);
    });
  }
}

export const offlineStorageManager = new OfflineStorageManager();

// ==================== 离线同步 ====================

/**
 * 离线同步管理器
 */
export class OfflineSyncManager {
  private pendingActions: Array<{
    id: string;
    action: string;
    data: any;
    timestamp: number;
  }> = [];

  /**
   * 添加待同步操作
   */
  addPendingAction(action: string, data: any): string {
    const id = `${action}_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    this.pendingActions.push({
      id,
      action,
      data,
      timestamp: Date.now(),
    });
    return id;
  }

  /**
   * 同步待处理操作
   */
  async syncPendingActions(): Promise<void> {
    if (!networkManager.isOnline()) {
      return;
    }

    const actions = [...this.pendingActions];
    this.pendingActions = [];

    for (const action of actions) {
      try {
        await this.executeAction(action);
      } catch (error) {
        console.error('Failed to sync action:', action, error);
        // 重新添加到待处理列表
        this.pendingActions.push(action);
      }
    }
  }

  /**
   * 执行操作
   */
  private async executeAction(action: { action: string; data: any }): Promise<void> {
    // 这里根据action类型执行相应的API调用
    switch (action.action) {
      case 'CREATE_TICKET':
        // 调用创建工单API
        break;
      case 'UPDATE_TICKET':
        // 调用更新工单API
        break;
      case 'DELETE_TICKET':
        // 调用删除工单API
        break;
      default:
        console.warn('Unknown action:', action.action);
    }
  }

  /**
   * 获取待处理操作数量
   */
  getPendingActionsCount(): number {
    return this.pendingActions.length;
  }

  /**
   * 清除所有待处理操作
   */
  clearPendingActions(): void {
    this.pendingActions = [];
  }
}

export const offlineSyncManager = new OfflineSyncManager();

export default {
  ServiceWorkerManager,
  serviceWorkerManager,
  CacheManager,
  cacheManager,
  NetworkManager,
  networkManager,
  PushNotificationManager,
  pushNotificationManager,
  OfflineStorageManager,
  offlineStorageManager,
  OfflineSyncManager,
  offlineSyncManager,
  usePWA,
};
