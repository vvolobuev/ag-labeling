-- Одно отображаемое имя версии на проект (без учёта регистра и краевых пробелов).
CREATE UNIQUE INDEX IF NOT EXISTS ux_ag_dataset_versions_project_name_normalized
ON ag_dataset_versions (project_id, lower(trim(name)));
