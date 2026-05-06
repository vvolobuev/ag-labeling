-- Project visibility and activity timestamp (dataset cards, Explore).
ALTER TABLE ag_projects ADD COLUMN IF NOT EXISTS is_public BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE ag_projects ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

CREATE INDEX IF NOT EXISTS ix_ag_projects_workspace_updated ON ag_projects (workspace_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS ix_ag_projects_public_updated ON ag_projects (is_public, updated_at DESC) WHERE is_public = TRUE;

-- Backfill updated_at from latest version or project creation.
UPDATE ag_projects p
SET updated_at = COALESCE(
  (SELECT MAX(v.created_at) FROM ag_dataset_versions v WHERE v.project_id = p.id),
  p.created_at
);
