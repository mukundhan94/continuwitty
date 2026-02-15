import type { ChatProvider, VisibilityScope } from './api/types'

interface WebRuntimeConfig {
  defaultProjectId: string
  defaultProvider: ChatProvider
  defaultVisibility: VisibilityScope
  defaultModelByProvider: Record<ChatProvider, string>
}

function parseProvider(value: string | undefined): ChatProvider {
  if (value === 'anthropic' || value === 'bedrock' || value === 'openai') {
    return value
  }
  return 'openai'
}

function parseVisibility(value: string | undefined): VisibilityScope {
  if (value === 'project' || value === 'private') {
    return value
  }
  return 'private'
}

function parseNonEmpty(value: string | undefined, fallback: string): string {
  const trimmed = (value || '').trim()
  return trimmed.length > 0 ? trimmed : fallback
}

function buildConfig(): WebRuntimeConfig {
  const provider = parseProvider(import.meta.env.VITE_DEFAULT_PROVIDER)
  return {
    defaultProjectId: parseNonEmpty(import.meta.env.VITE_DEFAULT_PROJECT_ID, 'engram-vault'),
    defaultProvider: provider,
    defaultVisibility: parseVisibility(import.meta.env.VITE_DEFAULT_VISIBILITY),
    defaultModelByProvider: {
      openai: parseNonEmpty(import.meta.env.VITE_DEFAULT_OPENAI_MODEL, 'gpt-4o-mini'),
      anthropic: parseNonEmpty(
        import.meta.env.VITE_DEFAULT_ANTHROPIC_MODEL,
        'claude-3-5-haiku-20241022',
      ),
      bedrock: parseNonEmpty(
        import.meta.env.VITE_DEFAULT_BEDROCK_MODEL,
        'anthropic.claude-3-haiku-20240307-v1:0',
      ),
    },
  }
}

export const WEB_CONFIG = buildConfig()

console.info('[engram-web] parsed config', WEB_CONFIG)
