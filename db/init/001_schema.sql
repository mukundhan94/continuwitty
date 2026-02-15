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

CREATE INDEX IF NOT EXISTS engrams_project_created_idx
  ON engrams (project_id, created_at DESC);

CREATE INDEX IF NOT EXISTS engrams_created_idx
  ON engrams (created_at DESC);

CREATE INDEX IF NOT EXISTS engrams_tags_gin_idx
  ON engrams USING GIN (tags);

CREATE INDEX IF NOT EXISTS engrams_keywords_gin_idx
  ON engrams USING GIN (keywords);

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
