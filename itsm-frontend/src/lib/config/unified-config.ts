/**
 * 统一配置管理系统
 * 集中管理前端应用的所有配置项
 */

// 基础环境配置
export const ENV_CONFIG = {
  NODE_ENV: process.env.NODE_ENV || 'development',
  NEXT_PUBLIC_API_URL: process.env.NEXT_PUBLIC_API_URL || '',
  NEXT_PUBLIC_API_VERSION: process.env.NEXT_PUBLIC_API_VERSION || 'v1',
  NEXT_PUBLIC_APP_NAME: process.env.NEXT_PUBLIC_APP_NAME || 'ITSM System',
  NEXT_PUBLIC_APP_VERSION: process.env.NEXT_PUBLIC_APP_VERSION || '1.0.0',
  NEXT_PUBLIC_ENABLE_ANALYTICS: process.env.NEXT_PUBLIC_ENABLE_ANALYTICS === 'true',
  NEXT_PUBLIC_ENABLE_DEBUG: process.env.NEXT_PUBLIC_ENABLE_DEBUG === 'true',
  NEXT_PUBLIC_ENABLE_MOCK: process.env.NEXT_PUBLIC_ENABLE_MOCK === 'true',
  NEXT_PUBLIC_SENTRY_DSN: process.env.NEXT_PUBLIC_SENTRY_DSN || '',
} as const;

// API配置
export const API_CONFIG = {
  BASE_URL:
    ENV_CONFIG.NEXT_PUBLIC_API_URL ||
    (typeof window === 'undefined' ? 'http://localhost:8090' : ''),
  VERSION: ENV_CONFIG.NEXT_PUBLIC_API_VERSION,
  TIMEOUT: 30000,
  RETRY_ATTEMPTS: 3,
  RETRY_DELAY: 1000,
  ENDPOINTS: {
    AUTH: '/auth',
    USERS: '/users',
    TICKETS: '/tickets',
    SERVICES: '/services',
    TENANTS: '/tenants',
    SYSTEM: '/system',
  },
} as const;

// 应用配置
export const APP_CONFIG = {
  NAME: ENV_CONFIG.NEXT_PUBLIC_APP_NAME,
  VERSION: ENV_CONFIG.NEXT_PUBLIC_APP_VERSION,
  DESCRIPTION: 'IT Service Management System',
  AUTHOR: 'ITSM Team',
  HOMEPAGE: '/',

  // 功能开关
  FEATURES: {
    MULTI_TENANT: true,
    RBAC: true,
    WORKFLOW: true,
    KNOWLEDGE_BASE: true,
    REPORTING: true,
    NOTIFICATIONS: true,
    AUDIT_LOG: true,
    EXPORT: true,
    IMPORT: true,
    API_DOCS: true,
    DARK_MODE: true,
    MOBILE_APP: true,
  },

  // 主题配置
  THEME: {
    DEFAULT_MODE: 'light',
    PRIMARY_COLOR: '#1890ff',
    SUCCESS_COLOR: '#52c41a',
    WARNING_COLOR: '#faad14',
    ERROR_COLOR: '#ff4d4f',
    INFO_COLOR: '#1890ff',

    // 布局配置
    LAYOUT: {
      SIDEBAR_WIDTH: 240,
      SIDEBAR_COLLAPSED_WIDTH: 80,
      HEADER_HEIGHT: 64,
      CONTENT_PADDING: 24,
    },

    // 断点配置
    BREAKPOINTS: {
      XS: 480,
      SM: 576,
      MD: 768,
      LG: 992,
      XL: 1200,
      XXL: 1600,
    },
  },

  // 分页配置
  PAGINATION: {
    DEFAULT_PAGE_SIZE: 20,
    PAGE_SIZE_OPTIONS: [10, 20, 50, 100],
    MAX_PAGE_SIZE: 100,
  },

  // 表格配置
  TABLE: {
    DEFAULT_PAGE_SIZE: 20,
    SHOW_SIZE_CHANGER: true,
    SHOW_QUICK_JUMPER: true,
    SHOW_TOTAL: true,
    SCROLL_X: 1200,
    ROW_HEIGHT: 54,
  },

  // 表单配置
  FORM: {
    LAYOUT: 'horizontal',
    LABEL_COL: { span: 6 },
    WRAPPER_COL: { span: 18 },
    AUTO_COMPLETE: 'off',
    SUBMIT_ON_ENTER: false,
  },

  // 文件上传配置
  UPLOAD: {
    MAX_FILE_SIZE: 50 * 1024 * 1024, // 50MB
    ALLOWED_TYPES: [
      'image/jpeg',
      'image/png',
      'image/gif',
      'application/pdf',
      'application/msword',
      'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
      'application/vnd.ms-excel',
      'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
      'text/plain',
      'text/csv',
    ],
    MAX_FILE_COUNT: 10,
    CHUNK_SIZE: 1024 * 1024, // 1MB
  },

  // 缓存配置
  CACHE: {
    API_CACHE_TTL: 5 * 60 * 1000, // 5分钟
    USER_CACHE_TTL: 30 * 60 * 1000, // 30分钟
    SYSTEM_CACHE_TTL: 60 * 60 * 1000, // 1小时
    MAX_CACHE_SIZE: 100,
  },

  // 本地存储配置
  STORAGE: {
    TOKEN_KEY: 'itsm_token',
    REFRESH_TOKEN_KEY: 'itsm_refresh_token',
    USER_KEY: 'itsm_user',
    SETTINGS_KEY: 'itsm_settings',
    THEME_KEY: 'itsm_theme',
    LANGUAGE_KEY: 'itsm_language',
    PREFERENCES_KEY: 'itsm_preferences',
  },
} as const;

