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
  ADD COLUMN IF NOT EXISTS source_session_id UUID,
  ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS deleted_by_user_id UUID,
  ADD COLUMN IF NOT EXISTS delete_reason TEXT,
  ADD COLUMN IF NOT EXISTS updated_by_user_id UUID;

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

CREATE INDEX IF NOT EXISTS engrams_active_project_created_idx
  ON engrams (project_id, created_at DESC)
  WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS engrams_deleted_at_idx
  ON engrams (deleted_at);

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
  default_project_id TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS users_role_idx ON users (role);
CREATE INDEX IF NOT EXISTS users_active_idx ON users (is_active);

ALTER TABLE users
  ADD COLUMN IF NOT EXISTS default_project_id TEXT;

-- Fresh setup seed user for local development.
-- username: admin
-- password: admin123
INSERT INTO users (user_id, username, password_hash, role, is_active)
VALUES (
  '00000000-0000-0000-0000-000000000001'::UUID,
  'admin',
  'pbkdf2_sha256$390000$00112233445566778899aabbccddeeff$45c0bdc16f1609a69d14a4e8d89974fd24556c45af75ee01c34bdb9de1c1832a',
  'admin',
  TRUE
)
ON CONFLICT (username) DO NOTHING;

CREATE TABLE IF NOT EXISTS projects (
  project_id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  owner_user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
  is_archived BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CHECK (length(trim(project_id)) > 0)
);

CREATE INDEX IF NOT EXISTS projects_owner_created_idx
  ON projects (owner_user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS projects_archived_idx
  ON projects (is_archived);

INSERT INTO projects (project_id, name, description, owner_user_id, is_archived)
SELECT
  'engram-vault',
  'Engram Vault',
  'Default local workspace project.',
  COALESCE(
    (SELECT user_id FROM users WHERE username = 'admin' LIMIT 1),
    (SELECT user_id FROM users ORDER BY created_at ASC LIMIT 1)
  ),
  FALSE
ON CONFLICT (project_id) DO NOTHING;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'users_default_project_fk'
  ) THEN
    ALTER TABLE users
      ADD CONSTRAINT users_default_project_fk
      FOREIGN KEY (default_project_id)
      REFERENCES projects(project_id)
      ON DELETE SET NULL;
  END IF;
END $$;

CREATE TABLE IF NOT EXISTS mcp_tokens (
  token_id UUID PRIMARY KEY,
  owner_user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  scope TEXT NOT NULL CHECK (scope IN ('read', 'write')),
  allowed_tools TEXT[] NOT NULL DEFAULT '{}',
  allowed_project_ids TEXT[] NOT NULL DEFAULT '{}',
  token_secret_hash TEXT NOT NULL,
  token_secret_hint TEXT NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  last_used_at TIMESTAMPTZ,
  revoked_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS mcp_tokens_owner_created_idx
  ON mcp_tokens (owner_user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS mcp_tokens_expires_idx
  ON mcp_tokens (expires_at);

CREATE INDEX IF NOT EXISTS mcp_tokens_active_idx
  ON mcp_tokens (owner_user_id, expires_at)
  WHERE revoked_at IS NULL;

CREATE TABLE IF NOT EXISTS oauth_clients (
  client_id TEXT PRIMARY KEY,
  client_name TEXT NOT NULL,
  redirect_uris TEXT[] NOT NULL,
  grant_types TEXT[] NOT NULL DEFAULT ARRAY['authorization_code'],
  response_types TEXT[] NOT NULL DEFAULT ARRAY['code'],
  token_endpoint_auth_method TEXT NOT NULL DEFAULT 'none'
    CHECK (token_endpoint_auth_method IN ('none', 'client_secret_post')),
  client_secret_hash TEXT,
  metadata_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS oauth_clients_created_idx
  ON oauth_clients (created_at DESC);

CREATE TABLE IF NOT EXISTS oauth_authorization_codes (
  code_id UUID PRIMARY KEY,
  code_hash TEXT NOT NULL UNIQUE,
  client_id TEXT NOT NULL REFERENCES oauth_clients(client_id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
  redirect_uri TEXT NOT NULL,
  code_challenge TEXT NOT NULL,
  code_challenge_method TEXT NOT NULL DEFAULT 'S256'
    CHECK (code_challenge_method IN ('S256', 'plain')),
  requested_scope TEXT NOT NULL DEFAULT '',
  resource TEXT,
  expires_at TIMESTAMPTZ NOT NULL,
  consumed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS oauth_authorization_codes_client_expires_idx
  ON oauth_authorization_codes (client_id, expires_at DESC);

CREATE INDEX IF NOT EXISTS oauth_authorization_codes_user_created_idx
  ON oauth_authorization_codes (user_id, created_at DESC);

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
  autosave_strategy TEXT NOT NULL DEFAULT 'off',
  autosave_interval_minutes INTEGER NOT NULL DEFAULT 30,
  autosave_min_messages INTEGER NOT NULL DEFAULT 6,
  retention_days INTEGER NOT NULL DEFAULT 30,
  retention_max_snapshots INTEGER NOT NULL DEFAULT 60,
  deleted_at TIMESTAMPTZ,
  deleted_by_user_id UUID,
  delete_reason TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE chat_sessions
  ADD COLUMN IF NOT EXISTS autosave_strategy TEXT NOT NULL DEFAULT 'off',
  ADD COLUMN IF NOT EXISTS autosave_interval_minutes INTEGER NOT NULL DEFAULT 30,
  ADD COLUMN IF NOT EXISTS autosave_min_messages INTEGER NOT NULL DEFAULT 6,
  ADD COLUMN IF NOT EXISTS retention_days INTEGER NOT NULL DEFAULT 30,
  ADD COLUMN IF NOT EXISTS retention_max_snapshots INTEGER NOT NULL DEFAULT 60,
  ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS deleted_by_user_id UUID,
  ADD COLUMN IF NOT EXISTS delete_reason TEXT;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'chat_sessions_autosave_strategy_check'
  ) THEN
    ALTER TABLE chat_sessions
      ADD CONSTRAINT chat_sessions_autosave_strategy_check
      CHECK (autosave_strategy IN ('off', 'interval', 'message_count'));
  END IF;
END $$;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'chat_sessions_autosave_interval_minutes_check'
  ) THEN
    ALTER TABLE chat_sessions
      ADD CONSTRAINT chat_sessions_autosave_interval_minutes_check
      CHECK (autosave_interval_minutes >= 1);
  END IF;
END $$;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'chat_sessions_autosave_min_messages_check'
  ) THEN
    ALTER TABLE chat_sessions
      ADD CONSTRAINT chat_sessions_autosave_min_messages_check
      CHECK (autosave_min_messages >= 1);
  END IF;
END $$;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'chat_sessions_retention_days_check'
  ) THEN
    ALTER TABLE chat_sessions
      ADD CONSTRAINT chat_sessions_retention_days_check
      CHECK (retention_days >= 1);
  END IF;
END $$;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'chat_sessions_retention_max_snapshots_check'
  ) THEN
    ALTER TABLE chat_sessions
      ADD CONSTRAINT chat_sessions_retention_max_snapshots_check
      CHECK (retention_max_snapshots >= 1);
  END IF;
