// Modern API client with TypeScript and proper error handling
import axios, { AxiosInstance, AxiosRequestConfig, AxiosResponse, AxiosError } from 'axios';

// Types
export interface ApiResponse<T = any> {
  success: boolean;
  data?: T;
  error?: {
    code: string;
    message: string;
    details?: string;
  };
}

export interface ApiError {
  code: string;
  message: string;
  details?: string;
  status?: number;
}

export interface PaginatedResponse<T> {
  items: T[];
  total_count: number;
  page: number;
  page_size: number;
  has_next: boolean;
  has_previous: boolean;
}

export interface RequestConfig extends AxiosRequestConfig {
  skipAuth?: boolean;
  skipErrorHandling?: boolean;
  retry?: {
    attempts: number;
    delay: number;
    backoff: boolean;
  };
}

// API Client Configuration
interface ApiClientConfig {
  baseURL: string;
  timeout: number;
  retries: number;
  retryDelay: number;
  enableLogging: boolean;
}

const DEFAULT_CONFIG: ApiClientConfig = {
  baseURL: process.env.NEXT_PUBLIC_API_BASE_URL || 'http://localhost:8080/api/v1',
  timeout: 30000,
  retries: 3,
  retryDelay: 1000,
  enableLogging: process.env.NODE_ENV === 'development',
};

// Custom Error Classes
export class ApiClientError extends Error {
  public code: string;
  public status?: number;
  public details?: string;

  constructor(message: string, code: string, status?: number, details?: string) {
    super(message);
    this.name = 'ApiClientError';
    this.code = code;
    this.status = status;
    this.details = details;
  }
}

export class NetworkError extends ApiClientError {
  constructor(message: string = 'Network connection failed') {
    super(message, 'NETWORK_ERROR');
  }
}

export class ValidationError extends ApiClientError {
  constructor(message: string, details?: string) {
    super(message, 'VALIDATION_ERROR', 400, details);
  }
}

export class AuthenticationError extends ApiClientError {
  constructor(message: string = 'Authentication failed') {
    super(message, 'AUTH_ERROR', 401);
  }
}

export class AuthorizationError extends ApiClientError {
  constructor(message: string = 'Access denied') {
    super(message, 'AUTHORIZATION_ERROR', 403);
  }
}

export class NotFoundError extends ApiClientError {
  constructor(message: string = 'Resource not found') {
    super(message, 'NOT_FOUND', 404);
  }
}

export class ServerError extends ApiClientError {
  constructor(message: string = 'Internal server error') {
    super(message, 'SERVER_ERROR', 500);
  }
}

// API Client Class
export class ApiClient {
  private client: AxiosInstance;
  private config: ApiClientConfig;
  private authToken: string | null = null;
  private tenantId: string | null = null;

  constructor(config: Partial<ApiClientConfig> = {}) {
    this.config = { ...DEFAULT_CONFIG, ...config };
    this.client = this.createAxiosInstance();
    this.setupInterceptors();
  }

  private createAxiosInstance(): AxiosInstance {
    return axios.create({
      baseURL: this.config.baseURL,
      timeout: this.config.timeout,
      headers: {
        'Content-Type': 'application/json',
        'Accept': 'application/json',
      },
    });
  }

