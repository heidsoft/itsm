/**
 * ITSM前端性能优化 - Service Worker
 * 
 * 实现缓存策略、离线支持、推送通知等功能
 */

const CACHE_NAME = 'itsm-cache-v1';
const OFFLINE_CACHE_NAME = 'itsm-offline-cache-v1';
const STATIC_CACHE_NAME = 'itsm-static-cache-v1';

// 需要缓存的静态资源
const STATIC_ASSETS = [
  '/',
  '/manifest.json',
  '/icon-192x192.png',
  '/icon-512x512.png',
  '/offline.html',
];

// 需要缓存的API路径
const API_CACHE_PATTERNS = [
  /^\/api\/v1\/tickets/,
  /^\/api\/v1\/users/,
  /^\/api\/v1\/incidents/,
  /^\/api\/v1\/problems/,
  /^\/api\/v1\/changes/,
];

// 不需要缓存的路径
const EXCLUDE_PATTERNS = [
  /^\/api\/v1\/auth\/login/,
  /^\/api\/v1\/auth\/logout/,
  /^\/api\/v1\/upload/,
];

// ==================== Service Worker 安装 ====================

self.addEventListener('install', (event) => {
  console.log('Service Worker installing...');
  
  event.waitUntil(
    Promise.all([
      // 缓存静态资源
      caches.open(STATIC_CACHE_NAME).then((cache) => {
        return cache.addAll(STATIC_ASSETS);
      }),
      // 跳过等待，立即激活
      self.skipWaiting(),
    ])
  );
});

// ==================== Service Worker 激活 ====================

self.addEventListener('activate', (event) => {
  console.log('Service Worker activating...');
  
  event.waitUntil(
    Promise.all([
      // 清理旧缓存
      cleanupOldCaches(),
      // 立即控制所有客户端
      self.clients.claim(),
    ])
  );
});

// ==================== 清理旧缓存 ====================

async function cleanupOldCaches() {
  const cacheNames = await caches.keys();
  const validCaches = [CACHE_NAME, OFFLINE_CACHE_NAME, STATIC_CACHE_NAME];
  
  return Promise.all(
    cacheNames
      .filter(cacheName => !validCaches.includes(cacheName))
      .map(cacheName => caches.delete(cacheName))
  );
}

// ==================== 请求拦截和缓存策略 ====================

self.addEventListener('fetch', (event) => {
  const request = event.request;
  const url = new URL(request.url);
  
  // 只处理GET请求
  if (request.method !== 'GET') {
    return;
  }
  
  // 排除不需要缓存的请求
  if (EXCLUDE_PATTERNS.some(pattern => pattern.test(url.pathname))) {
    return;
  }
  
  // 根据请求类型选择缓存策略
  if (isStaticAsset(request)) {
    event.respondWith(cacheFirst(request));
  } else if (isApiRequest(request)) {
    event.respondWith(networkFirst(request));
  } else if (isPageRequest(request)) {
    event.respondWith(networkFirstWithOfflineFallback(request));
  } else {
    event.respondWith(staleWhileRevalidate(request));
  }
});

// ==================== 缓存策略实现 ====================

/**
 * 缓存优先策略 - 适用于静态资源
 */
async function cacheFirst(request) {
  try {
    const cachedResponse = await caches.match(request);
    if (cachedResponse) {
      return cachedResponse;
    }
    
    const networkResponse = await fetch(request);
    if (networkResponse.ok) {
      const cache = await caches.open(STATIC_CACHE_NAME);
      cache.put(request, networkResponse.clone());
    }
    
    return networkResponse;
  } catch (error) {
    console.error('Cache first strategy failed:', error);
    return new Response('Network error', { status: 503 });
  }
}

/**
 * 网络优先策略 - 适用于API请求
 */
async function networkFirst(request) {
  try {
    const networkResponse = await fetch(request);
    
    if (networkResponse.ok) {
      const cache = await caches.open(CACHE_NAME);
      cache.put(request, networkResponse.clone());
    }
    
    return networkResponse;
  } catch (error) {
    console.log('Network failed, trying cache:', error);
    
    const cachedResponse = await caches.match(request);
    if (cachedResponse) {
      return cachedResponse;
    }
    
    // 返回离线响应
    return createOfflineResponse(request);
  }
}

/**
 * 网络优先策略（带离线回退）- 适用于页面请求
 */
async function networkFirstWithOfflineFallback(request) {
  try {
    const networkResponse = await fetch(request);
    
    if (networkResponse.ok) {
      const cache = await caches.open(CACHE_NAME);
      cache.put(request, networkResponse.clone());
    }
    
    return networkResponse;
  } catch (error) {
    console.log('Network failed, trying cache:', error);
    
    const cachedResponse = await caches.match(request);
    if (cachedResponse) {
      return cachedResponse;
    }
    
    // 返回离线页面
    return caches.match('/offline.html');
  }
}

/**
 * 陈旧重新验证策略 - 适用于其他请求
 */
