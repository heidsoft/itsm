/**
 * Mock Authentication Service for Development/Testing
 * This module provides mock authentication functionality
 */

export interface MockUser {
  id: string;
  username: string;
  email: string;
  role: string;
  permissions: string[];
  avatar?: string;
}

export interface MockAuthService {
  // Mock login
  login: (username: string, password: string) => Promise<{
    success: boolean;
    user?: MockUser;
    token?: string;
    error?: string;
  }>;

  // Mock logout
  logout: () => Promise<void>;

  // Mock token validation
  validateToken: (token: string) => Promise<boolean>;

  // Mock user info
  getCurrentUser: () => MockUser | null;
}

// Default mock user for development
const DEFAULT_MOCK_USER: MockUser = {
  id: '1',
  username: 'admin',
  email: 'admin@example.com',
  role: 'admin',
  permissions: ['*'],
};

// Create mock auth service
export const createMockAuthService = (): MockAuthService => {
  let currentUser: MockUser | null = DEFAULT_MOCK_USER;
  let isAuthenticated = true;

  return {
    login: async (username: string, _password: string) => {
      // Simulate network delay
      await new Promise(resolve => setTimeout(resolve, 500));

      if (username === 'admin' || username === 'test') {
        currentUser = {
          ...DEFAULT_MOCK_USER,
          username,
        };
        isAuthenticated = true;
        return {
          success: true,
          user: currentUser,
          token: `mock-token-${Date.now()}`,
        };
      }

      return {
        success: false,
        error: 'Invalid credentials',
      };
    },

    logout: async () => {
      await new Promise(resolve => setTimeout(resolve, 200));
      currentUser = null;
      isAuthenticated = false;
    },

    validateToken: async (_token: string) => {
      await new Promise(resolve => setTimeout(resolve, 100));
      return isAuthenticated;
    },

    getCurrentUser: () => currentUser,
  };
};

// Export singleton instance
export const MockAuthService = createMockAuthService();

export default MockAuthService;
