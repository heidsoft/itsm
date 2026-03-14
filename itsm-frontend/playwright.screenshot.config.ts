import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './tests/e2e',
  timeout: 60_000,
  use: {
    baseURL: 'http://localhost:3000',
    screenshot: 'off',
    trace: 'off',
  },
  reporter: [['list']],
  projects: [
    {
      name: 'screenshot',
      use: {
        ...devices['Desktop Chrome'],
        viewport: { width: 1400, height: 900 },
      },
    },
  ],
});
