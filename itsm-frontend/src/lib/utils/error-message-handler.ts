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
        message: error.message || 'Operation failed, please try again later.',
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
        message: (errorObj.message as string) || 'Operation failed, please try again later.',
        details: errorObj.details as string | undefined,
      };
    }

    return {
      message: 'Unknown error, please try again later',
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
        message: 'Network connection failed, please check network settings',
        actionText: 'Retry',
      };
    }

    if (errorInfo.message.includes('timeout')) {
      return {
        ...errorInfo,
        message: 'Request timed out, please try again later.',
        actionText: 'Retry',
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
      message: errorInfo.message || 'Data validation failed, please check input',
    };
  }

  /**
   * 格式化权限错误
   */
  static formatPermissionError(error: unknown): ErrorInfo {
    return {
      message: 'You do not have permission to perform this operation',
      code: 'PERMISSION_DENIED',
    };
  }

  /**
   * 格式化404错误
   */
  static formatNotFoundError(resource: string = 'Resources'): ErrorInfo {
    return {
      message: `${resource} Does not exist or has been deleted`,
      code: 'NOT_FOUND',
    };
  }

  /**
   * 格式化500错误
   */
  static formatServerError(): ErrorInfo {
    return {
      message: 'Server error, please try again later',
      code: 'SERVER_ERROR',
      actionText: 'Retry',
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

  let userMessage = 'Encountered a problem';

  if (severity === 'critical') {
    userMessage = "A serious error occurred in the system. Please try again later or contact the administrator.";
  } else if (severity === 'high') {
    userMessage = "Operation failed, please check your input or contact the administrator";
  } else if (severity === 'medium') {
    userMessage = "The operation was unsuccessful, please try again";
  } else if (severity === 'low') {
    userMessage = "The network may be unstable, please check the connection";
  }

  if (isRetryable) {
    userMessage += ". Can retry";
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
    'BAD_REQUEST': 'Invalid request, please check your input',
    'UNAUTHORIZED': 'Unauthorized, please log in again',
    'FORBIDDEN': 'No permission to perform this operation',
    'NOT_FOUND': 'The requested resource does not exist',
    'CONFLICT': 'Resource conflict, please refresh and try again',
    'VALIDATION_ERROR': 'Data validation failed',
    'RATE_LIMITED': 'Requests are too frequent, please try again later.',
    'INTERNAL_ERROR': 'Server internal error',
    'BAD_GATEWAY': 'Gateway error',
    'SERVICE_UNAVAILABLE': 'Service is temporarily unavailable',
    'GATEWAY_TIMEOUT': 'Gateway timeout',

    // 自定义错误码
    'NETWORK_ERROR': 'Network connection failed, please check the network',
    'TIMEOUT': 'Request timed out, please try again later.',
    'PERMISSION_DENIED': 'You do not have permission to perform this operation',
    'RESOURCE_NOT_FOUND': 'Resource does not exist',
    'VALIDATION_FAILED': 'Input data validation failed',
    'DUPLICATE_ENTRY': 'Record already exists',
    'DEPENDENCY_ERROR': 'Dependency exists and cannot be deleted',
  };

  return messages[code] || 'An error occurred';
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
    'tickets.load': 'Failed to load work order list',
    'tickets.create': 'Failed to create ticket',
    'tickets.update': 'Failed to update ticket',
    'tickets.delete': 'Failed to delete ticket',
    'tickets.assign': 'Failed to assign work order',
    'tickets.resolve': 'Failed to resolve ticket',
    'tickets.close': 'Failed to close ticket',
  };

  if (context && contextMessages[context]) {
    return `${contextMessages[context]}：${errorInfo.message}`;
  }

  return errorInfo.message;
}
