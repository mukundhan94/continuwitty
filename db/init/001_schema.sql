CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE IF NOT EXISTS engrams (
  engram_id UUID PRIMARY KEY,
  project_id TEXT NOT NULL,
  thread_id TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  schema_version TEXT NOT NULL DEFAULT '1.0',
  title TEXT NOT NULL,
  abstract TEXT NOT NULL,
  engram_json JSONB NOT NULL,
  engram_markdown TEXT NOT NULL,
  tags TEXT[] NOT NULL DEFAULT '{}',
  keywords TEXT[] NOT NULL DEFAULT '{}',
  retrieval_text TEXT NOT NULL,
  embedding_model TEXT NOT NULL DEFAULT 'local-deterministic-v1',
  embed VECTOR(256)
);

ALTER TABLE engrams
  ADD COLUMN IF NOT EXISTS owner_user_id UUID,
  ADD COLUMN IF NOT EXISTS visibility_scope TEXT NOT NULL DEFAULT 'private',
  ADD COLUMN IF NOT EXISTS source_session_id UUID;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'engrams_visibility_scope_check'
  ) THEN
    ALTER TABLE engrams
      ADD CONSTRAINT engrams_visibility_scope_check
      CHECK (visibility_scope IN ('private', 'project'));
  END IF;
END $$;

CREATE INDEX IF NOT EXISTS engrams_project_created_idx
  ON engrams (project_id, created_at DESC);

CREATE INDEX IF NOT EXISTS engrams_created_idx
  ON engrams (created_at DESC);

CREATE INDEX IF NOT EXISTS engrams_tags_gin_idx
  ON engrams USING GIN (tags);

CREATE INDEX IF NOT EXISTS engrams_keywords_gin_idx
  ON engrams USING GIN (keywords);

CREATE INDEX IF NOT EXISTS engrams_owner_idx
  ON engrams (owner_user_id);

CREATE INDEX IF NOT EXISTS engrams_visibility_idx
  ON engrams (visibility_scope);

CREATE INDEX IF NOT EXISTS engrams_embed_hnsw_idx
  ON engrams USING hnsw (embed vector_cosine_ops);

CREATE TABLE IF NOT EXISTS sources (
  source_id UUID PRIMARY KEY,
  engram_id UUID NOT NULL REFERENCES engrams(engram_id) ON DELETE CASCADE,
  captured_at TIMESTAMPTZ NOT NULL,
  url TEXT NOT NULL,
  title TEXT,
  snippet TEXT,
  content_text TEXT,
  content_hash TEXT
);

CREATE INDEX IF NOT EXISTS sources_engram_idx ON sources (engram_id);
CREATE INDEX IF NOT EXISTS sources_url_hash_idx ON sources (url, content_hash);

CREATE TABLE IF NOT EXISTS artifacts (
  artifact_id UUID PRIMARY KEY,
  engram_id UUID NOT NULL REFERENCES engrams(engram_id) ON DELETE CASCADE,
  artifact_type TEXT NOT NULL,
  storage_uri TEXT NOT NULL,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS artifacts_engram_idx ON artifacts (engram_id);

CREATE TABLE IF NOT EXISTS users (
  user_id UUID PRIMARY KEY,
  username TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  role TEXT NOT NULL CHECK (role IN ('admin', 'analyst', 'viewer')),
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS users_role_idx ON users (role);
CREATE INDEX IF NOT EXISTS users_active_idx ON users (is_active);

CREATE TABLE IF NOT EXISTS chat_sessions (
  session_id UUID PRIMARY KEY,
  owner_user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
  project_id TEXT NOT NULL,
  title TEXT NOT NULL,
  provider TEXT NOT NULL,
  model_id TEXT NOT NULL,
  system_prompt TEXT NOT NULL DEFAULT '',
  visibility_scope TEXT NOT NULL DEFAULT 'private' CHECK (visibility_scope IN ('private', 'project')),
  autosave_enabled BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS chat_sessions_owner_created_idx
  ON chat_sessions (owner_user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS chat_sessions_project_created_idx
  ON chat_sessions (project_id, created_at DESC);

CREATE INDEX IF NOT EXISTS chat_sessions_visibility_idx
  ON chat_sessions (visibility_scope);

CREATE TABLE IF NOT EXISTS chat_messages (
  message_id UUID PRIMARY KEY,
  session_id UUID NOT NULL REFERENCES chat_sessions(session_id) ON DELETE CASCADE,
  role TEXT NOT NULL CHECK (role IN ('system', 'user', 'assistant', 'tool')),
  content_text TEXT NOT NULL,
  provider TEXT,
  model_id TEXT,
  token_usage_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  used_engram_ids UUID[] NOT NULL DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS chat_messages_session_created_idx
  ON chat_messages (session_id, created_at ASC);

CREATE TABLE IF NOT EXISTS session_pinned_engrams (
  session_id UUID NOT NULL REFERENCES chat_sessions(session_id) ON DELETE CASCADE,
  engram_id UUID NOT NULL REFERENCES engrams(engram_id) ON DELETE CASCADE,
  pinned_by_user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (session_id, engram_id)
);

CREATE INDEX IF NOT EXISTS session_pinned_engrams_by_user_idx
  ON session_pinned_engrams (pinned_by_user_id, created_at DESC);

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'engrams_owner_user_fk'
  ) THEN
    ALTER TABLE engrams
      ADD CONSTRAINT engrams_owner_user_fk
      FOREIGN KEY (owner_user_id)
      REFERENCES users(user_id)
      ON DELETE SET NULL;
  END IF;
END $$;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'engrams_source_session_fk'
  ) THEN
    ALTER TABLE engrams
      ADD CONSTRAINT engrams_source_session_fk
      FOREIGN KEY (source_session_id)
      REFERENCES chat_sessions(session_id)
      ON DELETE SET NULL;
  END IF;
END $$;
