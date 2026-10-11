import { defineConfig, devices } from '@playwright/test'

const domain = process.env.PLAYWRIGHT_DOMAIN ?? 'bookworm.com'
const app = process.env.PLAYWRIGHT_APP ?? 'rustdesk'

export default defineConfig({
  testDir: './e2e',
  fullyParallel: false,
  workers: 1,
  retries: process.env.CI ? 1 : 0,
  reporter: 'list',
  use: {
    baseURL: `https://${app}.${domain}`,
    ignoreHTTPSErrors: true,
    permissions: ['clipboard-read', 'clipboard-write'],
  },
  projects: [
    {
      name: 'desktop',
      use: { ...devices['Desktop Chrome'], viewport: { width: 1440, height: 960 } },
    },
  ],
})
