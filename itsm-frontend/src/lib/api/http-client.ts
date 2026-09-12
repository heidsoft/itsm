import { API_BASE_URL } from '@/lib/api/api-config';
import { security } from '@/lib/security';
import { logger } from '@/lib/env';
import { getTenantId, getTenantCode, setTenantId as setContextTenantId, setTenantCode as setContextTenantCode, subscribe } from '@/lib/auth/tenant-context';

// 递归将对象的 key 从 snake_case 转换为 camelCase
const toCamelCase = (obj: unknown): unknown => {
  if (obj === null || obj === undefined) {
    return obj;
  }
  if (Array.isArray(obj)) {
    return obj.map(v => toCamelCase(v));
  } else if (obj && typeof obj === 'object' && obj.constructor === Object) {
    const typedObj = obj as Record<string, unknown>;
    return Object.keys(typedObj).reduce(
      (result, key) => {
        const camelKey = key.replace(/_([a-z])/g, g => g[1].toUpperCase());
        result[camelKey] = toCamelCase(typedObj[key]);
        return result;
      },
      {} as Record<string, unknown>
    );
  }
  return obj;
};

// Request configuration interface
export interface RequestConfig {
  method?: 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH';
  headers?: Record<string, string>;
  body?: BodyInit | null;
  timeout?: number;
  responseType?: 'json' | 'blob';
}

// Axios-like request config used by some legacy API modules
export interface AxiosLikeRequestConfig {
  method?: 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH';
  url: string;
  params?: object;
  data?: unknown;
  headers?: Record<string, string>;
  responseType?: 'json' | 'blob';
}

// API response interface
interface ApiResponse<T> {
  code: number;
  message: string;
  data: T;
}

class HttpClient {
  private baseURL: string;
  private token: string | null = null;
  private readonly timeout: number;

  constructor(baseURL: string = API_BASE_URL) {
    this.baseURL =
      typeof window === 'undefined' ? process.env.ITSM_BACKEND_URL || baseURL : baseURL;
    this.timeout = parseInt(process.env.NEXT_PUBLIC_API_TIMEOUT || '30000');
	// Browser authentication uses HttpOnly cookies. JavaScript deliberately never
	// reads the token value; non-browser callers may still set this field.
  }

  setToken(token: string) {
    this.token = token;
    // Token is stored in httpOnly cookie by backend, no need to set here
    // This method kept for backward compatibility
  }

  clearToken() {
    this.token = null;
  }

  setTenantId(tenantId: number | null) {
    // Tenant state is now managed by TenantContext — kept for backward compat
    // New code should use tenant-context.ts directly
    setContextTenantId(tenantId);
  }

  setTenantCode(code: string | null) {
    setContextTenantCode(code);
  }

  // Get tenant code — now reads from TenantContext (single source of truth)
  getTenantCode(): string | null {
    return getTenantCode();
  }

  getTenantId(): number | null {
    return getTenantId();
  }

  getAuthToken(): string | null {
    return this.token;
  }

  // Backward-compat helpers (some legacy code expects these)
  getToken(): string | null {
    return this.getAuthToken();
  }

  getBaseURL(): string {
    return this.baseURL;
  }

  private getHeaders(): Record<string, string> {
    // Set secure request headers
    const headers: Record<string, string> = {
      ...security.network.getSecureHeaders(),
    };

    // Browser requests authenticate with HttpOnly cookies. Only non-browser
    // callers that explicitly set a token use the Authorization header.
    const currentToken = typeof window === 'undefined' ? this.token : null;
    // Tenant state — read from TenantContext (single source of truth)
    const currentTenantId = getTenantId();
    const currentTenantCode = getTenantCode();

    if (currentToken) {
      headers['Authorization'] = `Bearer ${currentToken}`;
    }

    if (currentTenantId) {
      headers['X-Tenant-ID'] = currentTenantId.toString();
    }
    if (currentTenantCode) {
      headers['X-Tenant-Code'] = currentTenantCode;
    }

    return headers;
  }

  // 获取CSRF token（用于mutating请求）
  // 直接在 httpClient 中维护 token 缓存，避免跨模块实例化导致的状态不同步
  private csrfTokenCache: string | null = null;
  private csrfTokenPromise: Promise<string | null> | null = null;

  private async getCSRFToken(): Promise<string | null> {
    if (this.csrfTokenCache) {
      return this.csrfTokenCache;
    }
    if (this.csrfTokenPromise) {
      return this.csrfTokenPromise;
    }
    this.csrfTokenPromise = security.csrf.getToken().then(token => {
      this.csrfTokenCache = token;
      return token;
    }).catch(error => {
      console.warn('[HttpClient] getCSRFToken error:', error);
      return null;
    });
    this.csrfTokenPromise.finally(() => {
      this.csrfTokenPromise = null;
    });
    return this.csrfTokenPromise;
  }