// 开发配置
export const DEV_CONFIG = {
  ENABLE_MOCK_API: ENV_CONFIG.NEXT_PUBLIC_ENABLE_MOCK,
  ENABLE_DEBUG_LOGS: ENV_CONFIG.NEXT_PUBLIC_ENABLE_DEBUG,
  ENABLE_PERFORMANCE_MONITORING: ENV_CONFIG.NODE_ENV === 'development',
  LOG_LEVEL: ENV_CONFIG.NEXT_PUBLIC_ENABLE_DEBUG ? 'debug' : 'info',

  // 调试工具
  DEBUG_TOOLS: {
    REDUX_DEVTOOLS: true,
    REACT_QUERY_DEVTOOLS: true,
    STORYBOOK: true,
    VOLUME_METER: false,
  },

  // 性能监控
  PERFORMANCE: {
    ENABLE_PROFILING: true,
    SAMPLE_RATE: 1.0,
    TRACK_INTERACTIONS: true,
    TRACK_NAVIGATION: true,
    TRACK_RESOURCES: true,
  },
} as const;

// 生产配置
export const PROD_CONFIG = {
  ENABLE_ANALYTICS: ENV_CONFIG.NEXT_PUBLIC_ENABLE_ANALYTICS,
  SENTRY_DSN: ENV_CONFIG.NEXT_PUBLIC_SENTRY_DSN,
  LOG_LEVEL: 'error',

  // 性能优化
  PERFORMANCE: {
    ENABLE_LAZY_LOADING: true,
    ENABLE_CODE_SPLITTING: true,
    ENABLE_TREE_SHAKING: true,
    ENABLE_MINIFICATION: true,
  },

  // 安全配置
  SECURITY: {
    ENABLE_CSP: true,
    ENABLE_HSTS: true,
    CSRF_PROTECTION: true,
    XSS_PROTECTION: true,
  },
} as const;

