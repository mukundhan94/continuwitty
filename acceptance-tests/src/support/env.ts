import { config as loadDotenv } from 'dotenv'

loadDotenv({ path: process.env.ACCEPTANCE_ENV_FILE || '.env' })

function asInt(value: string | undefined, fallback: number): number {
  const parsed = Number(value)
  return Number.isFinite(parsed) && parsed > 0 ? parsed : fallback
}

function normalizeEnvText(value: string | undefined): string {
  const trimmed = (value || '').trim()
  if (
    (trimmed.startsWith('"') && trimmed.endsWith('"')) ||
    (trimmed.startsWith("'") && trimmed.endsWith("'"))
  ) {
    return trimmed.slice(1, -1).trim()
  }
  return trimmed
}

function asOptionalNonEmpty(value: string | undefined): string | undefined {
  const trimmed = normalizeEnvText(value)
  return trimmed.length > 0 ? trimmed : undefined
}

export const acceptanceEnv = {
  webBaseUrl: process.env.WEB_BASE_URL || 'http://localhost:5174',
  apiBaseUrl: process.env.API_BASE_URL || 'http://localhost:8000',
  username: process.env.UI_USERNAME || 'admin',
  password: process.env.UI_PASSWORD || 'admin123',
  headless: (process.env.PW_HEADLESS || 'true').toLowerCase() !== 'false',
  timeoutMs: asInt(process.env.PW_TIMEOUT_MS, 45000),
  bddTags: asOptionalNonEmpty(process.env.ACCEPTANCE_BDD_TAGS) || 'not @bedrock-live',
  bedrockLiveExpectedModel: asOptionalNonEmpty(process.env.BEDROCK_LIVE_EXPECTED_MODEL),
  bedrockLivePrompt:
    asOptionalNonEmpty(process.env.BEDROCK_LIVE_PROMPT) ||
    'Provide two concise bullet points on why session-to-engram continuity helps support handoffs.',
  bedrockLiveMinResponseChars: asInt(process.env.BEDROCK_LIVE_MIN_RESPONSE_CHARS, 20),
}
