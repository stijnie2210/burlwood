import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './e2e',
  testMatch: '**/*.e2e.ts',
  globalSetup: './e2e/global-setup.ts',
  fullyParallel: true,
  timeout: 30_000,
  expect: { timeout: 10_000 },
  reporter: process.env.CI ? 'github' : 'list',
  use: {
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
  },
  // WebKit is the closest engine to the WKWebView the macOS app renders in.
  projects: [
    {
      name: 'webkit',
      use: { ...devices['Desktop Safari'], viewport: { width: 1440, height: 900 } },
    },
  ],
});