// 路由配置
export const ROUTE_CONFIG = {
  // 公共路由（无需认证）
  PUBLIC_ROUTES: ['/login', '/register', '/forgot-password', '/reset-password', '/404', '/500'],

  // 管理员路由
  ADMIN_ROUTES: [
    '/admin/users',
    '/admin/tenants',
    '/admin/system',
    '/admin/logs',
    '/admin/settings',
  ],

  // 默认路由
  DEFAULT_ROUTE: '/dashboard',
  LOGIN_ROUTE: '/login',
  HOME_ROUTE: '/',

  // 路由守卫配置
  GUARD: {
    REQUIRE_AUTH: true,
    REQUIRE_TENANT: true,
    REQUIRE_PERMISSION: true,
    CHECK_USER_STATUS: true,
    CHECK_TENANT_STATUS: true,
  },
} as const;

// 权限配置
export const PERMISSION_CONFIG = {
  // 角色定义
  ROLES: {
    SUPER_ADMIN: 'super_admin',
    TENANT_ADMIN: 'tenant_admin',
    MANAGER: 'manager',
    AGENT: 'agent',
    USER: 'user',
  },

  // 权限定义
  PERMISSIONS: {
    // 用户管理
    USER_CREATE: 'user:create',
    USER_READ: 'user:read',
    USER_UPDATE: 'user:update',
    USER_DELETE: 'user:delete',

    // 工单管理
    TICKET_CREATE: 'ticket:create',
    TICKET_READ: 'ticket:read',
    TICKET_UPDATE: 'ticket:update',
    TICKET_DELETE: 'ticket:delete',
    TICKET_ASSIGN: 'ticket:assign',
    TICKET_CLOSE: 'ticket:close',

    // 服务管理
    SERVICE_CREATE: 'service:create',
    SERVICE_READ: 'service:read',
    SERVICE_UPDATE: 'service:update',
    SERVICE_DELETE: 'service:delete',

    // 租户管理
    TENANT_CREATE: 'tenant:create',
    TENANT_READ: 'tenant:read',
    TENANT_UPDATE: 'tenant:update',
    TENANT_DELETE: 'tenant:delete',

    // 系统管理
    SYSTEM_CONFIG: 'system:config',
    SYSTEM_LOGS: 'system:logs',
    SYSTEM_BACKUP: 'system:backup',
    SYSTEM_MONITOR: 'system:monitor',
  },

  // 角色权限映射
  ROLE_PERMISSIONS: {
    SUPER_ADMIN: ['*'], // 所有权限
    TENANT_ADMIN: ['user:*', 'ticket:*', 'service:*', 'system:read'],
    MANAGER: ['user:read', 'user:update', 'ticket:*', 'service:*'],
    AGENT: ['ticket:read', 'ticket:update', 'ticket:assign', 'ticket:close', 'service:read'],
    USER: ['ticket:create', 'ticket:read', 'service:read'],
  },
} as const;

// 国际化配置
export const I18N_CONFIG = {
  DEFAULT_LOCALE: 'zh-CN',
  SUPPORTED_LOCALES: ['zh-CN', 'en-US'],
  FALLBACK_LOCALE: 'zh-CN',

  // 资源路径
  RESOURCES_PATH: '/locales',

  // 格式化配置
  DATETIME: {
    DATE_FORMAT: 'YYYY-MM-DD',
    TIME_FORMAT: 'HH:mm:ss',
    DATETIME_FORMAT: 'YYYY-MM-DD HH:mm:ss',
    TIMEZONE: 'Asia/Shanghai',
  },

  NUMBER: {
    DECIMAL_PLACES: 2,
    THOUSAND_SEPARATOR: ',',
  },

  CURRENCY: {
    CODE: 'CNY',
    SYMBOL: '¥',
  },
} as const;

