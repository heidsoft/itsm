/**
 * 统一API响应处理工具
 * 用于统一处理API响应格式，提供数据验证和默认值
 */

// API响应接口
export interface ApiResponse<T> {
  code: number;
  message: string;
  data: T;
}

// 分页响应接口
export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

/**
 * 统一处理API响应
 * @param response API响应数据
 * @param defaultValue 默认值
 * @returns 处理后的数据
 */
export function handleApiResponse<T>(
  response: ApiResponse<T> | T | null | undefined,
  defaultValue: T
): T {
  if (!response) {
    return defaultValue;
  }

  // 如果已经是目标类型，直接返回
  if (typeof response === 'object' && 'code' in response) {
    const apiResponse = response as ApiResponse<T>;
    if (apiResponse.code === 0 && apiResponse.data !== undefined) {
      return apiResponse.data;
    }
    return defaultValue;
  }

  // 如果直接是数据，返回数据
  return (response as T) || defaultValue;
}

/**
 * 处理分页响应
 * @param response API响应数据
 * @returns 处理后的分页数据
 */
export function handlePaginatedResponse<T>(
  response: ApiResponse<PaginatedResponse<T>> | PaginatedResponse<T> | null | undefined
): PaginatedResponse<T> {
  const defaultValue: PaginatedResponse<T> = {
    data: [],
    total: 0,
    page: 1,
    page_size: 20,
    total_pages: 0,
  };

  if (!response) {
    return defaultValue;
  }

  // 如果包含 code 字段，提取 data
  if (typeof response === 'object' && 'code' in response) {
    const apiResponse = response as ApiResponse<PaginatedResponse<T>>;
    if (apiResponse.code === 0 && apiResponse.data) {
      return {
        data: Array.isArray(apiResponse.data.data) ? apiResponse.data.data : [],
        total: apiResponse.data.total || 0,
        page: apiResponse.data.page || 1,
        page_size: apiResponse.data.page_size || 20,
        total_pages: apiResponse.data.total_pages || 0,
      };
    }
    return defaultValue;
  }

  // 如果直接是分页数据
  const paginatedData = response as PaginatedResponse<T>;
  return {
    data: Array.isArray(paginatedData.data) ? paginatedData.data : [],
    total: paginatedData.total || 0,
    page: paginatedData.page || 1,
    page_size: paginatedData.page_size || 20,
    total_pages: paginatedData.total_pages || 0,
  };
}

/**
 * 处理数组响应
 * @param response API响应数据
 * @returns 处理后的数组
 */
export function handleArrayResponse<T>(
  response: ApiResponse<T[]> | T[] | null | undefined
): T[] {
  if (!response) {
    return [];
  }

  // 如果包含 code 字段，提取 data
  if (typeof response === 'object' && 'code' in response) {
    const apiResponse = response as ApiResponse<T[]>;
    if (apiResponse.code === 0 && Array.isArray(apiResponse.data)) {
      return apiResponse.data;
    }
    return [];
  }

  // 如果直接是数组
  if (Array.isArray(response)) {
    return response;
  }

  return [];
}

/**
 * 处理对象响应
 * @param response API响应数据
 * @param defaultValue 默认值
 * @returns 处理后的对象
 */
export function handleObjectResponse<T extends Record<string, unknown>>(
  response: ApiResponse<T> | T | null | undefined,
  defaultValue: T
): T {
  if (!response) {
    return defaultValue;
  }

  // 如果包含 code 字段，提取 data
  if (typeof response === 'object' && 'code' in response) {
    const apiResponse = response as ApiResponse<T>;
    if (apiResponse.code === 0 && apiResponse.data) {
      return { ...defaultValue, ...apiResponse.data };
    }
    return defaultValue;
  }

  // 如果直接是对象
  if (typeof response === 'object') {
    return { ...defaultValue, ...(response as T) };
  }

  return defaultValue;
}

/**
 * 验证响应数据
 * @param data 数据
 * @param validator 验证函数
 * @returns 是否有效
 */
export function validateResponse<T>(
  data: T | null | undefined,
  validator: (data: T) => boolean
): data is T {
  if (!data) {
    return false;
  }
  return validator(data);
}

/**
 * 安全获取嵌套属性
 * @param obj 对象
 * @param path 属性路径
 * @param defaultValue 默认值
 * @returns 属性值
 */
export function safeGet<T>(
  obj: unknown,
  path: string,
  defaultValue: T
): T {
  if (!obj || typeof obj !== 'object') {
    return defaultValue;
  }

  const keys = path.split('.');
  let current: any = obj;

  for (const key of keys) {
    if (current === null || current === undefined || typeof current !== 'object') {
      return defaultValue;
    }
    current = current[key];
  }

  return current !== undefined && current !== null ? current : defaultValue;
}

