/**
 * Unified API Client
 * Provides consistent API request handling across the application
 */

import { apiConfig } from './api-config';

const BASE_URL = apiConfig.baseURL;

// API Response type
export interface ApiResponse<T> {
  code: number;
  message: string;
  data: T;
}

// Request options
export interface RequestOptions extends RequestInit {
  requiresAuth?: boolean;
  timeout?: number;
}

// Token retrieval
function getToken(): string {
  if (typeof window !== 'undefined') {
    return localStorage.getItem('access_token') || localStorage.getItem('token') || '';
  }
  return '';
}

// Check if token is expired
function isTokenExpired(token: string): boolean {
  try {
    const parts = token.split('.');
    if (parts.length !== 3) return true;

    const payload = JSON.parse(atob(parts[1]));
    const currentTime = Math.floor(Date.now() / 1000);

    return payload.exp && payload.exp < currentTime;
  } catch {
    return true;
  }
}

// Base fetch with error handling
async function baseFetch<T>(
  endpoint: string,
  options: RequestOptions = {}
): Promise<T> {
  const {
    requiresAuth = true,
    timeout = 30000,
    headers = {},
    ...fetchOptions
  } = options;

  const url = `${BASE_URL}${endpoint}`;

  // Build headers
  const authHeaders: Record<string, string> = {
    'Content-Type': 'application/json',
  };

  if (requiresAuth) {
    const token = getToken();
    if (!token) {
      throw new Error('No authentication token found');
    }

    if (isTokenExpired(token)) {
      throw new Error('Token expired');
    }

    authHeaders['Authorization'] = `Bearer ${token}`;
  }

  const finalHeaders = { ...authHeaders, ...headers };

  // Create abort controller for timeout
  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), timeout);

  try {
    const response = await fetch(url, {
      ...fetchOptions,
      headers: finalHeaders,
      signal: controller.signal,
    });

    clearTimeout(timeoutId);

    // Handle non-OK responses
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.message || `HTTP ${response.status}: ${response.statusText}`);
    }

    // Parse response
    const responseData = await response.json() as ApiResponse<T>;

    // Check API response code
    if (responseData.code !== 0) {
      throw new Error(responseData.message || 'Request failed');
    }

    return responseData.data;
  } catch (error) {
    clearTimeout(timeoutId);

    if (error instanceof Error) {
      if (error.name === 'AbortError') {
        throw new Error('Request timeout');
      }
      throw error;
    }

    throw new Error('Unknown error occurred');
  }
}

// HTTP method helpers
export const apiClient = {
  get<T>(endpoint: string, options?: Omit<RequestOptions, 'method'>): Promise<T> {
    return baseFetch<T>(endpoint, { ...options, method: 'GET' });
  },

  post<T>(endpoint: string, body?: unknown, options?: Omit<RequestOptions, 'method' | 'body'>): Promise<T> {
    return baseFetch<T>(endpoint, {
      ...options,
      method: 'POST',
      body: body ? JSON.stringify(body) : undefined,
    });
  },

  put<T>(endpoint: string, body?: unknown, options?: Omit<RequestOptions, 'method' | 'body'>): Promise<T> {
    return baseFetch<T>(endpoint, {
      ...options,
      method: 'PUT',
      body: body ? JSON.stringify(body) : undefined,
    });
  },

  patch<T>(endpoint: string, body?: unknown, options?: Omit<RequestOptions, 'method' | 'body'>): Promise<T> {
    return baseFetch<T>(endpoint, {
      ...options,
      method: 'PATCH',
      body: body ? JSON.stringify(body) : undefined,
    });
  },

  delete<T>(endpoint: string, options?: Omit<RequestOptions, 'method'>): Promise<T> {
    return baseFetch<T>(endpoint, { ...options, method: 'DELETE' });
  },

  // Upload file
  upload<T>(endpoint: string, file: File, additionalData?: Record<string, string>): Promise<T> {
    const token = getToken();
    const formData = new FormData();
    formData.append('file', file);

    if (additionalData) {
      Object.entries(additionalData).forEach(([key, value]) => {
        formData.append(key, value);
      });
    }

    return baseFetch<T>(endpoint, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${token}`,
      },
      body: formData,
    });
  },

  // Download file
  async download(endpoint: string, filename: string): Promise<void> {
    const token = getToken();
    const response = await fetch(`${BASE_URL}${endpoint}`, {
      method: 'GET',
      headers: {
        'Authorization': `Bearer ${token}`,
      },
    });

    if (!response.ok) {
      throw new Error(`Download failed: ${response.statusText}`);
    }

    const blob = await response.blob();
    const downloadUrl = window.URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = downloadUrl;
    link.download = filename;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    window.URL.revokeObjectURL(downloadUrl);
  },
};

export default apiClient;
