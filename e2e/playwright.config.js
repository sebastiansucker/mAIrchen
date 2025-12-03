import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './',
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: 1,
  timeout: 120000, // 2 minutes per test
  reporter: [
    ['html'],
    ['list']
  ],
  use: {
    baseURL: 'http://localhost:80',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
    actionTimeout: 15000, // 15 seconds for actions
    navigationTimeout: 15000, // 15 seconds for page navigation
  },

  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],

  // Don't start a web server - we expect the app to be running already
  webServer: undefined,
});
