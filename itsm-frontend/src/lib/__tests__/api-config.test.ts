/**
 * API配置测试
 */

import {
  API_BASE_URL,
  API_VERSION,
  API_TIMEOUT,
  API_ERROR_CODES,
  ApiResponse,
  PaginationRequest,
  PaginationResponse,
} from '../api/api-config';

describe('API Configuration', () => {
  describe('API_BASE_URL', () => {
    it('should be defined', () => {
      expect(API_BASE_URL).toBeDefined();
    });

    it('should be a string', () => {
      expect(typeof API_BASE_URL).toBe('string');
    });
  });

  describe('API_VERSION', () => {
    it('should be defined', () => {
      expect(API_VERSION).toBeDefined();
    });

    it('should be v1 by default', () => {
      expect(API_VERSION).toBe('v1');
    });
  });

  describe('API_TIMEOUT', () => {
    it('should be defined', () => {
      expect(API_TIMEOUT).toBeDefined();
    });

    it('should be a number', () => {
      expect(typeof API_TIMEOUT).toBe('number');
    });

    it('should be positive', () => {
      expect(API_TIMEOUT).toBeGreaterThan(0);
    });

    it('should be 30000 by default', () => {
      expect(API_TIMEOUT).toBe(30000);
    });
  });

  describe('API_ERROR_CODES', () => {
    it('should be defined', () => {
      expect(API_ERROR_CODES).toBeDefined();
    });

    it('should have SUCCESS code', () => {
      expect(API_ERROR_CODES.SUCCESS).toBe(0);
    });

    it('should have PARAM_ERROR code', () => {
      expect(API_ERROR_CODES.PARAM_ERROR).toBe(1001);
    });

    it('should have AUTH_FAILED code', () => {
      expect(API_ERROR_CODES.AUTH_FAILED).toBe(2001);
    });

    it('should have FORBIDDEN code', () => {
      expect(API_ERROR_CODES.FORBIDDEN).toBe(2003);
    });

    it('should have NOT_FOUND code', () => {
      expect(API_ERROR_CODES.NOT_FOUND).toBe(4004);
    });

    it('should have INTERNAL_ERROR code', () => {
      expect(API_ERROR_CODES.INTERNAL_ERROR).toBe(5001);
    });
  });
});

describe('ApiResponse Interface', () => {
  it('should accept valid response structure', () => {
    const response: ApiResponse<string> = {
      code: 0,
      message: 'success',
      data: 'test data',
    };

    expect(response.code).toBe(0);
    expect(response.message).toBe('success');
    expect(response.data).toBe('test data');
  });

  it('should accept error response structure', () => {
    const response: ApiResponse<null> = {
      code: 1001,
      message: 'Parameter error',
      data: null,
    };

    expect(response.code).toBe(1001);
    expect(response.message).toBe('Parameter error');
    expect(response.data).toBeNull();
  });
});

describe('PaginationRequest Interface', () => {
  it('should accept valid pagination request', () => {
    const request: PaginationRequest = {
      page: 1,
      pageSize: 10,
    };

    expect(request.page).toBe(1);
    expect(request.pageSize).toBe(10);
  });

  it('should accept undefined values', () => {
    const request: PaginationRequest = {};

    expect(request.page).toBeUndefined();
    expect(request.pageSize).toBeUndefined();
  });
});

describe('PaginationResponse Interface', () => {
  it('should accept valid pagination response', () => {
    const response: PaginationResponse<string> = {
      items: ['item1', 'item2', 'item3'],
      total: 100,
      page: 1,
      pageSize: 10,
      totalPages: 10,
    };

    expect(response.items).toHaveLength(3);
    expect(response.total).toBe(100);
    expect(response.page).toBe(1);
    expect(response.pageSize).toBe(10);
    expect(response.totalPages).toBe(10);
  });

  it('should handle empty items', () => {
    const response: PaginationResponse<string> = {
      items: [],
      total: 0,
      page: 1,
      pageSize: 10,
      totalPages: 0,
    };

    expect(response.items).toHaveLength(0);
    expect(response.total).toBe(0);
    expect(response.totalPages).toBe(0);
  });
});
