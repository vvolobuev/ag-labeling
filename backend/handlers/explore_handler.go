package handlers

import (
	"database/sql"
	"net/http"
	"strings"

	"my-app/middleware"

	"github.com/gin-gonic/gin"
)

type ExploreHandler struct {
	DB *sql.DB
}

func escapeILike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

func (h *ExploreHandler) ListPublicProjects(c *gin.Context) {
	if strings.TrimSpace(middleware.UserID(c)) == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	q := strings.TrimSpace(c.Query("q"))
	var rows *sql.Rows
	var err error
	if q == "" {
		rows, err = h.DB.Query(`
			SELECT p.id::text, p.name, p.slug, w.id::text, w.name,
				p.is_public, EXTRACT(EPOCH FROM p.updated_at)::bigint,
				COALESCE(ic.cnt, 0)::int,
				COALESCE(ic.cover_id::text, '')
			FROM ag_projects p
			INNER JOIN ag_workspaces w ON w.id = p.workspace_id
			LEFT JOIN LATERAL (
				SELECT COUNT(*)::int AS cnt,
					(SELECT i.id FROM ag_dataset_images i
					 INNER JOIN ag_dataset_versions v ON v.id = i.version_id
					 WHERE v.project_id = p.id
					 ORDER BY v.created_at DESC, i.split ASC, i.stem ASC
					 LIMIT 1) AS cover_id
				FROM ag_dataset_images i2
				INNER JOIN ag_dataset_versions v2 ON v2.id = i2.version_id
				WHERE v2.project_id = p.id
			) ic ON true
			WHERE p.is_public = TRUE
			ORDER BY p.updated_at DESC
			LIMIT 80
		`)
	} else {
		pat := "%" + escapeILike(q) + "%"
		rows, err = h.DB.Query(`
			SELECT p.id::text, p.name, p.slug, w.id::text, w.name,
				p.is_public, EXTRACT(EPOCH FROM p.updated_at)::bigint,
				COALESCE(ic.cnt, 0)::int,
				COALESCE(ic.cover_id::text, '')
			FROM ag_projects p
			INNER JOIN ag_workspaces w ON w.id = p.workspace_id
			LEFT JOIN LATERAL (
				SELECT COUNT(*)::int AS cnt,
					(SELECT i.id FROM ag_dataset_images i
					 INNER JOIN ag_dataset_versions v ON v.id = i.version_id
					 WHERE v.project_id = p.id
					 ORDER BY v.created_at DESC, i.split ASC, i.stem ASC
					 LIMIT 1) AS cover_id
				FROM ag_dataset_images i2
				INNER JOIN ag_dataset_versions v2 ON v2.id = i2.version_id
				WHERE v2.project_id = p.id
			) ic ON true
			WHERE p.is_public = TRUE
			  AND (p.name ILIKE $1 OR p.slug ILIKE $1 OR w.name ILIKE $1)
			ORDER BY p.updated_at DESC
			LIMIT 80
		`, pat)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	defer rows.Close()
	var list []gin.H
	for rows.Next() {
		var id, name, slug, wid, wname string
		var isPub bool
		var ts int64
		var imgCount int
		var coverID string
		if err := rows.Scan(&id, &name, &slug, &wid, &wname, &isPub, &ts, &imgCount, &coverID); err != nil {
			continue
		}
		list = append(list, gin.H{
			"id": id, "name": name, "slug": slug,
			"workspace_id": wid, "workspace_name": wname,
			"is_public": isPub, "updated_at": ts,
			"image_count": imgCount, "cover_image_id": coverID,
		})
	}
	c.JSON(http.StatusOK, gin.H{"projects": list})
}
