ALTER TABLE users ADD COLUMN timezone TEXT NOT NULL DEFAULT 'UTC';
ALTER TABLE users ADD COLUMN avatar_storage_id UUID REFERENCES storage_objects(id);
ALTER TABLE users ADD COLUMN auth_method TEXT NOT NULL DEFAULT 'oidc';
