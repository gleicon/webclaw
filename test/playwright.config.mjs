// @ts-check
import { defineConfig, devices } from '@playwright/test';

/**
 * @see https://playwright.dev/docs/test-configuration
 */
export default defineConfig({
  testDir: './phase06-browser-tests',
  
  /* Run tests in files in parallel */
  fullyParallel: false,
  
  /* Fail the build on CI if you accidentally left test.only in the source code */
  forbidOnly: !!process.env.CI,
  
  /* Retry on CI only */
  retries: process.env.CI ? 2 : 0,
  
  /* Opt out of parallel tests on CI - WebClaw needs sequential execution */
  workers: 1,
  
  /* Reporter to use */
  reporter: [
    ['list'],
    ['html', { open: 'never', outputFolder: 'playwright-report' }],
    ['json', { outputFile: 'playwright-report/results.json' }]
  ],
  
  /* Shared settings for all the projects below */
  use: {
    /* Base URL to use in actions like `await page.goto('/')` */
    baseURL: 'http://localhost:8080',
    
    /* Collect trace when retrying the failed test */
    trace: 'on-first-retry',
    
    /* Screenshot on failure */
    screenshot: 'only-on-failure',
    
    /* Record video on failure */
    video: 'on-first-retry',
    
    /* Console capture settings */
    launchOptions: {
      args: [
        '--no-sandbox',
        '--disable-setuid-sandbox',
        '--enable-features=SharedArrayBuffer',
        '--disable-features=IsolateOrigins,site-per-process'
      ]
    },
    
    /* Viewport settings */
    viewport: { width: 1280, height: 720 },
    
    /* Action timeout */
    actionTimeout: 15000,
    
    /* Navigation timeout */
    navigationTimeout: 30000
  },
  
  /* Configure projects for major browsers - just Chromium for WASM */
  projects: [
    {
      name: 'chromium',
      use: { 
        ...devices['Desktop Chrome'],
        headless: process.env.HEADLESS !== 'false'
      },
    },
  ],
  
  /* Run local dev server before starting the tests */
  webServer: {
    command: 'cd .. && go run ./cmd/devserver/',
    url: 'http://localhost:8080',
    reuseExistingServer: !process.env.CI,
    timeout: 120000,
  },
});
