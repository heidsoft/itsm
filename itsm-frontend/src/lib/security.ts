// 安全工具库

// XSS防护
export const xssProtection = {
  // HTML转义
  escapeHtml: (text: string): string => {
    const map: Record<string, string> = {
      '&': '&amp;',
      '<': '&lt;',
      '>': '&gt;',
      '"': '&quot;',
      "'": '&#39;',
      '/': '&#x2F;',
    };
    return text.replace(/[&<>"'\/]/g, (s) => map[s]);
  },

  // HTML反转义
  unescapeHtml: (text: string): string => {
    const map: Record<string, string> = {
      '&amp;': '&',
      '&lt;': '<',
      '&gt;': '>',
      '&quot;': '"',
      '&#39;': "'",
      '&#x2F;': '/',
    };
    return text.replace(/&(amp|lt|gt|quot|#39|#x2F);/g, (s) => map[s]);
  },

  // 清理HTML标签
  stripHtml: (html: string): string => {
    return html.replace(/<[^>]*>/g, '');
  },

  // 安全的innerHTML设置
  safeInnerHTML: (element: HTMLElement, html: string): void => {
    element.innerHTML = xssProtection.escapeHtml(html);
  },
};

// CSRF防护
export const csrfProtection = {
  // 生成CSRF令牌
  generateToken: (): string => {
    const array = new Uint8Array(32);
    crypto.getRandomValues(array);
    return Array.from(array, byte => byte.toString(16).padStart(2, '0')).join('');
  },

  // 验证CSRF令牌
  validateToken: (token: string, expectedToken: string): boolean => {
    if (!token || !expectedToken) return false;
    return token === expectedToken;
  },

  // 从meta标签获取CSRF令牌
  getTokenFromMeta: (): string | null => {
    const metaTag = document.querySelector('meta[name="csrf-token"]');
    return metaTag ? metaTag.getAttribute('content') : null;
  },
};

// 输入验证和清理
export const inputSanitization = {
  // 清理用户输入
  sanitizeInput: (input: string): string => {
    return input
      .trim()
      .replace(/[<>"'&]/g, '') // 移除潜在的XSS字符
      .substring(0, 1000); // 限制长度
  },

  // 验证邮箱格式
  isValidEmail: (email: string): boolean => {
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    return emailRegex.test(email) && email.length <= 254;
  },

  // 验证URL格式
  isValidUrl: (url: string): boolean => {
    try {
      const urlObj = new URL(url);
      return ['http:', 'https:'].includes(urlObj.protocol);
    } catch {
      return false;
    }
  },

  // 验证文件类型
  isValidFileType: (fileName: string, allowedTypes: string[]): boolean => {
    const extension = fileName.split('.').pop()?.toLowerCase();
    return extension ? allowedTypes.includes(extension) : false;
  },

  // 验证文件大小
  isValidFileSize: (fileSize: number, maxSizeInMB: number): boolean => {
    const maxSizeInBytes = maxSizeInMB * 1024 * 1024;
    return fileSize <= maxSizeInBytes;
  },
};

// 密码安全
export const passwordSecurity = {
  // 密码强度检查
  checkPasswordStrength: (password: string): {
    score: number;
    feedback: string[];
    isStrong: boolean;
  } => {
    const feedback: string[] = [];
    let score = 0;

    // 长度检查
    if (password.length >= 8) {
      score += 1;
    } else {
      feedback.push('密码长度至少需要8个字符');
    }

    // 包含小写字母
    if (/[a-z]/.test(password)) {
      score += 1;
    } else {
      feedback.push('密码需要包含小写字母');
    }

    // 包含大写字母
    if (/[A-Z]/.test(password)) {
      score += 1;
    } else {
      feedback.push('密码需要包含大写字母');
    }

    // 包含数字
    if (/\d/.test(password)) {
      score += 1;
    } else {
      feedback.push('密码需要包含数字');
    }

    // 包含特殊字符
    if (/[!@#$%^&*(),.?":{}|<>]/.test(password)) {
      score += 1;
    } else {
      feedback.push('密码需要包含特殊字符');
    }

    // 不包含常见弱密码
    const commonPasswords = ['123456', 'password', '123456789', 'qwerty', 'abc123'];
    if (commonPasswords.includes(password.toLowerCase())) {
      score = 0;
      feedback.push('请不要使用常见的弱密码');
    }

    return {
      score,
      feedback,
      isStrong: score >= 4,
    };
  },

  // 生成随机密码
  generateRandomPassword: (length = 12): string => {
    const charset = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*';
    let password = '';
    
    for (let i = 0; i < length; i++) {
      const randomIndex = Math.floor(Math.random() * charset.length);
      password += charset[randomIndex];
    }
    
    return password;
  },
};

// 会话安全
export const sessionSecurity = {
  // 检查会话是否过期
  isSessionExpired: (expirationTime: number): boolean => {
    return Date.now() > expirationTime;
  },

  // 生成会话ID
  generateSessionId: (): string => {
    const array = new Uint8Array(16);
    crypto.getRandomValues(array);
    return Array.from(array, byte => byte.toString(16).padStart(2, '0')).join('');
  },

  // 安全的localStorage操作
  // 警告：localStorage不是一个安全的地方来存储敏感信息，例如认证令牌。
  // 任何在同一域名下运行的脚本都可以访问localStorage。
  // 对于敏感数据，请使用HttpOnly cookies。
  secureStorage: {
    setItem: (key: string, value: string, expirationMinutes?: number): void => {
      const item = {
        value,
        timestamp: Date.now(),
        expiration: expirationMinutes ? Date.now() + (expirationMinutes * 60 * 1000) : null,
      };
      localStorage.setItem(key, JSON.stringify(item));
    },

    getItem: (key: string): string | null => {
      try {
        const itemStr = localStorage.getItem(key);
        if (!itemStr) return null;

        const item = JSON.parse(itemStr);
        
        // 检查是否过期
        if (item.expiration && Date.now() > item.expiration) {
          localStorage.removeItem(key);
          return null;
        }

        return item.value;
      } catch {
        return null;
      }
    },

    removeItem: (key: string): void => {
      localStorage.removeItem(key);
    },

    clear: (): void => {
      localStorage.clear();
    },
  },
};

// 内容安全策略
export const contentSecurity = {
  // 验证内容类型
  validateContentType: (contentType: string, allowedTypes: string[]): boolean => {
    return allowedTypes.some(type => contentType.includes(type));
  },

  // 检查恶意脚本
  containsMaliciousScript: (content: string): boolean => {
    const maliciousPatterns = [
      /<script[^>]*>.*?<\/script>/gi,
      /javascript:/gi,
      /on\w+\s*=/gi,
      /eval\s*\(/gi,
      /document\.write/gi,
    ];
    
    return maliciousPatterns.some(pattern => pattern.test(content));
  },

  // 清理危险内容
  sanitizeContent: (content: string): string => {
    return content
      .replace(/<script[^>]*>.*?<\/script>/gi, '')
      .replace(/javascript:/gi, '')
      .replace(/on\w+\s*=/gi, '')
      .replace(/eval\s*\(/gi, '')
      .replace(/document\.write/gi, '');
  },
};

// 网络安全
export const networkSecurity = {
  // 检查是否为HTTPS
  isHttps: (): boolean => {
    return window.location.protocol === 'https:';
  },

  // 验证请求来源
  validateOrigin: (origin: string, allowedOrigins: string[]): boolean => {
    return allowedOrigins.includes(origin);
  },

  // 生成安全的请求头
  getSecureHeaders: (csrfToken?: string): Record<string, string> => {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      'X-Requested-With': 'XMLHttpRequest',
    };

    if (csrfToken) {
      headers['X-CSRF-Token'] = csrfToken;
    }

    return headers;
  },
};

// 数据加密（简单实现，生产环境建议使用专业加密库）
// 警告：这些函数不是真正的加密，不应用于敏感数据。
export const encryption = {
  // Base64编码
  base64Encode: (text: string): string => {
    return btoa(unescape(encodeURIComponent(text)));
  },

  // Base64解码
  base64Decode: (encodedText: string): string => {
    return decodeURIComponent(escape(atob(encodedText)));
  },

  // 简单的字符串混淆（不是真正的加密）
  obfuscate: (text: string, key: string): string => {
    let result = '';
    for (let i = 0; i < text.length; i++) {
      const textChar = text.charCodeAt(i);
      const keyChar = key.charCodeAt(i % key.length);
      result += String.fromCharCode(textChar ^ keyChar);
    }
    return encryption.base64Encode(result);
  },

  // 简单的字符串反混淆
  deobfuscate: (obfuscatedText: string, key: string): string => {
    const decodedText = encryption.base64Decode(obfuscatedText);
    let result = '';
    for (let i = 0; i < decodedText.length; i++) {
      const textChar = decodedText.charCodeAt(i);
      const keyChar = key.charCodeAt(i % key.length);
      result += String.fromCharCode(textChar ^ keyChar);
    }
    return result;
  },
};

// 安全配置类型定义
interface SecurityDefaults {
  maxFileSize: number;
  allowedFileTypes: string[];
  sessionTimeout: number;
  maxLoginAttempts: number;
  passwordMinLength: number;
  csrfTokenExpiry: number;
}

// 安全配置
export const securityConfig: {
  defaults: SecurityDefaults;
  get: (key: keyof SecurityDefaults) => number | string[];
  set: (key: keyof SecurityDefaults, value: number | string[]) => void;
} = {
  // 默认安全配置
  defaults: {
    maxFileSize: 10, // MB
    allowedFileTypes: ['jpg', 'jpeg', 'png', 'gif', 'pdf', 'doc', 'docx'],
    sessionTimeout: 30, // 分钟
    maxLoginAttempts: 5,
    passwordMinLength: 8,
    csrfTokenExpiry: 60, // 分钟
  },

  // 获取配置值
  get: (key: keyof SecurityDefaults) => {
    return securityConfig.defaults[key];
  },

  // 设置配置值
  set: (key: keyof SecurityDefaults, value: number | string[]) => {
    (securityConfig.defaults as unknown as Record<string, number | string[]>)[key] = value;
  },
}

// 安全事件日志
export const securityLogger = {
  // 记录安全事件
  logSecurityEvent: (event: string, details: Record<string, unknown>): void => {
    const logEntry = {
      timestamp: new Date().toISOString(),
      event,
      details,
      userAgent: navigator.userAgent,
      url: window.location.href,
    };

    // 在开发环境下输出到控制台
    if (process.env.NODE_ENV === 'development') {
      console.warn('[Security Event]', logEntry);
    }

    // 在生产环境下可以发送到安全监控服务
    // 注意：安全监控服务集成需要在后端配置安全日志收集
    if (process.env.NODE_ENV === 'production') {
      // 未来可通过 sendToSecurityService(logEntry) 发送到安全监控服务
    }
  },

  // 记录登录尝试
  logLoginAttempt: (success: boolean, username?: string): void => {
    securityLogger.logSecurityEvent('login_attempt', {
      success,
      username: username ? `${username.substring(0, 3)}***` : 'unknown',
    });
  },

  // 记录可疑活动
  logSuspiciousActivity: (activity: string, details: Record<string, unknown>): void => {
    securityLogger.logSecurityEvent('suspicious_activity', {
      activity,
      ...details,
    });
  },
};

// 导出所有安全工具
export const security = {
  xss: xssProtection,
  csrf: csrfProtection,
  input: inputSanitization,
  password: passwordSecurity,
  session: sessionSecurity,
  content: contentSecurity,
  network: networkSecurity,
  encryption,
  config: securityConfig,
  logger: securityLogger,
};