  private setupInterceptors(): void {
    // Request interceptor
    this.client.interceptors.request.use(
      (config) => {
        // Add auth token
        if (this.authToken && !config.headers.Authorization) {
          config.headers.Authorization = `Bearer ${this.authToken}`;
        }

        // Add tenant ID
        if (this.tenantId) {
          config.headers['X-Tenant-ID'] = this.tenantId;
        }

        // Add request ID for tracking
        config.headers['X-Request-ID'] = this.generateRequestId();

        // Log request in development
        if (this.config.enableLogging) {
          console.log('🚀 API Request:', {
            method: config.method?.toUpperCase(),
            url: config.url,
            data: config.data,
            headers: config.headers,
          });
        }

        return config;
      },
      (error) => {
        console.error('❌ Request Error:', error);
        return Promise.reject(error);
      }
    );

    // Response interceptor
    this.client.interceptors.response.use(
      (response) => {
        // Log response in development
        if (this.config.enableLogging) {
          console.log('✅ API Response:', {
            status: response.status,
            url: response.config.url,
            data: response.data,
          });
        }

        return response;
      },
      (error) => {
        const apiError = this.handleError(error);
        
        // Log error in development
        if (this.config.enableLogging) {
          console.error('❌ API Error:', {
            url: error.config?.url,
            status: error.response?.status,
            message: apiError.message,
            code: apiError.code,
          });
        }

        return Promise.reject(apiError);
      }
    );
  }

  private handleError(error: AxiosError): ApiClientError {
    if (error.response) {
      const response = error.response;
      const data = response.data as ApiResponse;
      
      switch (response.status) {
        case 400:
          return new ValidationError(
            data?.error?.message || 'Bad Request',
            data?.error?.details
          );
        case 401:
          return new AuthenticationError(data?.error?.message);
        case 403:
          return new AuthorizationError(data?.error?.message);
        case 404:
          return new NotFoundError(data?.error?.message);
        case 500:
        case 502:
        case 503:
        case 504:
          return new ServerError(data?.error?.message);
        default:
          return new ApiClientError(
            data?.error?.message || 'Request failed',
            data?.error?.code || 'UNKNOWN_ERROR',
            response.status,
            data?.error?.details
          );
      }
    } else if (error.request) {
      return new NetworkError('Network request failed');
    } else {
      return new ApiClientError(error.message, 'CLIENT_ERROR');
    }
  }

