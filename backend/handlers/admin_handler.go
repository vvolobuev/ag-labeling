package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type AdminHandler struct {
	DB          *sql.DB
	StorageRoot string
	JWTSecret   string
}

type adminClaims struct {
	Admin bool `json:"admin"`
	jwt.RegisteredClaims
}

func (h *AdminHandler) adminLoginEnv() (string, string) {
	login := strings.TrimSpace(os.Getenv("ADMIN_LOGIN"))
	pass := os.Getenv("ADMIN_PASSWORD")
	if login == "" {
		login = "admin"
	}
	if pass == "" {
		pass = "123456"
	}
	return login, pass
}

func (h *AdminHandler) signAdminToken() (string, error) {
	claims := adminClaims{
		Admin: true,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "alpha-guard-admin",
			Subject:   "admin",
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString([]byte(h.JWTSecret))
}

func (h *AdminHandler) requireAdmin(c *gin.Context) bool {
	hdr := strings.TrimSpace(c.GetHeader("Authorization"))
	if !strings.HasPrefix(strings.ToLower(hdr), "bearer ") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "admin auth required"})
		return false
	}
	tok := strings.TrimSpace(hdr[7:])
	if tok == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "admin auth required"})
		return false
	}
	claims := &adminClaims{}
	parsed, err := jwt.ParseWithClaims(tok, claims, func(t *jwt.Token) (any, error) {
		return []byte(h.JWTSecret), nil
	})
	if err != nil || !parsed.Valid || !claims.Admin {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid admin token"})
		return false
	}
	return true
}

func (h *AdminHandler) Login(c *gin.Context) {
	var b struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&b); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	envLogin, envPass := h.adminLoginEnv()
	if strings.TrimSpace(b.Login) != envLogin || b.Password != envPass {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid admin credentials"})
		return
	}
	tok, err := h.signAdminToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": tok})
}

func bytesToGB(n int64) float64 {
	if n <= 0 {
		return 0
	}
	return float64(n) / (1024.0 * 1024.0 * 1024.0)
}

func dirSizeBytes(root string) int64 {
	var total int64
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d == nil || d.IsDir() {
			return nil
		}
		st, er := d.Info()
		if er != nil {
			return nil
		}
		total += st.Size()
		return nil
	})
	return total
}

func fsTotalBytes(root string) int64 {
	var st syscall.Statfs_t
	if err := syscall.Statfs(root, &st); err != nil {
		return 0
	}
	return int64(st.Blocks) * int64(st.Bsize)
}

