import { defineConfig } from '@playwright/test'
import { defineBddConfig } from 'playwright-bdd'

import { acceptanceEnv } from './src/support/env'

const testDir = defineBddConfig({
  features: 'features/**/*.feature',
  steps: ['src/steps/**/*.ts', 'src/support/fixtures.ts'],
  tags: acceptanceEnv.bddTags,
})

export default defineConfig({
  testDir,
  timeout: Math.max(acceptanceEnv.timeoutMs * 2, 90_000),
  fullyParallel: false,
  outputDir: 'artifacts/test-results',
  reporter: [['list']],
  use: {
    baseURL: acceptanceEnv.webBaseUrl,
    headless: acceptanceEnv.headless,
    viewport: { width: 1660, height: 1024 },
    actionTimeout: acceptanceEnv.timeoutMs,
    navigationTimeout: acceptanceEnv.timeoutMs,
    screenshot: 'only-on-failure',
    trace: 'retain-on-failure',
    video: 'retain-on-failure',
  },
  expect: {
    timeout: Math.min(acceptanceEnv.timeoutMs, 15_000),
  },
})