// 日志配置
export const LOG_CONFIG = {
  LEVEL: ENV_CONFIG.NEXT_PUBLIC_ENABLE_DEBUG ? 'debug' : 'info',

  // 日志级别映射
  LEVEL_MAP: {
    DEBUG: 0,
    INFO: 1,
    WARN: 2,
    ERROR: 3,
    FATAL: 4,
  },

  // 日志颜色
  COLORS: {
    DEBUG: '#7C7C7C',
    INFO: '#1890FF',
    WARN: '#FAAD14',
    ERROR: '#FF4D4F',
    FATAL: '#722ED1',
  },

  // 控制台输出配置
  CONSOLE: {
    ENABLED: true,
    GROUP: true,
    TIMESTAMPS: true,
    STACK_TRACE: true,
  },

  // 远程日志配置
  REMOTE: {
    ENABLED: ENV_CONFIG.NODE_ENV === 'production',
    ENDPOINT: '/api/logs',
    BATCH_SIZE: 10,
    FLUSH_INTERVAL: 5000,
  },
} as const;

// 配置验证函数
export const validateConfig = () => {
  const errors: string[] = [];

  // 验证必需的环境变量
  if (!ENV_CONFIG.NEXT_PUBLIC_API_URL && typeof window === 'undefined') {
    errors.push('NEXT_PUBLIC_API_URL is required for server-side rendering');
  }

  // 验证文件大小限制
  if (APP_CONFIG.UPLOAD.MAX_FILE_SIZE <= 0) {
    errors.push('MAX_FILE_SIZE must be greater than 0');
  }

  // 验证分页配置
  if (APP_CONFIG.PAGINATION.DEFAULT_PAGE_SIZE <= 0) {
    errors.push('DEFAULT_PAGE_SIZE must be greater than 0');
  }

  // 验证权限配置
  const requiredPermissions = Object.values(PERMISSION_CONFIG.PERMISSIONS);
  const allPermissions = Object.values(PERMISSION_CONFIG.ROLE_PERMISSIONS).flat();

  for (const permission of requiredPermissions) {
    const permissionValue = permission as string;
    if (
      permissionValue !== '*' &&
      !allPermissions.includes(permission as any) &&
      !allPermissions.includes('*' as any)
    ) {
      errors.push(`Permission ${permissionValue} is not assigned to any role`);
    }
  }

  return {
    isValid: errors.length === 0,
    errors,
  };
};

// 获取当前环境配置
export const getCurrentConfig = () => {
  const isDev = ENV_CONFIG.NODE_ENV === 'development';
  const isProd = ENV_CONFIG.NODE_ENV === 'production';
  const isTest = ENV_CONFIG.NODE_ENV === 'test';

  return {
    ENV_CONFIG,
    API_CONFIG,
    APP_CONFIG,
    ...(isDev && { DEV_CONFIG }),
    ...(isProd && { PROD_CONFIG }),
    ROUTE_CONFIG,
    PERMISSION_CONFIG,
    I18N_CONFIG,
    LOG_CONFIG,
    isDev,
    isProd,
    isTest,
  };
};

// 配置初始化函数
export const initConfig = () => {
  const validation = validateConfig();

  if (!validation.isValid) {
    console.error('Configuration validation failed:', validation.errors);

    if (ENV_CONFIG.NODE_ENV === 'production') {
      throw new Error('Invalid configuration detected');
    }
  }

  if (ENV_CONFIG.NEXT_PUBLIC_ENABLE_DEBUG) {
    console.group('🔧 Configuration Debug');
    console.groupEnd();
  }

  return getCurrentConfig();
};

// 导出配置获取器
export const getConfig = (path?: string) => {
  const config = getCurrentConfig();

  if (!path) {
    return config;
  }

  // Use any to allow dynamic property access
  return path.split('.').reduce((obj: any, key: string) => {
    return obj?.[key];
  }, config);
};

// 导出所有配置
export {
  ENV_CONFIG as env,
  API_CONFIG as api,
  APP_CONFIG as app,
  DEV_CONFIG as dev,
  PROD_CONFIG as prod,
  ROUTE_CONFIG as routes,
  PERMISSION_CONFIG as permissions,
  I18N_CONFIG as i18n,
  LOG_CONFIG as logs,
};
