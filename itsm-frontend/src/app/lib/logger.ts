/**
 * Production-safe Logger
 * Provides consistent logging with production environment controls
 */

type LogLevel = 'debug' | 'info' | 'warn' | 'error';

// Check if we're in production
const isProduction = process.env.NODE_ENV === 'production';

let currentLevel: LogLevel = isProduction ? 'warn' : 'debug';

export function setLogLevel(level: LogLevel): void {
  currentLevel = level;
}

function shouldLog(level: LogLevel): boolean {
  const levels: Record<LogLevel, number> = {
    debug: 0,
    info: 1,
    warn: 2,
    error: 3,
  };
  return levels[level] >= levels[currentLevel];
}

function formatMessage(message: string, context?: Record<string, unknown>): string {
  if (!context || Object.keys(context).length === 0) {
    return message;
  }
  const contextStr = Object.entries(context)
    .map(([key, value]) => `${key}=${typeof value === 'object' ? JSON.stringify(value) : value}`)
    .join(' ');
  return `${message} [${contextStr}]`;
}

export const logger = {
  debug(message: string, context?: Record<string, unknown>): void {
    if (shouldLog('debug')) {
      console.debug(formatMessage(message, context));
    }
  },

  info(message: string, context?: Record<string, unknown>): void {
    if (shouldLog('info')) {
      console.info(formatMessage(message, context));
    }
  },

  warn(message: string, context?: Record<string, unknown>): void {
    if (shouldLog('warn')) {
      console.warn(formatMessage(message, context));
    }
  },

  error(message: string, error?: Error | unknown, context?: Record<string, unknown>): void {
    if (shouldLog('error')) {
      let fullMessage = formatMessage(message, context);
      if (error instanceof Error) {
        fullMessage += ` Error: ${error.message}`;
        if (!isProduction && error.stack) {
          fullMessage += `\n${error.stack}`;
        }
      } else if (error !== undefined) {
        fullMessage += ` Error: ${String(error)}`;
      }
      console.error(fullMessage);
    }
  },

  // For API responses - never log sensitive data
  apiSuccess(operation: string, duration?: number): void {
    if (shouldLog('info')) {
      const context = duration ? { duration: `${duration}ms` } : undefined;
      logger.info(`API success: ${operation}`, context);
    }
  },

  apiError(operation: string, statusCode?: number): void {
    logger.warn(`API error: ${operation}`, { status_code: statusCode });
  },
};

export default logger;
