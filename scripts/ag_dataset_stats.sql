-- Alpha Guard — сводка по проектам, версиям датасетов и строкам ag_dataset_images
-- Запуск из корня репозитория после source backend/.env:
--   cd /path/to/alpha-guard-ai && set -a && source backend/.env && set +a && PGCONNECT_TIMEOUT=30 psql "postgresql://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=${DB_SSLMODE}" -f scripts/ag_dataset_stats.sql

\pset linestyle unicode
\pset border 2

SELECT
  (SELECT COUNT(*) FROM ag_projects) AS projects_total,
  (SELECT COUNT(*) FROM ag_dataset_versions) AS dataset_versions_total,
  (SELECT COUNT(*) FROM ag_dataset_images) AS image_rows_total;

SELECT
  p.id::text AS project_id,
  p.name AS project_name,
  COUNT(DISTINCT v.id) AS versions_count,
  COUNT(i.id) AS image_rows
FROM ag_projects p
LEFT JOIN ag_dataset_versions v ON v.project_id = p.id
LEFT JOIN ag_dataset_images i ON i.version_id = v.id
GROUP BY p.id, p.name
ORDER BY project_name NULLS LAST;

SELECT
  v.id::text AS version_id,
  p.name AS project_name,
  v.name AS version_name,
  COUNT(i.id) AS images_count
FROM ag_dataset_versions v
JOIN ag_projects p ON p.id = v.project_id
LEFT JOIN ag_dataset_images i ON i.version_id = v.id
GROUP BY v.id, p.name, v.name
ORDER BY p.name, v.name;

SELECT
  COUNT(*)::int AS versions_total,
  SUM(CASE WHEN c > 0 THEN 1 ELSE 0 END)::int AS versions_with_images,
  SUM(CASE WHEN c = 0 THEN 1 ELSE 0 END)::int AS versions_without_images
FROM (
  SELECT v.id, COUNT(i.id)::bigint AS c
  FROM ag_dataset_versions v
  LEFT JOIN ag_dataset_images i ON i.version_id = v.id
  GROUP BY v.id
) t;