  private generateRequestId(): string {
    return `req_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }

  private async retryRequest<T>(
    requestFn: () => Promise<T>,
    retryConfig: RequestConfig['retry'] = { attempts: 3, delay: 1000, backoff: true }
  ): Promise<T> {
    let lastError: Error;
    
    for (let attempt = 1; attempt <= retryConfig.attempts; attempt++) {
      try {
        return await requestFn();
      } catch (error) {
        lastError = error as Error;
        
        // Don't retry on authentication/authorization errors
        if (error instanceof AuthenticationError || error instanceof AuthorizationError) {
          throw error;
        }
        
        if (attempt < retryConfig.attempts) {
          const delay = retryConfig.backoff 
            ? retryConfig.delay * Math.pow(2, attempt - 1)
            : retryConfig.delay;
          
          await new Promise(resolve => setTimeout(resolve, delay));
        }
      }
    }
    
    throw lastError!;
  }

  // Public methods
  public setAuthToken(token: string | null): void {
    this.authToken = token;
  }

  public setTenantId(tenantId: string | null): void {
    this.tenantId = tenantId;
  }

  public async get<T>(url: string, config: RequestConfig = {}): Promise<T> {
    const requestFn = () => this.client.get<ApiResponse<T>>(url, config);
    
    if (config.retry) {
      return this.retryRequest(async () => {
        const response = await requestFn();
        return this.processResponse(response);
      }, config.retry);
    }
    
    const response = await requestFn();
    return this.processResponse(response);
  }

  public async post<T, D = any>(url: string, data?: D, config: RequestConfig = {}): Promise<T> {
    const requestFn = () => this.client.post<ApiResponse<T>>(url, data, config);
    
    if (config.retry) {
      return this.retryRequest(async () => {
        const response = await requestFn();
        return this.processResponse(response);
      }, config.retry);
    }
    
    const response = await requestFn();
    return this.processResponse(response);
  }

  public async put<T, D = any>(url: string, data?: D, config: RequestConfig = {}): Promise<T> {
    const requestFn = () => this.client.put<ApiResponse<T>>(url, data, config);
    
    if (config.retry) {
      return this.retryRequest(async () => {
        const response = await requestFn();
        return this.processResponse(response);
      }, config.retry);
    }
    
    const response = await requestFn();
    return this.processResponse(response);
  }

  public async patch<T, D = any>(url: string, data?: D, config: RequestConfig = {}): Promise<T> {
    const response = await this.client.patch<ApiResponse<T>>(url, data, config);
    return this.processResponse(response);
  }

  public async delete<T>(url: string, config: RequestConfig = {}): Promise<T> {
    const response = await this.client.delete<ApiResponse<T>>(url, config);
    return this.processResponse(response);
  }

  private processResponse<T>(response: AxiosResponse<ApiResponse<T>>): T {
    const { data } = response;
    
    if (!data.success) {
      throw new ApiClientError(
        data.error?.message || 'Request failed',
        data.error?.code || 'UNKNOWN_ERROR',
        response.status,
        data.error?.details
      );
    }
    
    return data.data as T;
  }

  // Upload file method
  public async uploadFile<T>(
    url: string,
    file: File,
    onProgress?: (progress: number) => void,
    config: RequestConfig = {}
  ): Promise<T> {
    const formData = new FormData();
    formData.append('file', file);

    const requestConfig: AxiosRequestConfig = {
      ...config,
      headers: {
        ...config.headers,
        'Content-Type': 'multipart/form-data',
      },
      onUploadProgress: (progressEvent) => {
        if (onProgress && progressEvent.total) {
          const progress = Math.round((progressEvent.loaded * 100) / progressEvent.total);
          onProgress(progress);
        }
      },
    };

    const response = await this.client.post<ApiResponse<T>>(url, formData, requestConfig);
    return this.processResponse(response);
  }

  // Download file method
  public async downloadFile(
    url: string,
    filename?: string,
    onProgress?: (progress: number) => void
  ): Promise<void> {
    const response = await this.client.get(url, {
      responseType: 'blob',
      onDownloadProgress: (progressEvent) => {
        if (onProgress && progressEvent.total) {
          const progress = Math.round((progressEvent.loaded * 100) / progressEvent.total);
          onProgress(progress);
        }
      },
    });

    const blob = new Blob([response.data]);
    const downloadUrl = window.URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = downloadUrl;
    link.download = filename || 'download';
    document.body.appendChild(link);
    link.click();
    link.remove();
    window.URL.revokeObjectURL(downloadUrl);
  }

  // Health check method
  public async healthCheck(): Promise<{ status: string; timestamp: string }> {
    return this.get('/health');
  }

  // Cancel all pending requests
  public cancelAllRequests(): void {
    // This would be implemented with axios cancel tokens in a real scenario
    console.warn('Cancel all requests is not implemented in this example');
  }
}

// Create singleton instance
export const apiClient = new ApiClient();

// Export convenience methods
export const api = {
  get: <T>(url: string, config?: RequestConfig) => apiClient.get<T>(url, config),
  post: <T, D = any>(url: string, data?: D, config?: RequestConfig) => apiClient.post<T, D>(url, data, config),
  put: <T, D = any>(url: string, data?: D, config?: RequestConfig) => apiClient.put<T, D>(url, data, config),
  patch: <T, D = any>(url: string, data?: D, config?: RequestConfig) => apiClient.patch<T, D>(url, data, config),
  delete: <T>(url: string, config?: RequestConfig) => apiClient.delete<T>(url, config),
  upload: <T>(url: string, file: File, onProgress?: (progress: number) => void) => 
    apiClient.uploadFile<T>(url, file, onProgress),
  download: (url: string, filename?: string, onProgress?: (progress: number) => void) => 
    apiClient.downloadFile(url, filename, onProgress),
  
  // Auth methods
  setAuth: (token: string | null) => apiClient.setAuthToken(token),
  setTenant: (tenantId: string | null) => apiClient.setTenantId(tenantId),
  
  // Health check
  health: () => apiClient.healthCheck(),
};

export default api;