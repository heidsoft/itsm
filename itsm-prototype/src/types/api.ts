/**
 * API 通用类型定义
 * 统一管理所有API相关的类型定义
 */

// 基础响应类型
export interface BaseResponse<T = unknown> {
  code: number;
  message: string;
  data: T;
  timestamp?: string;
  requestId?: string;
}

// API响应类型别名
export type ApiResponse<T = unknown> = BaseResponse<T>;

// 分页响应类型
export interface PaginatedResponse<T> extends BaseResponse<{
  items: T[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}> {}

// 分页请求参数
export interface PaginationParams {
  page?: number;
  pageSize?: number;
  sortBy?: string;
  sortOrder?: 'asc' | 'desc';
}

// 搜索参数
export interface SearchParams {
  keyword?: string;
  filters?: Record<string, unknown>;
}

// 批量操作请求
export interface BatchOperationRequest {
  ids: number[];
  action: string;
  data?: Record<string, unknown>;
}

// 批量操作响应
export interface BatchOperationResponse {
  success: number;
  failed: number;
  errors: Array<{
    id: number;
    error: string;
  }>;
}

// 文件上传响应
export interface FileUploadResponse {
  id: number;
  filename: string;
  originalName: string;
  fileSize: number;
  mimeType: string;
  url: string;
  uploadedAt: string;
}

// 错误类型
export interface ApiError {
  code: number;
  message: string;
  details?: Record<string, unknown>;
  field?: string;
}

// 操作结果
export interface OperationResult {
  success: boolean;
  message: string;
  data?: unknown;
}

// 统计数据类型
export interface StatsData {
  total: number;
  active: number;
  inactive: number;
  percentage?: number;
}

// 趋势数据类型
export interface TrendData {
  date: string;
  value: number;
  label?: string;
}

// 图表数据类型
export interface ChartData {
  labels: string[];
  datasets: Array<{
    label: string;
    data: number[];
    backgroundColor?: string | string[];
    borderColor?: string | string[];
  }>;
}

// 导出选项
export interface ExportOptions {
  format: 'excel' | 'csv' | 'pdf';
  fields?: string[];
  filters?: Record<string, unknown>;
  filename?: string;
}

// 导入结果
export interface ImportResult {
  total: number;
  success: number;
  failed: number;
  errors: Array<{
    row: number;
    field: string;
    error: string;
  }>;
}
