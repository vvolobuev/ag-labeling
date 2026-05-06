package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"my-app/middleware"

	"github.com/gin-gonic/gin"
)

func roleRank(r string) int {
	switch r {
	case "viewer":
		return 0
	case "annotator":
		return 1
	case "admin":
		return 2
	case "owner":
		return 3
	default:
		return -1
	}
}

func projectMemberRole(db *sql.DB, userID, projectID string) (role string, err error) {
	err = db.QueryRow(`
		SELECT wm.role FROM ag_projects p
		INNER JOIN ag_workspace_members wm ON wm.workspace_id = p.workspace_id AND wm.user_id = $1::uuid
		WHERE p.id = $2::uuid
	`, userID, projectID).Scan(&role)
	return role, err
}

func projectCanEditByUser(db *sql.DB, userID, projectID string) (bool, error) {
	uid := strings.TrimSpace(userID)
	if uid == "" {
		return false, nil
	}
	role, err := projectMemberRole(db, uid, projectID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return roleRank(role) >= roleRank("admin"), nil
}

func workspaceMemberRole(db *sql.DB, userID, workspaceID string) (role string, err error) {
	err = db.QueryRow(`
		SELECT role FROM ag_workspace_members WHERE user_id=$1::uuid AND workspace_id=$2::uuid
	`, userID, workspaceID).Scan(&role)
	return role, err
}

func RequireProjectRole(c *gin.Context, db *sql.DB, projectID, minRole string) (role string, ok bool) {
	uid := middleware.UserID(c)
	r, err := projectMemberRole(db, uid, projectID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusForbidden, gin.H{"error": "no access"})
			return "", false
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return "", false
	}
	if roleRank(r) < 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "no access"})
		return "", false
	}
	if roleRank(r) < roleRank(minRole) {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient role"})
		return "", false
	}
	return r, true
}

func RequireProjectOwner(c *gin.Context, db *sql.DB, projectID string) (ok bool) {
	uid := middleware.UserID(c)
	canEdit, err := projectCanEditByUser(db, uid, projectID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return false
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return false
	}
	if !canEdit {
		c.JSON(http.StatusForbidden, gin.H{"error": "owner access required"})
		return false
	}
	return true
}

func projectIsPublic(db *sql.DB, projectID string) (bool, error) {
	var pub bool
	err := db.QueryRow(`SELECT is_public FROM ag_projects WHERE id=$1::uuid`, projectID).Scan(&pub)
	return pub, err
}

func RequireProjectViewerOrPublic(c *gin.Context, db *sql.DB, projectID string) (memberRole string, ok bool) {
	uid := middleware.UserID(c)
	if strings.TrimSpace(uid) == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return "", false
	}
	r, err := projectMemberRole(db, uid, projectID)
	if err == nil && roleRank(r) >= roleRank("viewer") {
		return r, true
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return "", false
	}
	pub, err2 := projectIsPublic(db, projectID)
	if errors.Is(err2, sql.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return "", false
	}
	if err2 != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return "", false
	}
	if pub {
		return "public", true
	}
	c.JSON(http.StatusForbidden, gin.H{"error": "no access"})
	return "", false
}

func RequireWorkspaceRole(c *gin.Context, db *sql.DB, workspaceID, minRole string) (role string, ok bool) {
	uid := middleware.UserID(c)
	r, err := workspaceMemberRole(db, uid, workspaceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusForbidden, gin.H{"error": "no access"})
			return "", false
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return "", false
	}
	if roleRank(r) < 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "no access"})
		return "", false
	}
	if roleRank(r) < roleRank(minRole) {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient role"})
		return "", false
	}
	return r, true
}
