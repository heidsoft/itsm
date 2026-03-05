import '@testing-library/jest-dom';

// Mock dayjs - required for Ant Design DatePicker
// Must mock both default and named exports
jest.mock('dayjs', () => {
  const mockDate = {
    format: () => '2024-01-01',
    isValid: () => true,
    isAfter: () => false,
    isBefore: () => false,
    isSame: () => false,
    add: () => mockDate,
    subtract: () => mockDate,
    startOf: () => mockDate,
    endOf: () => mockDate,
    diff: () => 0,
    valueOf: () => 1704067200000,
    toISOString: () => '2024-01-01T00:00:00.000Z',
    hour: () => 12,
    minute: () => 30,
    second: () => 45,
    year: () => 2024,
    month: () => 0,
    date: () => 1,
    day: () => 1,
    unix: () => 1704067200,
    toDate: () => new Date(),
    toArray: () => [2024, 0, 1, 12, 30, 45, 0],
    isLeapYear: () => false,
    clone: () => mockDate,
    set: () => mockDate,
  };

  const mockDayjs = jest.fn((date) => {
    if (!date) return mockDate;
    return mockDate;
  });

  // Mock dayjs instance methods
  mockDayjs.prototype.format = jest.fn(function(format) {
    return '2024-01-01';
  });
  mockDayjs.prototype.isValid = jest.fn(function() {
    return true;
  });
  mockDayjs.prototype.isAfter = jest.fn(function() {
    return false;
  });
  mockDayjs.prototype.isBefore = jest.fn(function() {
    return false;
  });
  mockDayjs.prototype.isSame = jest.fn(function() {
    return false;
  });
  mockDayjs.prototype.add = jest.fn(function() {
    return mockDate;
  });
  mockDayjs.prototype.subtract = jest.fn(function() {
    return mockDate;
  });
  mockDayjs.prototype.startOf = jest.fn(function() {
    return mockDate;
  });
  mockDayjs.prototype.endOf = jest.fn(function() {
    return mockDate;
  });
  mockDayjs.prototype.diff = jest.fn(function() {
    return 0;
  });
  mockDayjs.prototype.valueOf = jest.fn(function() {
    return 1704067200000;
  });
  mockDayjs.prototype.toISOString = jest.fn(function() {
    return '2024-01-01T00:00:00.000Z';
  });
  mockDayjs.prototype.hour = jest.fn(function() {
    return 12;
  });
  mockDayjs.prototype.minute = jest.fn(function() {
    return 30;
  });
  mockDayjs.prototype.second = jest.fn(function() {
    return 45;
  });
  mockDayjs.prototype.year = jest.fn(function() {
    return 2024;
  });
  mockDayjs.prototype.month = jest.fn(function() {
    return 0;
  });
  mockDayjs.prototype.date = jest.fn(function() {
    return 1;
  });
  mockDayjs.prototype.day = jest.fn(function() {
    return 1;
  });
  mockDayjs.prototype.unix = jest.fn(function() {
    return 1704067200;
  });
  mockDayjs.prototype.toDate = jest.fn(function() {
    return new Date();
  });
  mockDayjs.prototype.toArray = jest.fn(function() {
    return [2024, 0, 1, 12, 30, 45, 0];
  });
  mockDayjs.prototype.isLeapYear = jest.fn(function() {
    return false;
  });
  mockDayjs.prototype.clone = jest.fn(function() {
    return mockDate;
  });
  mockDayjs.prototype.set = jest.fn(function() {
    return mockDate;
  });

  // Mock static methods
  mockDayjs.extend = jest.fn();
  mockDayjs.locale = jest.fn();
  mockDayjs.unix = jest.fn(function() {
    return mockDate;
  });
  mockDayjs.isDayjs = jest.fn();

  return mockDayjs;
});

// Mock Next.js router
jest.mock('next/navigation', () => ({
  useRouter() {
    return {
      push: jest.fn(),
      replace: jest.fn(),
      prefetch: jest.fn(),
      back: jest.fn(),
      forward: jest.fn(),
      refresh: jest.fn(),
    };
  },
  useSearchParams() {
    return new URLSearchParams();
  },
  usePathname() {
    return '/';
  },
}));

// Mock window.matchMedia
Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: jest.fn().mockImplementation(query => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: jest.fn(), // deprecated
    removeListener: jest.fn(), // deprecated
    addEventListener: jest.fn(),
    removeEventListener: jest.fn(),
    dispatchEvent: jest.fn(),
  })),
});

// Mock IntersectionObserver
global.IntersectionObserver = class IntersectionObserver {
  constructor() {}
  observe() {
    return null;
  }
  disconnect() {
    return null;
  }
  unobserve() {
    return null;
  }
};

// Mock ResizeObserver
global.ResizeObserver = class ResizeObserver {
  constructor() {}
  observe() {
    return null;
  }
  disconnect() {
    return null;
  }
  unobserve() {
    return null;
  }
};

// Suppress console warnings in tests
const originalError = console.error;
beforeAll(() => {
  console.error = (...args) => {
    if (
      typeof args[0] === 'string' &&
      args[0].includes('Warning: ReactDOM.render is no longer supported')
    ) {
      return;
    }
    originalError.call(console, ...args);
  };
});

afterAll(() => {
  console.error = originalError;
});