END $$;

CREATE INDEX IF NOT EXISTS chat_sessions_owner_created_idx
  ON chat_sessions (owner_user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS chat_sessions_project_created_idx
  ON chat_sessions (project_id, created_at DESC);

CREATE INDEX IF NOT EXISTS chat_sessions_visibility_idx
  ON chat_sessions (visibility_scope);

CREATE INDEX IF NOT EXISTS chat_sessions_active_project_created_idx
  ON chat_sessions (project_id, created_at DESC)
  WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS chat_sessions_deleted_at_idx
  ON chat_sessions (deleted_at);

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

CREATE TABLE IF NOT EXISTS engram_collections (
  collection_id UUID PRIMARY KEY,
  project_id TEXT NOT NULL REFERENCES projects(project_id) ON DELETE CASCADE,
  owner_user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at TIMESTAMPTZ,
  deleted_by_user_id UUID REFERENCES users(user_id) ON DELETE SET NULL,
  delete_reason TEXT
);

CREATE UNIQUE INDEX IF NOT EXISTS engram_collections_project_name_active_uidx
  ON engram_collections (project_id, name)
  WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS engram_collections_owner_created_idx
  ON engram_collections (owner_user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS engram_collections_project_created_idx
  ON engram_collections (project_id, created_at DESC);

CREATE TABLE IF NOT EXISTS engram_collection_items (
  collection_id UUID NOT NULL REFERENCES engram_collections(collection_id) ON DELETE CASCADE,
  engram_id UUID NOT NULL REFERENCES engrams(engram_id) ON DELETE CASCADE,
  added_by_user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (collection_id, engram_id)
);

CREATE INDEX IF NOT EXISTS engram_collection_items_engram_idx
  ON engram_collection_items (engram_id, created_at DESC);

CREATE TABLE IF NOT EXISTS documents (
  document_id UUID PRIMARY KEY,
  owner_user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
  project_id TEXT NOT NULL,
  title TEXT NOT NULL,
  source_type TEXT NOT NULL CHECK (source_type IN ('text', 'file')),
  source_name TEXT,
  mime_type TEXT,
  visibility_scope TEXT NOT NULL DEFAULT 'private' CHECK (visibility_scope IN ('private', 'project')),
  content_text TEXT NOT NULL,
  content_hash TEXT NOT NULL,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  chunk_count INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS documents_owner_project_created_idx
  ON documents (owner_user_id, project_id, created_at DESC);

CREATE INDEX IF NOT EXISTS documents_project_created_idx
  ON documents (project_id, created_at DESC);

CREATE INDEX IF NOT EXISTS documents_visibility_idx
  ON documents (visibility_scope);

CREATE INDEX IF NOT EXISTS documents_content_hash_idx
  ON documents (content_hash);

CREATE TABLE IF NOT EXISTS session_pinned_documents (
  session_id UUID NOT NULL REFERENCES chat_sessions(session_id) ON DELETE CASCADE,
  document_id UUID NOT NULL REFERENCES documents(document_id) ON DELETE CASCADE,
  pinned_by_user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (session_id, document_id)
);

CREATE INDEX IF NOT EXISTS session_pinned_documents_by_user_idx
  ON session_pinned_documents (pinned_by_user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS document_chunks (
  chunk_id UUID PRIMARY KEY,
  document_id UUID NOT NULL REFERENCES documents(document_id) ON DELETE CASCADE,
  chunk_index INTEGER NOT NULL,
  chunk_text TEXT NOT NULL,
  snippet TEXT NOT NULL,
  char_start INTEGER NOT NULL,
  char_end INTEGER NOT NULL,
  token_estimate INTEGER NOT NULL DEFAULT 0,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  embedding_model TEXT NOT NULL DEFAULT 'local-deterministic-v1',
  embed VECTOR(256),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (document_id, chunk_index)
);

CREATE INDEX IF NOT EXISTS document_chunks_document_idx
  ON document_chunks (document_id, chunk_index);

CREATE INDEX IF NOT EXISTS document_chunks_embed_hnsw_idx
  ON document_chunks USING hnsw (embed vector_cosine_ops);

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

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'engrams_deleted_by_user_fk'
  ) THEN
    ALTER TABLE engrams
      ADD CONSTRAINT engrams_deleted_by_user_fk
      FOREIGN KEY (deleted_by_user_id)
      REFERENCES users(user_id)
      ON DELETE SET NULL;
  END IF;
END $$;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'engrams_updated_by_user_fk'
  ) THEN
    ALTER TABLE engrams
      ADD CONSTRAINT engrams_updated_by_user_fk
      FOREIGN KEY (updated_by_user_id)
      REFERENCES users(user_id)
      ON DELETE SET NULL;
  END IF;
END $$;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'chat_sessions_deleted_by_user_fk'
  ) THEN
    ALTER TABLE chat_sessions
      ADD CONSTRAINT chat_sessions_deleted_by_user_fk
      FOREIGN KEY (deleted_by_user_id)
      REFERENCES users(user_id)
      ON DELETE SET NULL;
  END IF;
END $$;

WITH discovered_projects AS (
  SELECT DISTINCT project_id
  FROM (
    SELECT project_id FROM engrams WHERE project_id IS NOT NULL AND length(trim(project_id)) > 0
    UNION
    SELECT project_id FROM chat_sessions WHERE project_id IS NOT NULL AND length(trim(project_id)) > 0
    UNION
    SELECT project_id FROM documents WHERE project_id IS NOT NULL AND length(trim(project_id)) > 0
  ) AS raw_ids
)
INSERT INTO projects (project_id, name, description, owner_user_id, is_archived)
SELECT
  discovered.project_id,
  discovered.project_id,
  'Backfilled project from existing memory records.',
  COALESCE(
    (
      SELECT e.owner_user_id
      FROM engrams e
      WHERE e.project_id = discovered.project_id
        AND e.owner_user_id IS NOT NULL
      ORDER BY e.created_at ASC
      LIMIT 1
    ),
    (
      SELECT s.owner_user_id
      FROM chat_sessions s
      WHERE s.project_id = discovered.project_id
      ORDER BY s.created_at ASC
      LIMIT 1
    ),
    (
      SELECT d.owner_user_id
      FROM documents d
      WHERE d.project_id = discovered.project_id
      ORDER BY d.created_at ASC
      LIMIT 1
    ),
    (SELECT user_id FROM users ORDER BY created_at ASC LIMIT 1)
  ),
  FALSE
FROM discovered_projects discovered
ON CONFLICT (project_id) DO NOTHING;

UPDATE users u
SET default_project_id = COALESCE(
  (
    SELECT p.project_id
    FROM projects p
    WHERE p.owner_user_id = u.user_id
      AND p.is_archived = FALSE
    ORDER BY p.created_at ASC
    LIMIT 1
  ),
  (
    SELECT p.project_id
    FROM projects p
    WHERE p.is_archived = FALSE
    ORDER BY p.created_at ASC
    LIMIT 1
  ),
  u.default_project_id
)
WHERE u.default_project_id IS NULL;
