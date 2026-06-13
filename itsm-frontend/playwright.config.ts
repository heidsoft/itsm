import { defineConfig, devices } from '@playwright/test';

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
  webServer: {
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
  ],
});
