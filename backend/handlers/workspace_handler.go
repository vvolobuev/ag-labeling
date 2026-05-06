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

type WorkspaceHandler struct {
	DB          *sql.DB
	StorageRoot string
}

type workspaceDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
	Role string `json:"role"`
}

func (h *WorkspaceHandler) List(c *gin.Context) {
	uid := middleware.UserID(c)
	rows, err := h.DB.Query(`
		SELECT w.id::text, w.name, w.slug, wm.role
		FROM ag_workspaces w
		INNER JOIN ag_workspace_members wm ON wm.workspace_id = w.id AND wm.user_id = $1::uuid
		ORDER BY w.created_at DESC
	`, uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	defer rows.Close()
	var list []workspaceDTO
	for rows.Next() {
		var w workspaceDTO
		if err := rows.Scan(&w.ID, &w.Name, &w.Slug, &w.Role); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
			return
		}
		list = append(list, w)
	}
	c.JSON(http.StatusOK, gin.H{"workspaces": list})
}

func (h *WorkspaceHandler) Create(c *gin.Context) {
	uid := middleware.UserID(c)
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
		_ = h.DB.QueryRow(`SELECT COUNT(*) FROM ag_workspaces WHERE slug=$1`, slug).Scan(&n)
		if n == 0 {
			break
		}
		suf, _ := util.RandomHex(2)
		slug = util.Slug(name) + "-" + suf
	}
	tx, err := h.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	defer tx.Rollback()

	var wid string
	err = tx.QueryRow(`
		INSERT INTO ag_workspaces (name, slug, created_by) VALUES ($1, $2, $3::uuid) RETURNING id::text
	`, name, slug, uid).Scan(&wid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	_, err = tx.Exec(`
		INSERT INTO ag_workspace_members (workspace_id, user_id, role) VALUES ($1::uuid, $2::uuid, 'owner')
	`, wid, uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": wid, "name": name, "slug": slug, "role": "owner"})
}

func (h *WorkspaceHandler) Get(c *gin.Context) {
	wid := c.Param("id")
	if _, ok := RequireWorkspaceRole(c, h.DB, wid, "viewer"); !ok {
		return
	}
	var name, slug string
	err := h.DB.QueryRow(`SELECT name, slug FROM ag_workspaces WHERE id=$1::uuid`, wid).Scan(&name, &slug)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": wid, "name": name, "slug": slug})
}

type memberDTO struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

func (h *WorkspaceHandler) ListMembers(c *gin.Context) {
	wid := c.Param("id")
	if _, ok := RequireWorkspaceRole(c, h.DB, wid, "viewer"); !ok {
		return
	}
	rows, err := h.DB.Query(`
		SELECT u.email, wm.role FROM ag_workspace_members wm
		INNER JOIN ag_users u ON u.id = wm.user_id
		WHERE wm.workspace_id = $1::uuid
		ORDER BY wm.role DESC, u.email
	`, wid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	defer rows.Close()
	var ms []memberDTO
	for rows.Next() {
		var m memberDTO
		if rows.Scan(&m.Email, &m.Role) != nil {
			continue
		}
		ms = append(ms, m)
	}
	c.JSON(http.StatusOK, gin.H{"members": ms})
}

func (h *WorkspaceHandler) AddMember(c *gin.Context) {
	wid := c.Param("id")
	if _, ok := RequireWorkspaceRole(c, h.DB, wid, "admin"); !ok {
		return
	}
	var b struct {
		Email string `json:"email" binding:"required"`
		Role  string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&b); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid"})
		return
	}
	email := strings.ToLower(strings.TrimSpace(b.Email))
	role := strings.ToLower(strings.TrimSpace(b.Role))
	if role != "admin" && role != "annotator" && role != "viewer" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad role"})
		return
	}
	var newUID string
	err := h.DB.QueryRow(`SELECT id::text FROM ag_users WHERE email=$1 AND email_verified=true`, email).Scan(&newUID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found or not verified"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	_, err = h.DB.Exec(`
		INSERT INTO ag_workspace_members (workspace_id, user_id, role) VALUES ($1::uuid, $2::uuid, $3)
		ON CONFLICT (workspace_id, user_id) DO UPDATE SET role = EXCLUDED.role
	`, wid, newUID, role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *WorkspaceHandler) Delete(c *gin.Context) {
	wid := c.Param("id")
	if _, ok := RequireWorkspaceRole(c, h.DB, wid, "owner"); !ok {
		return
	}
	tx, err := h.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	defer tx.Rollback()

	rows, err := tx.Query(`
		SELECT v.id::text
		FROM ag_dataset_versions v
		INNER JOIN ag_projects p ON p.id=v.project_id
		WHERE p.workspace_id=$1::uuid
	`, wid)
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
		USING ag_dataset_versions v, ag_projects p
		WHERE i.version_id=v.id
		  AND v.project_id=p.id
		  AND p.workspace_id=$1::uuid
	`, wid); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	if _, err := tx.Exec(`
		DELETE FROM ag_dataset_versions v
		USING ag_projects p
		WHERE v.project_id=p.id
		  AND p.workspace_id=$1::uuid
	`, wid); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	if _, err := tx.Exec(`DELETE FROM ag_projects WHERE workspace_id=$1::uuid`, wid); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	if _, err := tx.Exec(`DELETE FROM ag_workspace_members WHERE workspace_id=$1::uuid`, wid); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	res, err := tx.Exec(`DELETE FROM ag_workspaces WHERE id=$1::uuid`, wid)
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
	c.JSON(http.StatusOK, gin.H{"ok": true, "deleted_workspace_id": wid})
}
