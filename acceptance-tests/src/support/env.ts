import { config as loadDotenv } from 'dotenv'

loadDotenv({ path: process.env.ACCEPTANCE_ENV_FILE || '.env' })

function asInt(value: string | undefined, fallback: number): number {
  const parsed = Number(value)
  return Number.isFinite(parsed) && parsed > 0 ? parsed : fallback
}

export const acceptanceEnv = {
  webBaseUrl: process.env.WEB_BASE_URL || 'http://localhost:5174',
  apiBaseUrl: process.env.API_BASE_URL || 'http://localhost:8000',
  username: process.env.UI_USERNAME || 'admin',
  password: process.env.UI_PASSWORD || 'admin123',
  headless: (process.env.PW_HEADLESS || 'true').toLowerCase() !== 'false',
  timeoutMs: asInt(process.env.PW_TIMEOUT_MS, 45000),
}