  /** Public accessor for legacy API classes that use raw fetch */
  async getCSRFTokenForExternal(): Promise<string | null> {
    return this.getCSRFToken();
  }

  /**
   * External API classes that use raw fetch (instead of `requestInternal`) do not
   * trigger the post-mutation CSRF cache invalidation in `requestInternal`.
   * Backend rotates the CSRF cookie after every successful write; if the cache
   * keeps the old token, the next mutation races with the new cookie and returns
   * 403 "CSRF token mismatch". Call this after any external mutation succeeds.
   */
  invalidateCSRFToken(): void {
    this.csrfTokenCache = null;
    this.csrfTokenPromise = null;
    security.csrf.clearToken();
  }

  /** True when the method requires CSRF token in the request header. */
  isMutatingMethodPublic(method?: string): boolean {
    return this.isMutatingMethod(method);
  }

  // 为mutating请求添加CSRF header
  private async addCSRFHeader(
    headers: Record<string, string>,
    method: string
  ): Promise<Record<string, string>> {
    // 只对mutating请求添加CSRF token
    if (['POST', 'PUT', 'DELETE', 'PATCH'].includes(method)) {
      const csrfToken = await this.getCSRFToken();
      if (csrfToken) {
        return {
          ...headers,
          'X-CSRF-Token': csrfToken,
        };
      }
    }
    return headers;
  }

  private isMutatingMethod(method?: string): boolean {
    return ['POST', 'PUT', 'DELETE', 'PATCH'].includes(method || 'GET');
  }

  private async isCSRFRejection(response?: Response): Promise<boolean> {
    if (!response || response.status !== 403) return false;
    try {
      const payload = (await response.clone().json()) as { message?: string };
      return payload.message?.startsWith('CSRF token') === true;
    } catch {
      return false;
    }
  }

  // FormData 上传必须由浏览器生成带 boundary 的 multipart Content-Type，因此合并后
  // 要丢弃 Content-Type：既包括 getHeaders() 自带的 application/json，也包括调用方
  // 传入的无 boundary 的 'multipart/form-data'（后者会让后端无法解析 multipart 体）。
  private mergeRequestHeaders(
    base: Record<string, string>,
    extra?: Record<string, string>,
    isFormData = false
  ): Record<string, string> {
    const merged = { ...base, ...extra };
    if (isFormData) {
      delete merged['Content-Type'];
    }
    return merged;
  }

