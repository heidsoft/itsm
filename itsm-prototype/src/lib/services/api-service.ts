import axios, { AxiosInstance, AxiosRequestConfig, AxiosResponse } from "axios";
import { useAuthStore } from "../../app/lib/store";

// API响应类型定义
export interface ApiResponse<T = unknown> {
  success: boolean;
  data: T;
  message?: string;
  error_code?: string;
  timestamp: string;
}

export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  page: number;
  size: number;
  total_pages: number;
}

export interface PaginationParams {
  page?: number;
  size?: number;
  sort?: string;
  order?: 'asc' | 'desc';
}

// 业务实体基础类型
export interface BaseEntity {
  id: string;
  created_at: string;
  updated_at: string;
  created_by?: string;
  updated_by?: string;
  tenant_id: string;
}

// 错误类型
export class ApiError extends Error {
  constructor(
    message: string,
    public status: number,
    public code?: string,
    public details?: unknown
  ) {
    super(message);
    this.name = "ApiError";
  }
}

// API客户端配置
class ApiClient {
  private client: AxiosInstance;

  constructor() {
    this.client = axios.create({
      baseURL: process.env.NEXT_PUBLIC_API_BASE_URL || "http://localhost:8080",
      timeout: 10000,
      headers: {
        "Content-Type": "application/json",
      },
    });

    this.setupInterceptors();
  }

  private setupInterceptors() {
    // 请求拦截器 - 添加认证和租户信息
    this.client.interceptors.request.use(
      (config) => {
        const token = useAuthStore.getState().token;
        const currentTenant = useAuthStore.getState().currentTenant;
        
        if (token) {
          config.headers.Authorization = `Bearer ${token}`;
        }
        
        if (currentTenant?.id) {
          config.headers["X-Tenant-ID"] = currentTenant.id;
        }
        
        // 添加请求ID用于追踪
        config.headers["X-Request-ID"] = this.generateRequestId();
        
        return config;
      },
      (error) => Promise.reject(error)
    );

    // 响应拦截器 - 统一错误处理
    this.client.interceptors.response.use(
      (response: AxiosResponse<ApiResponse>) => {
        // 处理业务逻辑错误
        if (!response.data.success) {
          throw new ApiError(
            response.data.message || "请求失败",
            response.status,
            response.data.error_code,
            response.data
          );
        }
        return response;
      },
      (error) => {
        if (error.response) {
          const { status, data } = error.response;
          
          // 处理认证错误
          if (status === 401) {
            useAuthStore.getState().logout();
            window.location.href = "/login";
            return Promise.reject(new ApiError("认证失败，请重新登录", status));
          }
          
          // 处理权限错误
          if (status === 403) {
            return Promise.reject(new ApiError("权限不足", status));
          }
          
          // 处理业务错误
          if (data && data.message) {
            return Promise.reject(new ApiError(data.message, status, data.error_code));
          }
          
          return Promise.reject(new ApiError("请求失败", status));
        }
        
        if (error.request) {
          return Promise.reject(new ApiError("网络连接失败", 0));
        }
        
        return Promise.reject(new ApiError(error.message || "未知错误", 0));
      }
    );
  }

  private generateRequestId(): string {
    return `req_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }

  // 通用请求方法
  async get<T>(url: string, config?: AxiosRequestConfig): Promise<T> {
    const response = await this.client.get<ApiResponse<T>>(url, config);
    return response.data.data;
  }

  async post<T>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T> {
    const response = await this.client.post<ApiResponse<T>>(url, data, config);
    return response.data.data;
  }

  async put<T>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T> {
    const response = await this.client.put<ApiResponse<T>>(url, data, config);
    return response.data.data;
  }

  async delete<T>(url: string, config?: AxiosRequestConfig): Promise<T> {
    const response = await this.client.delete<ApiResponse<T>>(url, config);
    return response.data.data;
  }

  async patch<T>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T> {
    const response = await this.client.patch<ApiResponse<T>>(url, data, config);
    return response.data.data;
  }
}

// 创建API客户端实例
export const apiClient = new ApiClient();

// 基础服务类
export abstract class BaseService {
  protected client = apiClient;

  protected getUrl(path: string): string {
    return `/api/v1${path}`;
  }

  protected handleError(error: unknown): never {
    if (error instanceof ApiError) {
      throw error;
    }
    throw new ApiError("服务调用失败", 0);
  }
} 