-- Set schema
SET search_path TO public;

-- Drop triggers (you must drop them before dropping the tables)
DROP TRIGGER IF EXISTS update_catalogs_updated_at ON catalogs;
DROP TRIGGER IF EXISTS update_variants_updated_at ON variants;
DROP TRIGGER IF EXISTS update_tenants_updated_at ON tenants;
DROP TRIGGER IF EXISTS update_projects_updated_at ON projects;
DROP TRIGGER IF EXISTS update_catalog_objects_updated_at ON catalog_objects;
DROP TRIGGER IF EXISTS update_resource_directory_updated_at ON resource_directory;
DROP TRIGGER IF EXISTS update_skillset_directory_updated_at ON skillset_directory;
DROP TRIGGER IF EXISTS update_namespaces_updated_at ON namespaces;
DROP TRIGGER IF EXISTS update_view_tokens_updated_at ON view_tokens;
DROP TRIGGER IF EXISTS update_views_updated_at ON views;
DROP TRIGGER IF EXISTS update_signing_keys_updated_at ON signing_keys;
DROP TRIGGER IF EXISTS update_sessions_updated_at ON sessions;
DROP TRIGGER IF EXISTS update_tangents_updated_at ON tangents;

-- Drop functions
DROP FUNCTION IF EXISTS set_updated_at() CASCADE;

-- Drop tables (in reverse dependency order)
DROP TABLE IF EXISTS tangents CASCADE;
DROP TABLE IF EXISTS sessions CASCADE;
DROP TABLE IF EXISTS view_tokens CASCADE;
DROP TABLE IF EXISTS views CASCADE;
DROP TABLE IF EXISTS namespaces CASCADE;
DROP TABLE IF EXISTS resource_directory CASCADE;
DROP TABLE IF EXISTS skillset_directory CASCADE;
DROP TABLE IF EXISTS catalog_objects CASCADE;
DROP SEQUENCE IF EXISTS catalog_objects_id_seq CASCADE;
DROP TABLE IF EXISTS variants CASCADE;
DROP TABLE IF EXISTS catalogs CASCADE;
DROP TABLE IF EXISTS signing_keys CASCADE;
DROP TABLE IF EXISTS projects CASCADE;
DROP TABLE IF EXISTS tenants CASCADE;

-- Done!