  // Independent token refresh method to avoid circular dependencies
  private async refreshTokenInternal(): Promise<boolean> {
    try {
      const response = await fetch(`${this.baseURL}/api/v1/refresh-token`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include', // Include httpOnly cookies
		body: '{}',
      });

      if (response?.ok) {
        const data = await response.json();
        if (data.code === 0) {
          // Token is stored in httpOnly cookie by backend
          // Nothing to update in localStorage
          return true;
        }
      }
      return false;
    } catch (error) {
      logger.error('Token refresh failed:', error);
      return false;
    }
  }

  /**
   * 主动刷新会话（access_token 有效期 15 分钟）。
   * 由布局层定时调用，避免 token 过期后才被动触发刷新。
   * 刷新失败不抛错：下一次请求的 401 处理流程会兜底。
   */
  async refreshToken(): Promise<boolean> {
    try {
      return await this.refreshTokenInternal();
    } catch {
      return false;
    }
  }

  // Core request method using fetch API (internal)
  private async requestInternal<T>(endpoint: string, config: RequestConfig): Promise<T> {
    const url = `${this.baseURL}${endpoint}`;
    const responseType = config.responseType || 'json';
    const isFormData = typeof FormData !== 'undefined' && config.body instanceof FormData;
    let headers = this.getHeaders();
    // 为mutating请求添加CSRF token
    headers = await this.addCSRFHeader(headers, config.method || 'GET');
    const sanitizedHeaders = { ...headers };
    if (sanitizedHeaders.Authorization) {
      sanitizedHeaders.Authorization = '[REDACTED]';
    }
    const requestConfig: RequestInit = {
      method: config.method,
      headers: this.mergeRequestHeaders(headers, config.headers, isFormData),
      // Normalize request body keys to camelCase to keep the HTTP contract consistent.
      body:
        config.body && typeof config.body === 'string' && config.body.startsWith('{')
          ? JSON.stringify(toCamelCase(JSON.parse(config.body as string)))
          : config.body,
    };

    logger.debug('HTTP Client Request:', {
      url,
      method: config.method,
      headers: sanitizedHeaders,
      body: config.body,
    });

    // 在开发模式下，如果后端服务不可用，使用模拟数据
    if (process.env.NODE_ENV === 'development' && this.baseURL.includes('localhost')) {
      logger.warn('开发模式：正在连接到后端服务，如果后端服务未运行，将显示错误');
    }

    try {
      const controller = new AbortController();
      const timeoutMs = config.timeout ?? this.timeout;
      const timeoutId = setTimeout(() => controller.abort(), timeoutMs);

      let response = await fetch(url, {
        ...requestConfig,
        credentials: 'include', // Include httpOnly cookies for authenticated requests
        signal: controller.signal,
      });

      clearTimeout(timeoutId);

      // The backend rotates the token after every successful mutation. A cached token
      // can therefore race with the CSRF cookie; refresh it once and retry the request.
      if (
        this.isMutatingMethod(config.method) &&
        (await this.isCSRFRejection(response))
      ) {
        this.csrfTokenCache = null;
        security.csrf.clearToken();
        const retryHeaders = await this.addCSRFHeader(this.getHeaders(), config.method || 'GET');
        response = await fetch(url, {
          ...requestConfig,
          credentials: 'include',
          headers: this.mergeRequestHeaders(retryHeaders, config.headers, isFormData),
        });
      }

      if (response?.ok && this.isMutatingMethod(config.method)) {
        this.csrfTokenCache = null;
        security.csrf.clearToken();
      }

      logger.debug('HTTP Client Response:', {
        status: response?.status,
        statusText: response?.statusText,
        headers: response?.headers ? Object.fromEntries(response.headers.entries()) : {},
      });

      // If 401 error, try to refresh token
      if (response?.status === 401) {
        const refreshSuccess = await this.refreshTokenInternal();
        if (refreshSuccess) {
          // Retry original request with credentials: 'include' to send cookies
          const retryHeaders = await this.addCSRFHeader(this.getHeaders(), config.method || 'GET');
          const retryConfig: RequestInit = {
            ...requestConfig,
            credentials: 'include',
            headers: this.mergeRequestHeaders(retryHeaders, config.headers, isFormData),
          };
          const retryResponse = await fetch(url, retryConfig);
          if (!retryResponse.ok) {
            const rid = retryResponse.headers.get('X-Request-Id') || '';
            const suffix = rid ? ` [RID: ${rid}]` : '';
            throw new Error(`HTTP error! status: ${retryResponse?.status}${suffix}`);
          }

          if (responseType === 'blob') {
            return (await retryResponse.blob()) as unknown as T;
          }

          const retryData = (await retryResponse.json()) as ApiResponse<T>;
          logger.debug('HTTP Client Retry Response Data:', retryData);

          // Check response code - 容忍后端没有返回 code 字段的情况
          if (retryData.code !== undefined && retryData.code !== null && retryData.code !== 0) {
            const rid = retryResponse.headers.get('X-Request-Id') || '';
            const suffix = rid ? ` [RID: ${rid}]` : '';
            throw new Error((retryData.message || 'Request failed') + suffix);
          }

          return toCamelCase(retryData.data) as T;
        } else {
          // Refresh failed, clear token and redirect to login
          this.clearToken();
          if (typeof window !== 'undefined') {
            // Only redirect if not already on login page to avoid loops
            if (!window.location.pathname.startsWith('/login')) {
              window.location.href = `/login?redirect=${encodeURIComponent(window.location.pathname)}`;
            }
          }
          throw new Error('Authentication failed');
        }
      }

      if (!response?.ok) {
        const rid = response?.headers?.get('X-Request-Id') || '';
        const suffix = rid ? ` [RID: ${rid}]` : '';
        // 尝试从响应中获取更详细的错误信息
        try {
          const errorData = await response.json();
          if (errorData && errorData.message) {
            throw new Error(errorData.message + suffix);
          }
        } catch {
          // Ignore JSON parse errors, use default message
        }
        throw new Error(`HTTP error! status: ${response?.status}${suffix}`);
      }

      // Blob 响应：直接返回二进制，跳过 JSON 解析与 code 校验
      if (responseType === 'blob') {
        logger.debug('HTTP Client Blob Response:', { status: response?.status });
        return (await response.blob()) as unknown as T;
      }

      const responseData = (await response.json()) as ApiResponse<T>;
      logger.debug('HTTP Client Raw Response Data:', responseData);

      // Check response code - defense-in-depth，容忍历史未使用 common.Success 的 controller。
      // 允许 code 为 0、undefined、null 或不存在
      if (
        responseData.code !== undefined &&
        responseData.code !== null &&
        responseData.code !== 0
      ) {
        const rid = (response.headers && response.headers.get('X-Request-Id')) || '';
        const suffix = rid ? ` [RID: ${rid}]` : '';
        throw new Error((responseData.message || 'Request failed') + suffix);
      }

      // 自动转换响应数据 key 为 camelCase
      return toCamelCase(responseData.data) as T;
    } catch (error: unknown) {
      // AbortError 是正常的中止请求，不记录错误日志
      if (error instanceof Error && error.name === 'AbortError') {
        logger.debug('Request aborted (page navigation)');
        return null as T;
      }
      logger.error('Request failed:', error);
      if (error instanceof Error) {
        if (error.message.includes('fetch')) {
          throw new Error('无法连接到服务器，请检查网络连接和后端服务是否运行');
        }
        throw error;
      }
      throw new Error('未知错误发生');
    }
  }

  /**
   * Unified request method.
   * Supports:
   * - request('/path', { method, headers, body })
   * - request({ method, url, data, headers, responseType })
   */
  async request<T>(endpoint: string, config?: Partial<RequestConfig>): Promise<T>;
  async request<T>(config: AxiosLikeRequestConfig): Promise<T>;
  async request<T>(
    arg1: string | AxiosLikeRequestConfig,
    arg2?: Partial<RequestConfig>
  ): Promise<T> {
    // Axios-like style: request({ url, method, data, headers, responseType })
    if (typeof arg1 === 'object' && arg1 && 'url' in arg1) {
      const cfg = arg1 as AxiosLikeRequestConfig;
      const method = cfg.method || 'GET';
      let endpoint = cfg.url;
      if (cfg.params) {
        const qs = new URLSearchParams();
        Object.entries(cfg.params as Record<string, unknown>).forEach(([k, v]) => {
          if (v !== undefined && v !== null) qs.append(k, String(v));
        });
        const q = qs.toString();
        if (q) endpoint += (endpoint.includes('?') ? '&' : '?') + q;
      }

      let body: BodyInit | null | undefined = undefined;
      if (cfg.data !== undefined) {
        if (cfg.data instanceof FormData) body = cfg.data;
        else body = JSON.stringify(cfg.data);
      }

      // blob 路径走 requestInternal 统一处理 CSRF / credentials / tenant header / retry
      const data = await this.requestInternal<T>(endpoint, {
        method,
        headers: cfg.headers,
        body,
        responseType: cfg.responseType || 'json',
      });

      return data as T;
    }

    // Fetch-like style: request('/path', { ... })
    const endpoint = arg1 as string;
    const cfg = arg2 || {};
    return this.requestInternal<T>(endpoint, {
      method: cfg.method || 'GET',
      headers: cfg.headers,
      body: cfg.body,
      timeout: cfg.timeout,
      responseType: cfg.responseType || 'json',
    });
  }

  async get<T>(endpoint: string, params?: object): Promise<T> {
    let url = endpoint;
    if (params) {
      const searchParams = new URLSearchParams();
      Object.entries(params as Record<string, unknown>).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          searchParams.append(key, String(value));
        }
      });
      url += `?${searchParams.toString()}`;
    }

    return this.requestInternal<T>(url, {
      method: 'GET',
    });
  }

  private isCSRFRejectionText(responseText: string): boolean {
    try {
      const payload = JSON.parse(responseText) as { message?: string };
      return payload.message?.startsWith('CSRF token') === true;
    } catch {
      return false;
    }
  }

  // 带上传进度的 FormData 上传只能走 XHR，无法复用 requestInternal，因此这里显式对齐
  // 同一套语义：X-CSRF-Token、withCredentials（httpOnly cookie）、tenant header、
  // CSRF 轮换后重试一次、code 校验与 camelCase 转换。
  private postFormDataWithProgress<T>(
    url: string,
    data: FormData,
    onProgress: (progress: number) => void
  ): Promise<T> {
    const send = (attempt: number): Promise<T> =>
      this.addCSRFHeader(this.getHeaders(), 'POST').then((headers) => {
        const requestHeaders = this.mergeRequestHeaders(headers, undefined, true);

        return new Promise<T>((resolve, reject) => {
          const xhr = new XMLHttpRequest();
          // 浏览器端认证由 httpOnly cookie 承载，XHR 必须显式开启凭证
          xhr.withCredentials = true;

          xhr.upload.addEventListener('progress', event => {
            if (event.lengthComputable) {
              onProgress(Math.round((event.loaded * 100) / event.total));
            }
          });

          xhr.addEventListener('load', () => {
            // 后端每次成功变更后轮换 token，缓存值可能与 cookie 失配；清缓存后重试一次
            if (xhr.status === 403 && attempt === 0 && this.isCSRFRejectionText(xhr.responseText)) {
              this.csrfTokenCache = null;
              security.csrf.clearToken();
              send(attempt + 1).then(resolve, reject);
              return;
            }

            if (xhr.status < 200 || xhr.status >= 300) {
              reject(new Error(`HTTP error! status: ${xhr.status}`));
              return;
            }

            let response: ApiResponse<T>;
            try {
              response = JSON.parse(xhr.responseText) as ApiResponse<T>;
            } catch {
              reject(new Error('Failed to parse response'));
              return;
            }

            // 容忍后端没有返回 code 字段的情况
            if (response.code !== undefined && response.code !== null && response.code !== 0) {
              reject(new Error(response.message || 'Request failed'));
              return;
            }

            this.csrfTokenCache = null;
            security.csrf.clearToken();
            resolve(toCamelCase(response.data) as T);
          });

          xhr.addEventListener('error', () => reject(new Error('Network error')));

          xhr.open('POST', url);
          Object.entries(requestHeaders).forEach(([key, value]) => {
            xhr.setRequestHeader(key, value);
          });
          xhr.send(data);
        });
      });

    return send(0);
  }

  async post<T>(
    endpoint: string,
    data?: unknown,
    config?: {
      onUploadProgress?: (progress: number) => void;
      headers?: Record<string, string>;
      responseType?: 'json' | 'blob';
    }
  ): Promise<T> {
    // FormData 上传：无进度需求时统一交给 requestInternal，复用 CSRF / credentials /
    // tenant header / 401 刷新 / CSRF 轮换重试 / camelCase 转换。
    if (data instanceof FormData) {
      if (config?.onUploadProgress) {
        return this.postFormDataWithProgress<T>(
          `${this.baseURL}${endpoint}`,
          data,
          config.onUploadProgress
        );
      }

      return this.requestInternal<T>(endpoint, {
        method: 'POST',
        body: data,
        headers: config?.headers,
        responseType: config?.responseType || 'json',
      });
    }

    return this.requestInternal<T>(endpoint, {
      method: 'POST',
      body: data ? JSON.stringify(data) : undefined,
      headers: config?.headers,
    });
  }

  async put<T>(endpoint: string, data?: unknown): Promise<T> {
    return this.requestInternal<T>(endpoint, {
      method: 'PUT',
      body: data ? JSON.stringify(data) : undefined,
    });
  }

  async patch<T>(endpoint: string, data?: unknown): Promise<T> {
    return this.requestInternal<T>(endpoint, {
      method: 'PATCH',
      body: data ? JSON.stringify(data) : undefined,
    });
  }

  async delete<T>(endpoint: string, data?: unknown): Promise<T> {
    return this.requestInternal<T>(endpoint, {
      method: 'DELETE',
      body: data ? JSON.stringify(data) : undefined,
    });
  }

  // 分页查询方法
  async getPaginated<T>(
    endpoint: string,
    params?: {
      page?: number;
      pageSize?: number;
      sortBy?: string;
      sortOrder?: 'asc' | 'desc';
      filters?: Record<string, unknown>;
    }
  ): Promise<{
    data: T[];
    total: number;
    page: number;
    pageSize: number;
    totalPages: number;
  }> {
    const queryParams = new URLSearchParams();
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          if (key === 'filters' && typeof value === 'object') {
            Object.entries(value).forEach(([filterKey, filterValue]) => {
              if (filterValue !== undefined && filterValue !== null) {
                queryParams.append(`filters[${filterKey}]`, String(filterValue));
              }
            });
          } else {
            queryParams.append(key, String(value));
          }
        }
      });
    }

    const url = queryParams.toString() ? `${endpoint}?${queryParams.toString()}` : endpoint;

    return this.get(url);
  }

  // 批量操作方法
  async batchOperation<T>(endpoint: string, operation: string, data: unknown[]): Promise<T[]> {
    return this.post<T[]>(endpoint, {
      operation,
      data,
    });
  }

  // 文件上传方法
  async uploadFile(
    endpoint: string,
    file: File,
    onProgress?: (progress: number) => void
  ): Promise<{
    url: string;
    filename: string;
    size: number;
  }> {
    const formData = new FormData();
    formData.append('file', file);

    const uploadUrl = endpoint.startsWith('http') ? endpoint : `${this.baseURL}${endpoint}`;
    return this.postFormDataWithProgress(uploadUrl, formData, onProgress ?? (() => {}));
  }
}

export const httpClient = new HttpClient();
export default HttpClient;
