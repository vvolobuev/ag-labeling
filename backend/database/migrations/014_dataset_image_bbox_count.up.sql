ALTER TABLE ag_dataset_images ADD COLUMN IF NOT EXISTS bbox_count INT NOT NULL DEFAULT 0;

UPDATE ag_dataset_images AS t
SET bbox_count = (
  SELECT COUNT(*)::int
  FROM unnest(string_to_array(regexp_replace(trim(COALESCE(t.label_text,'')), E'\r', '', 'g'), E'\n')) AS line
  WHERE trim(line) <> ''
    AND left(trim(line),1) <> '#'
    AND trim(line) ~ '^[0-9]+[[:space:]]+[-+0-9.eE]+[[:space:]]+[-+0-9.eE]+[[:space:]]+[-+0-9.eE]+[[:space:]]+[-+0-9.eE]+[[:space:]]*$'
);

CREATE INDEX IF NOT EXISTS ix_ag_dataset_images_vsplit_stem ON ag_dataset_images (version_id, split, stem);
CREATE INDEX IF NOT EXISTS ix_ag_dataset_images_vsplit_dims ON ag_dataset_images (version_id, split, width, height);
CREATE INDEX IF NOT EXISTS ix_ag_dataset_images_vsplit_bbox_count ON ag_dataset_images (version_id, split, bbox_count);
