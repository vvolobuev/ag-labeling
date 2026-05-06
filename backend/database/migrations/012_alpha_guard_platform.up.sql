-- Alpha Guard AI platform (orthogonal to legacy clinic tables)

CREATE TABLE IF NOT EXISTS ag_users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email VARCHAR(320) NOT NULL UNIQUE,
  password_hash VARCHAR(255) NOT NULL,
  email_verified BOOLEAN NOT NULL DEFAULT FALSE,
  verification_token VARCHAR(128),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS ix_ag_users_verification_token ON ag_users (verification_token) WHERE verification_token IS NOT NULL;

CREATE TABLE IF NOT EXISTS ag_workspaces (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name VARCHAR(255) NOT NULL,
  slug VARCHAR(255) NOT NULL UNIQUE,
  created_by UUID REFERENCES ag_users(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS ag_workspace_members (
  workspace_id UUID NOT NULL REFERENCES ag_workspaces(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES ag_users(id) ON DELETE CASCADE,
  role VARCHAR(32) NOT NULL CHECK (role IN ('owner','admin','annotator','viewer')),
  invited_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (workspace_id, user_id)
);

CREATE TABLE IF NOT EXISTS ag_projects (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id UUID NOT NULL REFERENCES ag_workspaces(id) ON DELETE CASCADE,
  name VARCHAR(255) NOT NULL,
  slug VARCHAR(255) NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (workspace_id, slug)
);

CREATE TABLE IF NOT EXISTS ag_dataset_versions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id UUID NOT NULL REFERENCES ag_projects(id) ON DELETE CASCADE,
  name VARCHAR(255) NOT NULL,
  data_yaml TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS ag_dataset_images (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  version_id UUID NOT NULL REFERENCES ag_dataset_versions(id) ON DELETE CASCADE,
  split VARCHAR(16) NOT NULL CHECK (split IN ('train','valid','test')),
  stem VARCHAR(512) NOT NULL,
  ext VARCHAR(32) NOT NULL DEFAULT '',
  rel_image_path VARCHAR(2048) NOT NULL,
  label_text TEXT NOT NULL DEFAULT '',
  width INT NOT NULL DEFAULT 0,
  height INT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (version_id, split, stem)
);

CREATE INDEX IF NOT EXISTS ix_ag_dataset_images_vsplit ON ag_dataset_images (version_id, split);