func (h *AdminHandler) Overview(c *gin.Context) {
	if !h.requireAdmin(c) {
		return
	}

	var totalImages int64
	if err := h.DB.QueryRow(`SELECT COUNT(*) FROM ag_dataset_images`).Scan(&totalImages); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}

	users := make([]gin.H, 0)
	rows, err := h.DB.Query(`
		SELECT
			u.id::text,
			u.email,
			COALESCE(u.first_name,''),
			COALESCE(u.last_name,''),
			COALESCE(ws.workspace_count, 0),
			COALESCE(ps.project_count, 0),
			COALESCE(isum.image_count, 0)
		FROM ag_users u
		LEFT JOIN (
			SELECT wm.user_id, COUNT(DISTINCT wm.workspace_id)::bigint AS workspace_count
			FROM ag_workspace_members wm
			GROUP BY wm.user_id
		) ws ON ws.user_id = u.id
		LEFT JOIN (
			SELECT wm.user_id, COUNT(DISTINCT p.id)::bigint AS project_count
			FROM ag_workspace_members wm
			INNER JOIN ag_projects p ON p.workspace_id = wm.workspace_id
			GROUP BY wm.user_id
		) ps ON ps.user_id = u.id
		LEFT JOIN (
			SELECT wm.user_id, COUNT(i.id)::bigint AS image_count
			FROM ag_workspace_members wm
			INNER JOIN ag_projects p ON p.workspace_id = wm.workspace_id
			INNER JOIN ag_dataset_versions v ON v.project_id = p.id
			INNER JOIN ag_dataset_images i ON i.version_id = v.id
			GROUP BY wm.user_id
		) isum ON isum.user_id = u.id
		ORDER BY u.created_at DESC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	for rows.Next() {
		var id, email, first, last string
		var wsCount, prCount, imgCount int64
		if rows.Scan(&id, &email, &first, &last, &wsCount, &prCount, &imgCount) != nil {
			continue
		}
		users = append(users, gin.H{
			"id":              id,
			"email":           email,
			"first_name":      first,
			"last_name":       last,
			"workspace_count": wsCount,
			"project_count":   prCount,
			"image_count":     imgCount,
		})
	}
	rows.Close()

	workspaces := make([]gin.H, 0)
	wRows, err := h.DB.Query(`
		SELECT
			w.id::text,
			w.name,
			COALESCE(owner.email, ''),
			COALESCE(prj.project_count, 0),
			COALESCE(img.image_count, 0)
		FROM ag_workspaces w
		LEFT JOIN ag_users owner ON owner.id = w.created_by
		LEFT JOIN (
			SELECT p.workspace_id, COUNT(*)::bigint AS project_count
			FROM ag_projects p
			GROUP BY p.workspace_id
		) prj ON prj.workspace_id = w.id
		LEFT JOIN (
			SELECT p.workspace_id, COUNT(i.id)::bigint AS image_count
			FROM ag_projects p
			INNER JOIN ag_dataset_versions v ON v.project_id = p.id
			INNER JOIN ag_dataset_images i ON i.version_id = v.id
			GROUP BY p.workspace_id
		) img ON img.workspace_id = w.id
		ORDER BY w.created_at DESC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	for wRows.Next() {
		var id, name, owner string
		var prCount, imgCount int64
		if wRows.Scan(&id, &name, &owner, &prCount, &imgCount) != nil {
			continue
		}
		workspaces = append(workspaces, gin.H{
			"id":            id,
			"name":          name,
			"owner_email":   owner,
			"project_count": prCount,
			"image_count":   imgCount,
		})
	}
	wRows.Close()

	projects := make([]gin.H, 0)
	pRows, err := h.DB.Query(`
		SELECT
			p.id::text,
			p.name,
			p.workspace_id::text,
			COALESCE(w.name, ''),
			COALESCE(COUNT(i.id), 0)::bigint
		FROM ag_projects p
		LEFT JOIN ag_workspaces w ON w.id = p.workspace_id
		LEFT JOIN ag_dataset_versions v ON v.project_id = p.id
		LEFT JOIN ag_dataset_images i ON i.version_id = v.id
		GROUP BY p.id, p.name, p.workspace_id, w.name, p.updated_at
		ORDER BY p.updated_at DESC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	for pRows.Next() {
		var id, name, wsID, wsName string
		var imgCount int64
		if pRows.Scan(&id, &name, &wsID, &wsName, &imgCount) != nil {
			continue
		}
		projects = append(projects, gin.H{
			"id":             id,
			"name":           name,
			"workspace_id":   wsID,
			"workspace_name": wsName,
			"image_count":    imgCount,
		})
	}
	pRows.Close()

	usedBytes := dirSizeBytes(h.StorageRoot)
	totalBytes := fsTotalBytes(h.StorageRoot)
	usagePct := 0.0
	if totalBytes > 0 {
		usagePct = (float64(usedBytes) * 100.0) / float64(totalBytes)
	}

	c.JSON(http.StatusOK, gin.H{
		"disk": gin.H{
			"used_bytes":  usedBytes,
			"total_bytes": totalBytes,
			"used_gb":     bytesToGB(usedBytes),
			"total_gb":    bytesToGB(totalBytes),
			"used_pct":    usagePct,
		},
		"totals": gin.H{
			"images": totalImages,
			"users":  len(users),
		},
		"users":      users,
		"workspaces": workspaces,
		"projects":   projects,
	})
}

func (h *AdminHandler) DeleteWorkspace(c *gin.Context) {
	if !h.requireAdmin(c) {
		return
	}
	wid := c.Param("id")
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
	imagePaths := make([]string, 0)
	pRows, err := tx.Query(`
		SELECT v.id::text, i.rel_image_path
		FROM ag_dataset_images i
		INNER JOIN ag_dataset_versions v ON v.id=i.version_id
		INNER JOIN ag_projects p ON p.id=v.project_id
		WHERE p.workspace_id=$1::uuid
	`, wid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	for pRows.Next() {
		var vid, rel string
		if pRows.Scan(&vid, &rel) == nil {
			fp := filepath.Join(h.StorageRoot, vid, filepath.FromSlash(strings.ReplaceAll(rel, `\`, `/`)))
			imagePaths = append(imagePaths, fp)
		}
	}
	pRows.Close()

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
	for _, fp := range imagePaths {
		if strings.TrimSpace(fp) == "" {
			continue
		}
		if err := os.Remove(fp); err != nil && !os.IsNotExist(err) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("storage cleanup failed: %v", err)})
			return
		}
	}
	for _, vid := range versionIDs {
		if strings.TrimSpace(vid) == "" {
			continue
		}
		if err := os.RemoveAll(filepath.Join(h.StorageRoot, vid)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("storage cleanup failed: %v", err)})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *AdminHandler) DeleteProject(c *gin.Context) {
	if !h.requireAdmin(c) {
		return
	}
	pid := c.Param("pid")
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
	imagePaths := make([]string, 0)
	pRows, err := tx.Query(`
		SELECT v.id::text, i.rel_image_path
		FROM ag_dataset_images i
		INNER JOIN ag_dataset_versions v ON v.id=i.version_id
		WHERE v.project_id=$1::uuid
	`, pid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	for pRows.Next() {
		var vid, rel string
		if pRows.Scan(&vid, &rel) == nil {
			fp := filepath.Join(h.StorageRoot, vid, filepath.FromSlash(strings.ReplaceAll(rel, `\`, `/`)))
			imagePaths = append(imagePaths, fp)
		}
	}
	pRows.Close()

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
	for _, fp := range imagePaths {
		if strings.TrimSpace(fp) == "" {
			continue
		}
		if err := os.Remove(fp); err != nil && !os.IsNotExist(err) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("storage cleanup failed: %v", err)})
			return
		}
	}
	for _, vid := range versionIDs {
		if strings.TrimSpace(vid) == "" {
			continue
		}
		if err := os.RemoveAll(filepath.Join(h.StorageRoot, vid)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("storage cleanup failed: %v", err)})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
