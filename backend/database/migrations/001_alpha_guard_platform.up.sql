-- Alpha Guard final baseline schema.
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS ag_users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email VARCHAR(320) NOT NULL UNIQUE,
  password_hash VARCHAR(255) NOT NULL,
  email_verified BOOLEAN NOT NULL DEFAULT FALSE,
  verification_token VARCHAR(128),
  first_name TEXT NOT NULL DEFAULT '',
  last_name TEXT NOT NULL DEFAULT '',
  avatar_path TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS ix_ag_users_verification_token
  ON ag_users (verification_token)
  WHERE verification_token IS NOT NULL;

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
  is_public BOOLEAN NOT NULL DEFAULT FALSE,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  version_counter INT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (workspace_id, slug)
);

CREATE INDEX IF NOT EXISTS ix_ag_projects_workspace_updated
  ON ag_projects (workspace_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS ix_ag_projects_public_updated
  ON ag_projects (is_public, updated_at DESC)
  WHERE is_public = TRUE;

CREATE TABLE IF NOT EXISTS ag_dataset_versions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id UUID NOT NULL REFERENCES ag_projects(id) ON DELETE CASCADE,
  name VARCHAR(255) NOT NULL,
  is_draft BOOLEAN NOT NULL DEFAULT FALSE,
  data_yaml TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS ix_ag_dataset_versions_project_draft_created
  ON ag_dataset_versions (project_id, is_draft, created_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS ux_ag_dataset_versions_project_name_normalized
  ON ag_dataset_versions (project_id, lower(trim(name)));

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
  bbox_count INT NOT NULL DEFAULT 0,
  batch_name TEXT NOT NULL DEFAULT '',
  uploaded_by_email TEXT NOT NULL DEFAULT '',
  uploaded_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  in_dataset BOOLEAN NOT NULL DEFAULT FALSE,
  image_sha256 TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (version_id, split, stem)
);

CREATE INDEX IF NOT EXISTS ix_ag_dataset_images_vsplit
  ON ag_dataset_images (version_id, split);
CREATE INDEX IF NOT EXISTS ix_ag_dataset_images_vsplit_stem
  ON ag_dataset_images (version_id, split, stem);
CREATE INDEX IF NOT EXISTS ix_ag_dataset_images_vsplit_dims
  ON ag_dataset_images (version_id, split, width, height);
CREATE INDEX IF NOT EXISTS ix_ag_dataset_images_vsplit_bbox_count
  ON ag_dataset_images (version_id, split, bbox_count);
CREATE INDEX IF NOT EXISTS ix_ag_dataset_images_version_batch_uploaded
  ON ag_dataset_images (version_id, batch_name, uploaded_at DESC);
CREATE INDEX IF NOT EXISTS ix_ag_dataset_images_version_dataset_bbox
  ON ag_dataset_images (version_id, in_dataset, bbox_count);
CREATE INDEX IF NOT EXISTS ix_ag_dataset_images_version_dataset_sha
  ON ag_dataset_images (version_id, in_dataset, image_sha256);
