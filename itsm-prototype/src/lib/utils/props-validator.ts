/**
 * Props验证和空值处理工具
 * 提供运行时Props验证和默认值处理
 */

/**
 * 验证必需的props是否存在
 */
export function validateRequiredProps<T extends object>(
  props: T,
  requiredKeys: (keyof T)[],
  componentName: string
): boolean {
  const missingProps = requiredKeys.filter(key => {
    const value = props[key];
    return value === undefined || value === null;
  });

  if (missingProps.length > 0) {
    console.error(
      `[${componentName}] Missing required props: ${missingProps.join(', ')}`
    );
    
    if (process.env.NODE_ENV === 'development') {
      throw new Error(
        `${componentName}: Required props are missing - ${missingProps.join(', ')}`
      );
    }
    
    return false;
  }

  return true;
}

/**
 * 为props提供默认值
 */
export function withDefaults<T extends object>(
  props: Partial<T>,
  defaults: T
): T {
  return { ...defaults, ...props } as T;
}

/**
 * 深度合并对象，处理嵌套的默认值
 */
export function deepMergeDefaults<T extends object>(
  props: Partial<T>,
  defaults: T
): T {
  const result = { ...defaults };

  for (const key in props) {
    const propsValue = props[key];
    const defaultValue = defaults[key];

    if (propsValue === undefined || propsValue === null) {
      continue;
    }

    if (
      typeof propsValue === 'object' &&
      !Array.isArray(propsValue) &&
      typeof defaultValue === 'object' &&
      !Array.isArray(defaultValue)
    ) {
      result[key] = deepMergeDefaults(
        propsValue as Partial<any>,
        defaultValue as any
      );
    } else {
      result[key] = propsValue as T[Extract<keyof T, string>];
    }
  }

  return result;
}

/**
 * 验证props类型
 */
export function validatePropType<T>(
  value: any,
  expectedType: string,
  propName: string,
  componentName: string
): value is T {
  const actualType = Array.isArray(value) ? 'array' : typeof value;

  if (actualType !== expectedType) {
    console.warn(
      `[${componentName}] Invalid prop type for '${propName}': expected '${expectedType}', got '${actualType}'`
    );
    return false;
  }

  return true;
}

/**
 * 安全获取嵌套对象属性
 */
export function safeGet<T, K extends keyof T>(
  obj: T | undefined | null,
  key: K,
  defaultValue?: T[K]
): T[K] | undefined {
  if (obj === undefined || obj === null) {
    return defaultValue;
  }

  const value = obj[key];
  return value !== undefined ? value : defaultValue;
}

/**
 * 安全获取深度嵌套属性
 */
export function safeGetNested<T>(
  obj: any,
  path: string,
  defaultValue?: T
): T | undefined {
  try {
    const keys = path.split('.');
    let current = obj;

    for (const key of keys) {
      if (current === undefined || current === null) {
        return defaultValue;
      }
      current = current[key];
    }

    return current !== undefined ? current : defaultValue;
  } catch (error) {
    console.warn(`Error accessing nested property: ${path}`, error);
    return defaultValue;
  }
}

/**
 * 数组props验证
 */
export function validateArrayProp<T>(
  value: unknown,
  propName: string,
  componentName: string,
  minLength?: number,
  maxLength?: number
): value is T[] {
  if (!Array.isArray(value)) {
    console.warn(
      `[${componentName}] '${propName}' should be an array, got ${typeof value}`
    );
    return false;
  }

  if (minLength !== undefined && value.length < minLength) {
    console.warn(
      `[${componentName}] '${propName}' should have at least ${minLength} items, got ${value.length}`
    );
    return false;
  }

  if (maxLength !== undefined && value.length > maxLength) {
    console.warn(
      `[${componentName}] '${propName}' should have at most ${maxLength} items, got ${value.length}`
    );
    return false;
  }

  return true;
}

/**
 * 函数props验证
 */
export function validateFunctionProp(
  value: unknown,
  propName: string,
  componentName: string
): value is Function {
  if (typeof value !== 'function') {
    console.warn(
      `[${componentName}] '${propName}' should be a function, got ${typeof value}`
    );
    return false;
  }
  return true;
}

/**
 * 创建带默认值的Props Hook
 */
export function usePropsWithDefaults<T extends object>(
  props: Partial<T>,
  defaults: T,
  componentName?: string
): T {
  return React.useMemo(() => {
    const merged = withDefaults(props, defaults);

    // 在开发环境中验证props
    if (process.env.NODE_ENV === 'development' && componentName) {
      for (const key in merged) {
        if (merged[key] === undefined) {
          console.warn(
            `[${componentName}] Prop '${String(key)}' is undefined after merging with defaults`
          );
        }
      }
    }

    return merged;
  }, [props, defaults, componentName]);
}

// React import for hook
import React from 'react';

/**
 * HOC: 为组件添加Props验证
 */
export function withPropsValidation<P extends object>(
  Component: React.ComponentType<P>,
  validator: (props: P) => boolean,
  fallback?: React.ReactNode
) {
  const ValidatedComponent = (props: P) => {
    const isValid = validator(props);

    if (!isValid) {
      if (fallback) {
        return <>{fallback}</>;
      }

      if (process.env.NODE_ENV === 'development') {
        return (
          <div style={{ padding: 20, background: '#fff3cd', border: '1px solid #ffc107' }}>
            <strong>Props Validation Error</strong>
            <p>Component: {Component.displayName || Component.name}</p>
            <p>Please check the console for details.</p>
          </div>
        );
      }

      return null;
    }

    return <Component {...props} />;
  };

  ValidatedComponent.displayName = `withPropsValidation(${Component.displayName || Component.name})`;

  return ValidatedComponent;
}

/**
 * 创建安全的事件处理器
 */
export function createSafeHandler<T extends (...args: any[]) => any>(
  handler: T | undefined,
  fallback?: T
): T {
  return (((...args: any[]) => {
    try {
      if (handler && typeof handler === 'function') {
        return handler(...args);
      } else if (fallback && typeof fallback === 'function') {
        return fallback(...args);
      }
    } catch (error) {
      console.error('Error in event handler:', error);
    }
  }) as unknown) as T;
}

/**
 * 验证并清理对象props
 */
export function sanitizeObjectProp<T extends object>(
  value: unknown,
  propName: string,
  componentName: string,
  defaultValue: T
): T {
  if (typeof value !== 'object' || value === null || Array.isArray(value)) {
    console.warn(
      `[${componentName}] '${propName}' should be an object, got ${typeof value}. Using default value.`
    );
    return defaultValue;
  }

  return value as T;
}

export default {
  validateRequiredProps,
  withDefaults,
  deepMergeDefaults,
  validatePropType,
  safeGet,
  safeGetNested,
  validateArrayProp,
  validateFunctionProp,
  usePropsWithDefaults,
  withPropsValidation,
  createSafeHandler,
  sanitizeObjectProp,
};

