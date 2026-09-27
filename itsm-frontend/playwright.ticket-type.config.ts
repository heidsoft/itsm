import { defineConfig } from '@playwright/test';
import baseConfig from './playwright.config';

// Exported for the spec's own guard when discovered with a different config.
// Keep this config independent of tests/: Docker excludes that directory.
export function requireIsolatedStack(): string {
  if (process.env.ITSM_E2E_ISOLATED_STACK !== '1') {
    throw new Error(
      'TicketType E2E requires ITSM_E2E_ISOLATED_STACK=1 and a disposable Compose stack'
    );
  }
  const baseURL = process.env.PLAYWRIGHT_BASE_URL;
  if (baseURL !== 'http://localhost:3000' && baseURL !== 'http://127.0.0.1:3000') {
    throw new Error(
      'TicketType E2E requires an explicit loopback PLAYWRIGHT_BASE_URL on port 3000'
    );
  }
  return baseURL;
}

// Reuse browser settings, but never start/reuse a developer server implicitly.
export default defineConfig({
  testDir: './tests/e2e',
  testMatch: 'ticket-type-full-chain.spec.ts',
  outputDir: './test-results/ticket-type-full-chain',
  timeout: 180_000,
  expect: { timeout: 10_000 },
  fullyParallel: false,
  workers: 1,
  retries: 0,
  forbidOnly: true,
  use: {
    ...baseConfig.use,
    baseURL: requireIsolatedStack(),
    trace: 'retain-on-failure',
  },
  projects: [{ name: 'chromium', use: { browserName: 'chromium' } }],
  reporter: [
    ['list'],
    ['html', { open: 'never', outputFolder: './playwright-report/ticket-type-full-chain' }],
    ['junit', { outputFile: './test-results/ticket-type-full-chain.xml' }],
  ],
  // Deliberately no webServer: only the explicitly provisioned Compose stack.
});
