/**
 * 统一错误消息处理工具
 * 用于统一格式化错误消息，提供友好的错误提示
 */

export interface ErrorInfo {
  code?: string | number;
  message: string;
  details?: string;
  actionText?: string;
  onAction?: () => void;
}

/**
 * 错误 severity 级别
 */
export type ErrorSeverity = 'low' | 'medium' | 'high' | 'critical';

/**
 * 统一错误消息格式
 */
export class ErrorMessageHandler {
  /**
   * 格式化API错误消息
   */
  static formatApiError(error: unknown): ErrorInfo {
    if (error instanceof Error) {
      return {
        message: error.message || '操作失败，请稍后重试',
        details: error.stack,
      };
    }

    if (typeof error === 'string') {
      return {
        message: error,
      };
    }

    if (typeof error === 'object' && error !== null) {
      const errorObj = error as Record<string, unknown>;
      return {
        code: errorObj.code as string | number | undefined,
        message: (errorObj.message as string) || '操作失败，请稍后重试',
        details: errorObj.details as string | undefined,
      };
    }

    return {
      message: '未知错误，请稍后重试',
    };
  }

  /**
   * 格式化网络错误
   */
  static formatNetworkError(error: unknown): ErrorInfo {
    const errorInfo = this.formatApiError(error);

    if (errorInfo.message.includes('fetch') || errorInfo.message.includes('network')) {
      return {
        ...errorInfo,
        message: '网络连接失败，请检查网络设置',
        actionText: '重试',
      };
    }

    if (errorInfo.message.includes('timeout')) {
      return {
        ...errorInfo,
        message: '请求超时，请稍后重试',
        actionText: '重试',
      };
    }

    return errorInfo;
  }

  /**
   * 格式化验证错误
   */
  static formatValidationError(error: unknown): ErrorInfo {
    const errorInfo = this.formatApiError(error);

    return {
      ...errorInfo,
      message: errorInfo.message || '数据验证失败，请检查输入',
    };
  }

  /**
   * 格式化权限错误
   */
  static formatPermissionError(error: unknown): ErrorInfo {
    return {
      message: '您没有权限执行此操作',
      code: 'PERMISSION_DENIED',
    };
  }

  /**
   * 格式化404错误
   */
  static formatNotFoundError(resource: string = '资源'): ErrorInfo {
    return {
      message: `${resource}不存在或已被删除`,
      code: 'NOT_FOUND',
    };
  }

  /**
   * 格式化500错误
   */
  static formatServerError(): ErrorInfo {
    return {
      message: '服务器错误，请稍后重试',
      code: 'SERVER_ERROR',
      actionText: '重试',
    };
  }
}

/**
 * ===== 函数式API（供测试和兼容使用）=====
 */

/**
 * 从错误对象中提取错误消息
 */
export function getErrorMessage(error: unknown): string {
  if (error instanceof Error) {
    return error.message;
  }

  if (typeof error === 'string') {
    return error;
  }

  if (typeof error === 'object' && error !== null) {
    const errorObj = error as Record<string, unknown>;
    return (errorObj.message as string) || 'An unknown error occurred';
  }

  return 'An unknown error occurred';
}

/**
 * 格式化API错误响应
 */
export function formatApiError(error: unknown): string {
  const errorInfo = ErrorMessageHandler.formatApiError(error);
  const parts: string[] = [];

  if (errorInfo.code) {
    parts.push(`[${errorInfo.code}]`);
  }

  parts.push(errorInfo.message);

  if (errorInfo.details) {
    parts.push(`- ${errorInfo.details}`);
  }

  return parts.join(' ');
}

/**
 * 判断错误是否可重试
 */
export function isRetryableError(error: unknown): boolean {
  if (typeof error !== 'object' || error === null) {
    return false;
  }

  const errorObj = error as Record<string, unknown>;

  // 基于状态码
  if (typeof errorObj.status === 'number') {
    return [502, 503, 504, 408].includes(errorObj.status);
  }

  // 基于错误代码
  if (typeof errorObj.code === 'string') {
    return ['NETWORK_ERROR', 'TIMEOUT', 'SERVICE_UNAVAILABLE', 'RATE_LIMITED'].includes(errorObj.code);
  }

  return false;
}

/**
 * 获取错误严重程度
 */
export function getErrorSeverity(error: unknown): ErrorSeverity {
  if (typeof error !== 'object' || error === null) {
    return 'medium';
  }

  const errorObj = error as Record<string, unknown>;

  // 基于HTTP状态码
  if (typeof errorObj.status === 'number') {
    const status = errorObj.status as number;
    if (status >= 500) return 'critical';
    if (status >= 400) return 'high';
    if (status >= 300) return 'medium';
    return 'low';
  }

  // 基于自定义错误代码
  if (typeof errorObj.code === 'string') {
    const code = errorObj.code as string;
    if (code.includes('AUTH') || code.includes('PERMISSION')) return 'high';
    if (code.includes('VALIDATION') || code.includes('CONFLICT')) return 'medium';
    if (code.includes('NETWORK') || code.includes('TIMEOUT')) return 'low';
  }

  return 'medium';
}

/**
 * 记录错误到控制台或日志系统
 */
export function logError(error: unknown, context?: Record<string, unknown>): void {
  const errorMessage = getErrorMessage(error);
  const timestamp = new Date().toISOString();

  if (typeof console !== 'undefined' && console.error) {
    console.error(`[${timestamp}] Error${context ? ` in ${JSON.stringify(context)}` : ''}:`, error);
  }

  // 这里可以集成到实际的日志系统，如 Sentry、LogRocket 等
  // if (window.Sentry) { window.Sentry.captureException(error); }
}

