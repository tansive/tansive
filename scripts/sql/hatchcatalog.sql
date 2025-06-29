SET search_path TO public;

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE IF NOT EXISTS tenants (
  tenant_id VARCHAR(10) PRIMARY KEY,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TRIGGER update_tenants_updated_at
BEFORE UPDATE ON tenants
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS projects (
  project_id VARCHAR(10),
  tenant_id VARCHAR(10),
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW(),
  PRIMARY KEY (tenant_id, project_id),
  FOREIGN KEY (tenant_id) REFERENCES tenants(tenant_id) ON DELETE CASCADE
);

CREATE TRIGGER update_projects_updated_at
BEFORE UPDATE ON projects
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS catalogs (
  catalog_id UUID DEFAULT uuid_generate_v4(),
  name VARCHAR(128) NOT NULL,
  description VARCHAR(1024),
  info JSONB,
  project_id VARCHAR(10) NOT NULL,
  tenant_id VARCHAR(10) NOT NULL,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW(),
  UNIQUE (tenant_id, project_id, name),
  PRIMARY KEY (tenant_id, catalog_id),
  FOREIGN KEY (tenant_id, project_id) REFERENCES projects(tenant_id, project_id) ON DELETE CASCADE,
  CHECK (name ~ '^[A-Za-z0-9_-]+$') -- CHECK constraint to allow only alphanumeric and underscore in name
);

CREATE TRIGGER update_catalogs_updated_at
BEFORE UPDATE ON catalogs
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS variants (
  variant_id UUID DEFAULT uuid_generate_v4(),
  name VARCHAR(128) NOT NULL,
  description VARCHAR(1024),
  info JSONB,
  resource_directory UUID DEFAULT uuid_nil(),
  skillset_directory UUID DEFAULT uuid_nil(),
  catalog_id UUID NOT NULL,
  tenant_id VARCHAR(10) NOT NULL,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW(),
  UNIQUE (tenant_id, catalog_id, name),
  PRIMARY KEY (tenant_id, variant_id),
  FOREIGN KEY (tenant_id, catalog_id) REFERENCES catalogs(tenant_id, catalog_id) ON DELETE CASCADE,
  CHECK (name ~ '^[A-Za-z0-9_-]+$') -- CHECK constraint to allow only alphanumeric and underscore in name
);

CREATE TRIGGER update_variants_updated_at
BEFORE UPDATE ON variants
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS catalog_objects (
  id SERIAL PRIMARY KEY,
  hash_id CHAR(16) NOT NULL,
  hash CHAR(128) NOT NULL,
  type VARCHAR(64) NOT NULL CHECK (type IN ('resource', 'skillset')),
  version VARCHAR(16) NOT NULL,
  tenant_id VARCHAR(10) NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  data BYTEA NOT NULL,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_catalog_objects_tenant_id_hash ON catalog_objects (tenant_id, hash_id);

CREATE TRIGGER update_catalog_objects_updated_at
BEFORE UPDATE ON catalog_objects
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS resource_directory ( 
  directory_id UUID NOT NULL DEFAULT uuid_generate_v4(),
  variant_id UUID NOT NULL,
  tenant_id VARCHAR(10) NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  directory JSONB NOT NULL,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW(),
  PRIMARY KEY (tenant_id, directory_id),
  FOREIGN KEY (tenant_id, variant_id) REFERENCES variants(tenant_id, variant_id) ON DELETE CASCADE
);

CREATE TRIGGER update_resource_directory_updated_at
BEFORE UPDATE ON resource_directory
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE INDEX IF NOT EXISTS idx_resource_directory_hash_gin
ON resource_directory USING GIN (jsonb_path_query_array(directory, '$.*.hash'));

CREATE TABLE IF NOT EXISTS skillset_directory ( 
  directory_id UUID NOT NULL DEFAULT uuid_generate_v4(),
  variant_id UUID NOT NULL,
  tenant_id VARCHAR(10) NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  directory JSONB NOT NULL,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW(),
  PRIMARY KEY (tenant_id, directory_id),
  FOREIGN KEY (tenant_id, variant_id) REFERENCES variants(tenant_id, variant_id) ON DELETE CASCADE
);

CREATE TRIGGER update_skillset_directory_updated_at
BEFORE UPDATE ON skillset_directory
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE INDEX IF NOT EXISTS idx_skillset_directory_hash_gin
ON skillset_directory USING GIN (jsonb_path_query_array(directory, '$.*.hash'));

CREATE TABLE IF NOT EXISTS namespaces (
  name VARCHAR(128) NOT NULL,
  variant_id UUID NOT NULL,
  tenant_id VARCHAR(10) NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  description VARCHAR(1024),
  info JSONB,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW(),
  PRIMARY KEY (tenant_id, variant_id, name),
  FOREIGN KEY (tenant_id, variant_id) REFERENCES variants(tenant_id, variant_id) ON DELETE CASCADE,
  CHECK (name ~ '^[A-Za-z0-9_-]+$') -- CHECK constraint to allow only alphanumeric and underscore in name
);

CREATE TRIGGER update_namespaces_updated_at
BEFORE UPDATE ON namespaces
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS views (
  view_id UUID NOT NULL DEFAULT uuid_generate_v4(),
  label VARCHAR(128),
  description VARCHAR(1024),
  info JSONB,
  rules JSONB NOT NULL,
  catalog_id UUID NOT NULL,
  created_by VARCHAR(128) NOT NULL,
  updated_by VARCHAR(128) NOT NULL,
  tenant_id VARCHAR(10) NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW(),
  UNIQUE (tenant_id, catalog_id, label),
  PRIMARY KEY (tenant_id, view_id),
  FOREIGN KEY (tenant_id, catalog_id) REFERENCES catalogs(tenant_id, catalog_id) ON DELETE CASCADE,
  CHECK (label IS NULL OR label ~ '^[A-Za-z0-9_-]+$')  -- CHECK constraint to allow only alphanumeric and underscore in label
);

CREATE TRIGGER update_views_updated_at
BEFORE UPDATE ON views
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS view_tokens (
  token_id UUID NOT NULL DEFAULT uuid_generate_v4(),
  view_id UUID NOT NULL,
  tenant_id VARCHAR(10) NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  expire_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW(),
  PRIMARY KEY (tenant_id, token_id)
);

CREATE TRIGGER update_view_tokens_updated_at
BEFORE UPDATE ON view_tokens
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS signing_keys (
  key_id UUID NOT NULL DEFAULT uuid_generate_v4(),
  public_key BYTEA NOT NULL,
  private_key BYTEA NOT NULL,
  is_active BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW(),
  PRIMARY KEY (key_id)
);

CREATE TRIGGER update_signing_keys_updated_at
BEFORE UPDATE ON signing_keys
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE UNIQUE INDEX idx_active_signing_key
ON signing_keys (is_active)
WHERE is_active = true;

CREATE TABLE IF NOT EXISTS sessions (
  session_id UUID NOT NULL DEFAULT uuid_generate_v4(),
  skillset VARCHAR(128) NOT NULL,
  skill VARCHAR(128) NOT NULL,
  view_id UUID NOT NULL,
  tangent_id UUID NOT NULL,
  status_summary VARCHAR(128) NOT NULL,
  status JSONB NOT NULL,
  info JSONB,
  user_id VARCHAR(128) NOT NULL,
  catalog_id UUID NOT NULL,
  variant_id UUID NOT NULL,
  tenant_id VARCHAR(10) NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  started_at TIMESTAMPTZ NOT NULL,
  ended_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ DEFAULT NOW(),
  expires_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (tenant_id, session_id),
  FOREIGN KEY (tenant_id, catalog_id) REFERENCES catalogs(tenant_id, catalog_id) ON DELETE CASCADE
);

CREATE TRIGGER update_sessions_updated_at
BEFORE UPDATE ON sessions
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE INDEX IF NOT EXISTS idx_sessions_tenant_catalog_status
ON sessions (tenant_id, catalog_id, status_summary);

CREATE TABLE IF NOT EXISTS tangents (
  id UUID NOT NULL DEFAULT uuid_generate_v4() PRIMARY KEY,
  public_key BYTEA NOT NULL,
  info JSONB,
  status VARCHAR(128) NOT NULL,
  tenant_id VARCHAR(10) NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE UNIQUE INDEX idx_tangents_tenant_id_id ON tangents (tenant_id, id);

CREATE TRIGGER update_tangents_updated_at
BEFORE UPDATE ON tangents
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

GRANT ALL PRIVILEGES ON TABLE
	tenants,
	projects,
	catalogs,
	variants,
  catalog_objects,
  resource_directory,
  skillset_directory,
  namespaces,
  views,
  view_tokens,
  signing_keys,
  sessions,
  tangents
TO catalogrw;

GRANT USAGE, SELECT ON SEQUENCE catalog_objects_id_seq TO catalogrw;
