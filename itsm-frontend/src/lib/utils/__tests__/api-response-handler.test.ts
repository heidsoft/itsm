/**
 * api-response-handler - API 响应处理工具测试
 */

import {
  handleApiResponse,
  handleApiError,
  isApiError,
  parseResponse,
  mergeQueryParams,
  buildUrl,
  getQueryParam,
  setQueryParam,
  removeQueryParam,
  formatApiUrl,
  shouldRetry,
  getRetryDelay,
} from '../api-response-handler';

describe('api-response-handler', () => {
  describe('handleApiResponse', () => {
    it('应该成功处理响应数据', () => {
      const response = { data: { result: 'success' }, status: 200 };
      const result = handleApiResponse(response);
      expect(result).toEqual({ result: 'success' });
    });

    it('应该处理包含分页的响应', () => {
      const response = {
        data: { items: [1, 2, 3], total: 10, page: 1 },
        status: 200,
      };
      const result = handleApiResponse(response);
      expect(result.items).toHaveLength(3);
      expect(result.total).toBe(10);
    });

    it('应该处理空响应', () => {
      const response = { data: null, status: 204 };
      const result = handleApiResponse(response);
      expect(result).toBeNull();
    });

    it('应该处理非 200 状态码', () => {
      const response = { data: { error: 'Not Found' }, status: 404 };
      const result = handleApiResponse(response);
      expect(result).toHaveProperty('error', 'Not Found');
    });
  });

  describe('handleApiError', () => {
    it('应该处理网络错误', () => {
      const error = new Error('Network error');
      const result = handleApiError(error);
      expect(result).toHaveProperty('message', 'Network error');
      expect(result.code).toBe('NETWORK_ERROR');
    });

    it('应该处理超时错误', () => {
      const error = new Error('Timeout');
      const result = handleApiError(error, 'timeout');
      expect(result.code).toBe('TIMEOUT');
    });
  });

  describe('isApiError', () => {
    it('应该识别 API 错误对象', () => {
      const apiError = { message: 'Error', code: 400, details: {} };
      expect(isApiError(apiError)).toBe(true);
    });

    it('应该拒绝普通错误', () => {
      const error = new Error('Normal error');
      expect(isApiError(error)).toBe(false);
    });
  });

  describe('parseResponse', () => {
    it('应该解析 JSON 响应', () => {
      const json = '{"key": "value"}';
      const result = parseResponse(json);
      expect(result).toEqual({ key: 'value' });
    });

    it('应该处理无效 JSON', () => {
      const invalidJson = 'invalid';
      const result = parseResponse(invalidJson);
      expect(result).toBeNull();
    });

    it('应该处理空响应', () => {
      expect(parseResponse(null)).toBeNull();
      expect(parseResponse(undefined)).toBeNull();
    });
  });

  describe('mergeQueryParams', () => {
    it('应该合并查询参数', () => {
      const existing = '?page=1&limit=10';
      const additional = { sort: 'name', filter: 'active' };
      const result = mergeQueryParams(existing, additional);
      expect(result).toContain('page=1');
      expect(result).toContain('limit=10');
      expect(result).toContain('sort=name');
      expect(result).toContain('filter=active');
    });

    it('应该覆盖现有参数', () => {
      const existing = '?page=1&limit=10';
      const additional = { limit: 20 };
      const result = mergeQueryParams(existing, additional);
      expect(result).toContain('limit=20');
    });
  });

  describe('buildUrl', () => {
    it('应该构建完整 URL', () => {
      const result = buildUrl('https://api.example.com', '/endpoint', { param: 'value' });
      expect(result).toBe('https://api.example.com/endpoint?param=value');
    });

    it('应该处理基础 URL 结尾斜杠', () => {
      const result = buildUrl('https://api.example.com/', '/endpoint', {});
      expect(result).toBe('https://api.example.com/endpoint');
    });

    it('应该处理端点开头斜杠', () => {
      const result = buildUrl('https://api.example.com', 'endpoint', {});
      expect(result).toBe('https://api.example.com/endpoint');
    });
  });

  describe('getQueryParam', () => {
    it('应该获取查询参数', () => {
      const url = 'https://example.com/path?foo=bar&baz=qux';
      expect(getQueryParam(url, 'foo')).toBe('bar');
      expect(getQueryParam(url, 'baz')).toBe('qux');
    });

    it('应该返回默认值', () => {
      const url = 'https://example.com/path?foo=bar';
      expect(getQueryParam(url, 'missing', 'default')).toBe('default');
    });

    it('应该处理重复参数', () => {
      const url = 'https://example.com/path?foo=bar&foo=baz';
      expect(getQueryParam(url, 'foo')).toBe('bar'); // 返回第一个
    });
  });

  describe('setQueryParam', () => {
    it('应该设置查询参数', () => {
      const url = 'https://example.com/path';
      const result = setQueryParam(url, 'page', '2');
      expect(result).toContain('page=2');
    });

    it('应该替换已有参数', () => {
      const url = 'https://example.com/path?page=1';
      const result = setQueryParam(url, 'page', '2');
      expect(result).toContain('page=2');
      expect(result).not.toContain('page=1');
    });
  });

  describe('removeQueryParam', () => {
    it('应该移除查询参数', () => {
      const url = 'https://example.com/path?page=1&limit=10';
      const result = removeQueryParam(url, 'page');
      expect(result).not.toContain('page=1');
      expect(result).toContain('limit=10');
    });

    it('移除不存在的参数应无变化', () => {
      const url = 'https://example.com/path?page=1';
      const result = removeQueryParam(url, 'missing');
      expect(result).toBe(url);
    });
  });

  describe('formatApiUrl', () => {
    it('应该格式化 API URL', () => {
      expect(formatApiUrl('/api/users')).toBe('/api/users');
      expect(formatApiUrl('users')).toBe('/api/users'); // 假设有默认前缀
    });

    it('应该处理绝对 URL', () => {
      expect(formatApiUrl('https://external.com/api')).toBe('https://external.com/api');
    });
  });

  describe('shouldRetry', () => {
    it('应该根据状态码决定重试', () => {
      expect(shouldRetry(502)).toBe(true); // Bad Gateway
      expect(shouldRetry(503)).toBe(true); // Service Unavailable
      expect(shouldRetry(504)).toBe(true); // Gateway Timeout
      expect(shouldRetry(400)).toBe(false); // Bad Request
      expect(shouldRetry(200)).toBe(false); // OK
    });

    it('应该处理网络错误', () => {
      expect(shouldRetry('network_error')).toBe(true);
    });
  });

  describe('getRetryDelay', () => {
    it('应该计算重试延迟', () => {
      expect(getRetryDelay(0)).toBe(1000);
      expect(getRetryDelay(1)).toBe(2000);
      expect(getRetryDelay(2)).toBe(4000);
    });

    it('应该设置最大延迟', () => {
      expect(getRetryDelay(10)).toBe(30000); // 最大 30 秒
    });
  });
});
