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
 * 获取友好的错误消息
 */
export function getFriendlyErrorMessage(error: unknown, context?: string): string {
  const errorInfo = ErrorMessageHandler.formatApiError(error);
  
  // 根据上下文提供更具体的错误消息
  if (context) {
    const contextMessages: Record<string, string> = {
      'tickets.load': '加载工单列表失败',
      'tickets.create': '创建工单失败',
      'tickets.update': '更新工单失败',
      'tickets.delete': '删除工单失败',
      'tickets.assign': '分配工单失败',
      'tickets.resolve': '解决工单失败',
      'tickets.close': '关闭工单失败',
    };

    if (contextMessages[context]) {
      return `${contextMessages[context]}：${errorInfo.message}`;
    }
  }

  return errorInfo.message;
}

