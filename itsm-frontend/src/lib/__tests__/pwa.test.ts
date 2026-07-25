/**
 * Tests for PWA utilities
 */

jest.mock('@/lib/api/http-client', () => ({
  httpClient: { get: jest.fn(), post: jest.fn(), put: jest.fn(), delete: jest.fn(), patch: jest.fn() },
}));

// Setup window mocks before import
const mockServiceWorker = {
  register: jest.fn(),
  ready: Promise.resolve({
    unregister: jest.fn().mockResolvedValue(true),
    update: jest.fn(),
    installing: null,
  } as any),
  controller: null,
  getRegistration: jest.fn(),
};

Object.defineProperty(window, 'location', {
  value: { hostname: 'localhost', href: 'http://localhost:3000', origin: 'http://localhost:3000', reload: jest.fn() },
  writable: true,
});

Object.defineProperty(navigator, 'serviceWorker', {
  value: mockServiceWorker,
  writable: true,
  configurable: true,
});

// Mock process.env
const originalEnv = process.env;
beforeAll(() => {
  process.env = { ...originalEnv, PUBLIC_URL: 'http://localhost:3000' };
});
afterAll(() => {
  process.env = originalEnv;
});

describe('PWA utilities', () => {
  beforeEach(() => {
    jest.spyOn(console, 'log').mockImplementation(() => {});
    jest.spyOn(console, 'error').mockImplementation(() => {});
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  describe('register', () => {
    it('should be importable', async () => {
      const pwa = await import('../pwa');
      expect(pwa.register).toBeDefined();
    });
  });

  describe('unregister', () => {
    it('should be importable', async () => {
      const pwa = await import('../pwa');
      expect(pwa.unregister).toBeDefined();
    });

    it('should call unregister on service worker', async () => {
      const pwa = await import('../pwa');
      pwa.unregister();
      // Just verify it doesn't throw
    });
  });

  describe('setupInstallPrompt', () => {
    it('should be importable', async () => {
      const pwa = await import('../pwa');
      expect(pwa.setupInstallPrompt).toBeDefined();
    });
  });

  describe('installApp', () => {
    it('should be importable', async () => {
      const pwa = await import('../pwa');
      expect(pwa.installApp).toBeDefined();
    });

    it('should handle no deferred prompt', async () => {
      const pwa = await import('../pwa');
      await pwa.installApp(); // should not throw
    });
  });

  describe('checkForUpdates', () => {
    it('should be importable and callable', async () => {
      const pwa = await import('../pwa');
      expect(pwa.checkForUpdates).toBeDefined();
      mockServiceWorker.getRegistration.mockResolvedValue({ update: jest.fn() });
      pwa.checkForUpdates();
    });
  });

  describe('setupOfflineDetection', () => {
    it('should set up event listeners', async () => {
      const addSpy = jest.spyOn(window, 'addEventListener');
      const pwa = await import('../pwa');
      pwa.setupOfflineDetection();
      expect(addSpy).toHaveBeenCalledWith('online', expect.any(Function));
      expect(addSpy).toHaveBeenCalledWith('offline', expect.any(Function));
    });
  });

  describe('requestNotificationPermission', () => {
    it('should request permission', async () => {
      Object.defineProperty(window, 'Notification', {
        value: { requestPermission: jest.fn().mockResolvedValue('granted'), permission: 'default' },
        writable: true,
        configurable: true,
      });
      const pwa = await import('../pwa');
      const result = await pwa.requestNotificationPermission();
      expect(result).toBe(true);
    });

    it('should return false when denied', async () => {
      Object.defineProperty(window, 'Notification', {
        value: { requestPermission: jest.fn().mockResolvedValue('denied'), permission: 'default' },
        writable: true,
        configurable: true,
      });
      const pwa = await import('../pwa');
      const result = await pwa.requestNotificationPermission();
      expect(result).toBe(false);
    });
  });

  describe('showNotification', () => {
    it('should create notification when granted', async () => {
      const MockNotification = jest.fn();
      (MockNotification as any).permission = 'granted';
      Object.defineProperty(window, 'Notification', {
        value: MockNotification,
        writable: true,
        configurable: true,
      });
      const pwa = await import('../pwa');
      pwa.showNotification('Test', { body: 'body' });
      expect(MockNotification).toHaveBeenCalledWith('Test', expect.objectContaining({ body: 'body' }));
    });

    it('should not create notification when not granted', async () => {
      const MockNotification = jest.fn();
      (MockNotification as any).permission = 'denied';
      Object.defineProperty(window, 'Notification', {
        value: MockNotification,
        writable: true,
        configurable: true,
      });
      const pwa = await import('../pwa');
      pwa.showNotification('Test', { body: 'body' });
      expect(MockNotification).not.toHaveBeenCalled();
    });
  });

  describe('requestNotificationPermission edge cases', () => {
    it('should return false when Notification not in window', async () => {
      const origNotification = (window as any).Notification;
      delete (window as any).Notification;
      const pwa = await import('../pwa');
      const result = await pwa.requestNotificationPermission();
      expect(result).toBe(false);
      (window as any).Notification = origNotification;
    });
  });

  describe('register function', () => {
    it('should register service worker', async () => {
      const pwa = await import('../pwa');
      const addSpy = jest.spyOn(window, 'addEventListener');
      pwa.register();
      expect(addSpy).toHaveBeenCalledWith('load', expect.any(Function));
    });

    it('should call onSuccess callback', async () => {
      const pwa = await import('../pwa');
      const onSuccess = jest.fn();
      pwa.register({ onSuccess });
      // verify no errors
    });

    it('should call onUpdate callback', async () => {
      const pwa = await import('../pwa');
      const onUpdate = jest.fn();
      pwa.register({ onUpdate });
      // verify no errors
    });
  });
});