async function staleWhileRevalidate(request) {
  const cache = await caches.open(CACHE_NAME);
  const cachedResponse = await cache.match(request);
  
  const fetchPromise = fetch(request).then((networkResponse) => {
    if (networkResponse.ok) {
      cache.put(request, networkResponse.clone());
    }
    return networkResponse;
  });
  
  return cachedResponse || fetchPromise;
}

// ==================== 请求类型判断 ====================

function isStaticAsset(request) {
  const url = new URL(request.url);
  return url.pathname.match(/\.(js|css|png|jpg|jpeg|gif|svg|ico|woff|woff2|ttf|eot)$/);
}

function isApiRequest(request) {
  const url = new URL(request.url);
  return API_CACHE_PATTERNS.some(pattern => pattern.test(url.pathname));
}

function isPageRequest(request) {
  const url = new URL(request.url);
  return request.headers.get('accept')?.includes('text/html') && url.origin === location.origin;
}

// ==================== 离线响应创建 ====================

function createOfflineResponse(request) {
  const url = new URL(request.url);
  
  if (url.pathname.startsWith('/api/')) {
    return new Response(JSON.stringify({
      success: false,
      error: '网络连接不可用，请检查网络后重试',
      offline: true,
    }), {
      status: 503,
      headers: {
        'Content-Type': 'application/json',
      },
    });
  }
  
  return new Response('网络连接不可用', {
    status: 503,
    headers: {
      'Content-Type': 'text/plain',
    },
  });
}

// ==================== 推送通知处理 ====================

self.addEventListener('push', (event) => {
  console.log('Push event received:', event);
  
  const options = {
    body: '您有新的工单需要处理',
    icon: '/icon-192x192.png',
    badge: '/badge-72x72.png',
    vibrate: [100, 50, 100],
    data: {
      dateOfArrival: Date.now(),
      primaryKey: 1,
    },
    actions: [
      {
        action: 'explore',
        title: '查看详情',
        icon: '/icon-192x192.png',
      },
      {
        action: 'close',
        title: '关闭',
        icon: '/icon-192x192.png',
      },
    ],
  };
  
  event.waitUntil(
    self.registration.showNotification('ITSM系统通知', options)
  );
});

// ==================== 通知点击处理 ====================

self.addEventListener('notificationclick', (event) => {
  console.log('Notification click received:', event);
  
  event.notification.close();
  
  if (event.action === 'explore') {
    event.waitUntil(
      clients.openWindow('/tickets')
    );
  } else if (event.action === 'close') {
    // 关闭通知，不做任何操作
  } else {
    // 默认点击行为
    event.waitUntil(
      clients.openWindow('/')
    );
  }
});

// ==================== 后台同步处理 ====================

self.addEventListener('sync', (event) => {
  console.log('Background sync event:', event.tag);
  
  if (event.tag === 'background-sync') {
    event.waitUntil(doBackgroundSync());
  }
});

async function doBackgroundSync() {
  try {
    // 同步离线数据
    const cache = await caches.open(OFFLINE_CACHE_NAME);
    const requests = await cache.keys();
    
    for (const request of requests) {
      try {
        const response = await fetch(request);
        if (response.ok) {
          await cache.delete(request);
        }
      } catch (error) {
        console.error('Failed to sync request:', request.url, error);
      }
    }
  } catch (error) {
    console.error('Background sync failed:', error);
  }
}

// ==================== 消息处理 ====================

self.addEventListener('message', (event) => {
  console.log('Service Worker received message:', event.data);
  
  if (event.data && event.data.type === 'SKIP_WAITING') {
    self.skipWaiting();
  }
  
  if (event.data && event.data.type === 'CACHE_URLS') {
    event.waitUntil(
      caches.open(CACHE_NAME).then((cache) => {
        return cache.addAll(event.data.urls);
      })
    );
  }
  
  if (event.data && event.data.type === 'CLEAR_CACHE') {
    event.waitUntil(
      caches.keys().then((cacheNames) => {
        return Promise.all(
          cacheNames.map((cacheName) => {
            return caches.delete(cacheName);
          })
        );
      })
    );
  }
});

// ==================== 错误处理 ====================

self.addEventListener('error', (event) => {
  console.error('Service Worker error:', event.error);
});

self.addEventListener('unhandledrejection', (event) => {
  console.error('Service Worker unhandled rejection:', event.reason);
});

// ==================== 缓存管理 ====================

/**
 * 清理过期缓存
 */
async function cleanExpiredCache() {
  const cacheNames = await caches.keys();
  
  for (const cacheName of cacheNames) {
    const cache = await caches.open(cacheName);
    const requests = await cache.keys();
    
    for (const request of requests) {
      const response = await cache.match(request);
      if (response) {
        const cacheTime = response.headers.get('sw-cache-time');
        if (cacheTime) {
          const age = Date.now() - parseInt(cacheTime);
          const maxAge = 24 * 60 * 60 * 1000; // 24小时
          
          if (age > maxAge) {
            await cache.delete(request);
          }
        }
      }
    }
  }
}

// 定期清理过期缓存
setInterval(cleanExpiredCache, 60 * 60 * 1000); // 每小时清理一次

console.log('Service Worker loaded successfully');
