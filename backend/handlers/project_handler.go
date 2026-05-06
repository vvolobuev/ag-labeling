package handlers

import (
	"database/sql"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"my-app/middleware"
	"my-app/util"

	"github.com/gin-gonic/gin"
)

type ProjectHandler struct {
	DB          *sql.DB
	StorageRoot string
}

func (h *ProjectHandler) List(c *gin.Context) {
	wid := c.Param("id")
	if _, ok := RequireWorkspaceRole(c, h.DB, wid, "viewer"); !ok {
		return
	}
	q := strings.TrimSpace(c.Query("q"))
	var rows *sql.Rows
	var err error
	if q == "" {
		rows, err = h.DB.Query(`
			SELECT p.id::text, p.name, p.slug, p.is_public,
				EXTRACT(EPOCH FROM p.updated_at)::bigint, EXTRACT(EPOCH FROM p.created_at)::bigint,
				COALESCE(ic.cnt, 0)::int, COALESCE(ic.cover_id::text, '')
			FROM ag_projects p
			LEFT JOIN LATERAL (
				SELECT COUNT(*)::int AS cnt,
					(SELECT i.id FROM ag_dataset_images i
					 INNER JOIN ag_dataset_versions v ON v.id = i.version_id
					 WHERE v.project_id = p.id
					   AND v.is_draft = TRUE
					   AND i.in_dataset = TRUE
					 ORDER BY v.created_at DESC, i.split ASC, i.stem ASC
					 LIMIT 1) AS cover_id
				FROM ag_dataset_images i2
				INNER JOIN ag_dataset_versions v2 ON v2.id = i2.version_id
				WHERE v2.project_id = p.id
				  AND v2.is_draft = TRUE
				  AND i2.in_dataset = TRUE
			) ic ON true
			WHERE p.workspace_id = $1::uuid
			ORDER BY p.updated_at DESC
		`, wid)
	} else {
		pat := "%" + escapeILike(q) + "%"
		rows, err = h.DB.Query(`
			SELECT p.id::text, p.name, p.slug, p.is_public,
				EXTRACT(EPOCH FROM p.updated_at)::bigint, EXTRACT(EPOCH FROM p.created_at)::bigint,
				COALESCE(ic.cnt, 0)::int, COALESCE(ic.cover_id::text, '')
			FROM ag_projects p
			LEFT JOIN LATERAL (
				SELECT COUNT(*)::int AS cnt,
					(SELECT i.id FROM ag_dataset_images i
					 INNER JOIN ag_dataset_versions v ON v.id = i.version_id
					 WHERE v.project_id = p.id
					   AND v.is_draft = TRUE
					   AND i.in_dataset = TRUE
					 ORDER BY v.created_at DESC, i.split ASC, i.stem ASC
					 LIMIT 1) AS cover_id
				FROM ag_dataset_images i2
				INNER JOIN ag_dataset_versions v2 ON v2.id = i2.version_id
				WHERE v2.project_id = p.id
				  AND v2.is_draft = TRUE
				  AND i2.in_dataset = TRUE
			) ic ON true
			WHERE p.workspace_id = $1::uuid
			  AND (p.name ILIKE $2 OR p.slug ILIKE $2)
			ORDER BY p.updated_at DESC
		`, wid, pat)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	defer rows.Close()
	var out []gin.H
	for rows.Next() {
		var id, name, slug, coverID string
		var isPub bool
		var ts, createdAt int64
		var imgCount int
		if rows.Scan(&id, &name, &slug, &isPub, &ts, &createdAt, &imgCount, &coverID) != nil {
			continue
		}
		out = append(out, gin.H{
			"id": id, "name": name, "slug": slug,
			"is_public": isPub, "updated_at": ts, "created_at": createdAt,
			"image_count": imgCount, "cover_image_id": coverID,
		})
	}
	c.JSON(http.StatusOK, gin.H{"projects": out})
}

func (h *ProjectHandler) Create(c *gin.Context) {
	wid := c.Param("id")
	if _, ok := RequireWorkspaceRole(c, h.DB, wid, "admin"); !ok {
		return
	}
	var b struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&b); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid"})
		return
	}
	name := strings.TrimSpace(b.Name)
	slug := util.Slug(name)
	for i := 0; i < 20; i++ {
		var n int64
		_ = h.DB.QueryRow(`SELECT COUNT(*) FROM ag_projects WHERE workspace_id=$1::uuid AND slug=$2`, wid, slug).Scan(&n)
		if n == 0 {
			break
		}
		suf, _ := util.RandomHex(2)
		slug = util.Slug(name) + "-" + suf
	}
	var pid string
	err := h.DB.QueryRow(`
		INSERT INTO ag_projects (workspace_id, name, slug) VALUES ($1::uuid, $2, $3) RETURNING id::text
	`, wid, name, slug).Scan(&pid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": pid, "name": name, "slug": slug})
}

func (h *ProjectHandler) Get(c *gin.Context) {
	pid := c.Param("pid")
	if _, ok := RequireProjectViewerOrPublic(c, h.DB, pid); !ok {
		return
	}
	var wid, name, slug string
	var isPub bool
	var ts int64
	err := h.DB.QueryRow(`
		SELECT workspace_id::text, name, slug, is_public, EXTRACT(EPOCH FROM updated_at)::bigint
		FROM ag_projects WHERE id=$1::uuid
	`, pid).Scan(&wid, &name, &slug, &isPub, &ts)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	canEdit, _ := projectCanEditByUser(h.DB, middleware.UserID(c), pid)
	c.JSON(http.StatusOK, gin.H{
		"id": pid, "workspace_id": wid, "name": name, "slug": slug,
		"is_public": isPub, "updated_at": ts, "can_edit": canEdit,
	})
}

func (h *ProjectHandler) Patch(c *gin.Context) {
	pid := c.Param("pid")
	if ok := RequireProjectOwner(c, h.DB, pid); !ok {
		return
	}
	var b struct {
		IsPublic *bool `json:"is_public"`
	}
	if err := c.ShouldBindJSON(&b); err != nil || b.IsPublic == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid"})
		return
	}
	if _, err := h.DB.Exec(`UPDATE ag_projects SET is_public=$1, updated_at=now() WHERE id=$2::uuid`, *b.IsPublic, pid); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "is_public": *b.IsPublic})
}

func (h *ProjectHandler) Delete(c *gin.Context) {
	pid := c.Param("pid")
	if ok := RequireProjectOwner(c, h.DB, pid); !ok {
		return
	}
	tx, err := h.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	defer tx.Rollback()

	rows, err := tx.Query(`SELECT id::text FROM ag_dataset_versions WHERE project_id=$1::uuid`, pid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	versionIDs := make([]string, 0)
	for rows.Next() {
		var vid string
		if rows.Scan(&vid) == nil {
			versionIDs = append(versionIDs, vid)
		}
	}
	rows.Close()

	if _, err := tx.Exec(`
		DELETE FROM ag_dataset_images i
		USING ag_dataset_versions v
		WHERE i.version_id=v.id AND v.project_id=$1::uuid
	`, pid); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	if _, err := tx.Exec(`DELETE FROM ag_dataset_versions WHERE project_id=$1::uuid`, pid); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	res, err := tx.Exec(`DELETE FROM ag_projects WHERE id=$1::uuid`, pid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}

	for _, vid := range versionIDs {
		if strings.TrimSpace(vid) == "" {
			continue
		}
		_ = os.RemoveAll(filepath.Join(h.StorageRoot, vid))
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "deleted_project_id": pid})
}
