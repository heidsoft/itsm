import nextJest from 'next/jest.js';

const createJestConfig = nextJest({
  // Provide the path to your Next.js app to load next.config.js and .env files
  dir: './',
});

// 覆盖率双口径（2026-10-17 决策）：
//   - 默认 = 阻断门槛口径：src/lib/**，80% —— 「业务逻辑契约」口径（纯函数/状态机/API 客户端），
//     能真正拦住回归。src/lib 只占产品代码约 30%，所以这个 80% 不是产品覆盖率。
//   - COVERAGE_SCOPE=product = 只报告口径：src/**，无 threshold —— 分母含 169 个页面与
//     257 个组件，数字会掉到 30% 上下；它的用途是「看得见 UI 覆盖有没有在动」，
//     不能当阻断门槛（否则只能把门槛下调到 30%，等于失去约束力）。
// 两个口径不可互相替代，ROADMAP 必须同时披露（Gate C.6.5 守卫）。
const productScope = process.env.COVERAGE_SCOPE === 'product';

// Add any custom config to be passed to Jest
const customJestConfig = {
  setupFilesAfterEnv: ['<rootDir>/jest.setup.js'],
  // 自定义环境:让 jsdom 样式级联对 nwsapi 无法解析的 `:has()` + Tailwind 任意值类
  // 选择器返回"不匹配"而非抛错(浏览器行为),详见 jest-environment-safe-nwsapi.js
  testEnvironment: '<rootDir>/jest-environment-safe-nwsapi.js',
  moduleNameMapper: {
    '^@/(.*)$': '<rootDir>/src/$1',
    '^lodash-es$': 'lodash',
  },
  transformIgnorePatterns: [
    'node_modules/(?!(lodash-es)/)',
  ],
  testMatch: [
    '<rootDir>/src/**/__tests__/**/*.{js,jsx,ts,tsx}',
    '<rootDir>/src/**/*.{test,spec}.{js,jsx,ts,tsx}',
  ],
  collectCoverage: true,
  collectCoverageFrom: productScope
    ? [
        'src/**/*.{ts,tsx}',
        '!src/**/__tests__/**',
        '!src/**/*.test.{ts,tsx}',
        '!src/**/*.spec.{ts,tsx}',
        '!src/**/*.d.ts',
      ]
    : [
        'src/lib/**/*.{ts,tsx}',
        '!src/lib/**/__tests__/**',
        '!src/lib/**/index.ts',
        '!src/lib/**/types.ts',
      ],
  // 产品口径只报告、不阻断：空对象 = 关闭全局阈值。
  coverageThreshold: productScope
    ? {}
    : {
        global: {
          // Keep the gate just below the measured baseline while the legacy UI
          // modules are incrementally covered; statements/functions/lines remain
          // at the 80% production threshold.
          branches: 64.5,
          functions: 80,
          lines: 80,
          statements: 80,
        },
      },
  coverageReporters: [
    'text',
    'text-summary',
    'html',
    'lcov',
    'json-summary',
  ],
  coverageDirectory: productScope ? 'coverage/product' : 'coverage',
  testTimeout: 10000,
  verbose: true,
  clearMocks: true,
  restoreMocks: true,
  resetModules: true,
  maxWorkers: '50%',
  cacheDirectory: '<rootDir>/.jest-cache',
  errorOnDeprecated: true,
  notify: false,
  notifyMode: 'failure-change',
  watchPlugins: [
    'jest-watch-typeahead/filename',
    'jest-watch-typeahead/testname',
  ],
  reporters: [
    'default',
    [
      'jest-junit',
      {
        outputDirectory: 'test-results',
        outputName: 'junit.xml',
        ancestorSeparator: ' › ',
        uniqueOutputName: 'false',
        suiteNameTemplate: '{filepath}',
        classNameTemplate: '{classname}',
        titleTemplate: '{title}',
      },
    ],
  ],
};

// createJestConfig is exported this way to ensure that next/jest can load the Next.js config which is async
export default createJestConfig(customJestConfig);
