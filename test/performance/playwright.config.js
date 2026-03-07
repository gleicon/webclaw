const { defineConfig, devices } = require('@playwright/test');
module.exports = defineConfig({
  testDir: './specs',
  fullyParallel: false,
  workers: 1,
  timeout: 60000,
  reporter: [
    ['html'],
    ['list'],
    ['json', { outputFile: 'results/performance-results.json' }]
  ],
  use: {
    baseURL: process.env.WEBCLAW_URL || 'http://localhost:8080',
  },
  // Start the dev server automatically when no external URL is provided
  webServer: process.env.WEBCLAW_URL ? undefined : {
    command: 'cd ../.. && go run ./cmd/devserver/',
    url: 'http://localhost:8080',
    reuseExistingServer: true,
    timeout: 30000,
  },
  projects: [
    { name: 'chromium', use: { ...devices['Desktop Chrome'] } },
  ],
});
