import { defineConfig, devices } from '@playwright/test';

const isolatedStack = process.env.ITSM_E2E_ISOLATED_STACK === '1';
// 一次性栈模式下不允许有默认地址，也不允许隐式起/复用宿主 :3000：
// 那台 dev server 的代理 upstream 是 http://localhost:8090，本机该端口属于生产容器
// （DB_NAME=itsm_prod），夹具写入会直接落到生产库。地址只能来自
// scripts/e2e-isolated-stack.sh 导出的 PLAYWRIGHT_BASE_URL。
if (isolatedStack && !process.env.PLAYWRIGHT_BASE_URL) {
  throw new Error(
    'ITSM_E2E_ISOLATED_STACK=1 必须显式提供 PLAYWRIGHT_BASE_URL（由 scripts/e2e-isolated-stack.sh up 生成）'
  );
}
const baseURL = process.env.PLAYWRIGHT_BASE_URL || 'http://localhost:3000';
const enableBrowserChannels = process.env.PLAYWRIGHT_SKIP_CHANNELS !== '1';
const enableEdge = process.env.PLAYWRIGHT_ENABLE_EDGE === '1';

export default defineConfig({
  testDir: './tests/e2e',
  outputDir: process.env.PLAYWRIGHT_OUTPUT_DIR || '/tmp/itsm-playwright-results',
  timeout: 30_000, // 减少超时时间，快速失败
  expect: {
    timeout: 5_000,
  },
  use: {
    baseURL,
    trace: 'retain-on-failure', // 失败时保留trace
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
    actionTimeout: 10_000, // 单个操作超时
    launchOptions: {
      args: ['--disable-crashpad', '--disable-breakpad', '--allow-insecure-localhost'],
    },
  },
  reporter: [['list'], ['html', { open: 'never' }]],
  // 一次性栈里前端已在容器内跑 dev server，绝不能再由宿主起一个 :3000 顶掉它。
  webServer: isolatedStack
    ? undefined
    : {
        command: 'npm run dev',
        port: 3000,
        reuseExistingServer: true,
        timeout: 60_000, // 减少启动超时
      },
  projects: [
    ...(enableBrowserChannels
      ? [
          {
            name: 'chrome',
            use: { browserName: 'chromium' as const, channel: 'chrome' as const },
          },
        ]
      : []),
    ...(enableEdge
      ? [
          {
            name: 'edge',
            use: { browserName: 'chromium' as const, channel: 'msedge' as const },
          },
        ]
      : []),
    {
      name: 'firefox',
      use: { ...devices['Desktop Firefox'] },
    },
    {
      name: 'webkit',
      use: { ...devices['Desktop Safari'] },
    },
    {
      name: 'chromium',
      use: { browserName: 'chromium' as const },
    },
    {
      name: 'business-flows',
      use: {
        browserName: 'chromium' as const,
        viewport: { width: 1440, height: 900 },
      },
      testDir: './tests/e2e/business-flows',
      timeout: 60_000,
    },

    // PR-0.4: golden — 25 critical business flows. Default OFF locally so
    // day-to-day `npm run e2e` stays fast; CI enables it via PLAYWRIGHT_ENABLE_GOLDEN=1.
    ...(process.env.PLAYWRIGHT_ENABLE_GOLDEN === '1'
      ? [
          {
            name: 'golden',
            use: {
              browserName: 'chromium' as const,
              viewport: { width: 1440, height: 900 },
              // Golden runs want trace on every test, not just failures —
              // the GA gate uses them as the official pass/fail signal.
              trace: 'on' as const,
              video: 'retain-on-failure' as const,
            },
            testDir: './tests/e2e/golden',
            timeout: 60_000,
            grep: /@golden/,
          },
        ]
      : []),

    // PR-0.4 / PR-3.5: multi-tenant — cross-tenant isolation regression.
    // Tags tests @multi-tenant so PR-3.4 can wire them into the GA gate.
    ...(process.env.PLAYWRIGHT_ENABLE_MULTI_TENANT === '1'
      ? [
          {
            name: 'multi-tenant',
            use: {
              browserName: 'chromium' as const,
              viewport: { width: 1440, height: 900 },
            },
            testDir: './tests/e2e/multi-tenant',
            timeout: 60_000,
            grep: /@multi-tenant/,
          },
        ]
      : []),
  ],
});
