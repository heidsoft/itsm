import { defineConfig, devices } from '@playwright/test';

const baseURL = process.env.PLAYWRIGHT_BASE_URL || 'http://localhost:3000';

export default defineConfig({
  testDir: './tests/e2e',
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
      args: ['--disable-crashpad', '--disable-breakpad'],
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
    {
      name: 'chrome',
      use: { browserName: 'chromium', channel: 'chrome' },
    },
    {
      name: 'chromium',
      use: { browserName: 'chromium' },
    },
    {
      name: 'webkit',
      use: { ...devices['Desktop Safari'] },
    },
    // 业务流程测试专用配置
    {
      name: 'business-flows',
      use: {
        browserName: 'chromium',
        viewport: { width: 1440, height: 900 },
      },
      testDir: './tests/e2e/business-flows',
      timeout: 60_000,
    },
  ],
});