/**
 * 创建用户友好的错误消息
 */
export function createUserFriendlyError(error: unknown): string {
  const severity = getErrorSeverity(error);
  const isRetryable = isRetryableError(error);

  let userMessage = '遇到了一个问题';

  if (severity === 'critical') {
    userMessage = '系统出现严重错误，请稍后重试或联系管理员';
  } else if (severity === 'high') {
    userMessage = '操作失败，请检查输入或联系管理员';
  } else if (severity === 'medium') {
    userMessage = '操作未成功，请重试';
  } else if (severity === 'low') {
    userMessage = '网络可能不稳定，请检查连接';
  }

  if (isRetryable) {
    userMessage += '。可以进行重试';
  }

  return userMessage;
}

/**
 * 验证错误消息格式
 */
export function validateErrorMessage(error: unknown): boolean {
  if (error === null || error === undefined) {
    return false;
  }

  if (typeof error === 'string') {
    return error.length > 0;
  }

  if (error instanceof Error) {
    return error.message.length > 0;
  }

  if (typeof error === 'object') {
    const errorObj = error as Record<string, unknown>;
    return typeof errorObj.message === 'string' && errorObj.message.length > 0;
  }

  return false;
}

/**
 * 从错误中解析错误代码
 */
export function parseErrorCode(error: unknown): string {
  if (typeof error !== 'object' || error === null) {
    return 'UNKNOWN_ERROR';
  }

  const errorObj = error as Record<string, unknown>;

  // 优先使用 code 字段
  if (typeof errorObj.code === 'string') {
    return errorObj.code as string;
  }

  // 从状态码推断
  if (typeof errorObj.status === 'number') {
    const status = errorObj.status as number;
    switch (status) {
      case 400: return 'BAD_REQUEST';
      case 401: return 'UNAUTHORIZED';
      case 403: return 'FORBIDDEN';
      case 404: return 'NOT_FOUND';
      case 409: return 'CONFLICT';
      case 422: return 'VALIDATION_ERROR';
      case 429: return 'RATE_LIMITED';
      case 500: return 'INTERNAL_ERROR';
      case 502: return 'BAD_GATEWAY';
      case 503: return 'SERVICE_UNAVAILABLE';
      case 504: return 'GATEWAY_TIMEOUT';
      default: return 'UNKNOWN_ERROR';
    }
  }

  return 'UNKNOWN_ERROR';
}

/**
 * 根据错误代码获取预定义的错误消息
 */
export function getErrorMessageByCode(code: string): string {
  const messages: Record<string, string> = {
    // HTTP 状态码
    'BAD_REQUEST': '请求无效，请检查输入',
    'UNAUTHORIZED': '未授权，请重新登录',
    'FORBIDDEN': '没有权限执行此操作',
    'NOT_FOUND': '请求的资源不存在',
    'CONFLICT': '资源冲突，请刷新后重试',
    'VALIDATION_ERROR': '数据验证失败',
    'RATE_LIMITED': '请求过于频繁，请稍后重试',
    'INTERNAL_ERROR': '服务器内部错误',
    'BAD_GATEWAY': '网关错误',
    'SERVICE_UNAVAILABLE': '服务暂时不可用',
    'GATEWAY_TIMEOUT': '网关超时',

    // 自定义错误码
    'NETWORK_ERROR': '网络连接失败，请检查网络',
    'TIMEOUT': '请求超时，请稍后重试',
    'PERMISSION_DENIED': '您没有权限执行此操作',
    'RESOURCE_NOT_FOUND': '资源不存在',
    'VALIDATION_FAILED': '输入数据验证失败',
    'DUPLICATE_ENTRY': '记录已存在',
    'DEPENDENCY_ERROR': '存在依赖关系，无法删除',
  };

  return messages[code] || '发生了一个错误';
}

/**
 * 将错误映射到建议操作
 */
export function mapErrorToAction(error: unknown): string {
  const code = parseErrorCode(error);
  const severity = getErrorSeverity(error);

  const actionMap: Record<string, string> = {
    'UNAUTHORIZED': 'login',
    'FORBIDDEN': 'request_permission',
    'NOT_FOUND': 'create',
    'VALIDATION_ERROR': 'correct_input',
    'RATE_LIMITED': 'wait',
    'NETWORK_ERROR': 'retry',
    'TIMEOUT': 'retry',
    'SERVICE_UNAVAILABLE': 'retry_later',
    'INTERNAL_ERROR': 'contact_support',
    'DEPENDENCY_ERROR': 'remove_dependencies_first',
  };

  return actionMap[code] || 'none';
}

/**
 * 获取友好的错误消息（旧版兼容）
 */
export function getFriendlyErrorMessage(error: unknown, context?: string): string {
  const errorInfo = ErrorMessageHandler.formatApiError(error);

  // 根据上下文提供更具体的错误消息
  const contextMessages: Record<string, string> = {
    'tickets.load': '加载工单列表失败',
    'tickets.create': '创建工单失败',
    'tickets.update': '更新工单失败',
    'tickets.delete': '删除工单失败',
    'tickets.assign': '分配工单失败',
    'tickets.resolve': '解决工单失败',
    'tickets.close': '关闭工单失败',
  };

  if (context && contextMessages[context]) {
    return `${contextMessages[context]}：${errorInfo.message}`;
  }

  return errorInfo.message;
}
