const { defineConfig, devices } = require('@playwright/test');
module.exports = defineConfig({
  testDir: './specs',
  fullyParallel: false,
  workers: 1,
  reporter: [
    ['html'],
    ['list'],
    ['json', { outputFile: 'results/performance-results.json' }]
  ],
  use: {
    baseURL: process.env.WEBCLAW_URL || 'http://localhost:8080',
  },
  projects: [
    { name: 'chromium', use: { ...devices['Desktop Chrome'] } },
    { name: 'firefox', use: { ...devices['Desktop Firefox'] } },
    { name: 'webkit', use: { ...devices['Desktop Safari'] } },
  ],
});
