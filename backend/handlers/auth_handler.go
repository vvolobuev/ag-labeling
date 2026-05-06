package handlers

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"net/smtp"
	"os"
	"path/filepath"
	"strings"
	"time"

	"my-app/middleware"
	"my-app/util"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	DB          *sql.DB
	JWTSecret   string
	StorageRoot string
}

type registerBody struct {
	Email     string `json:"email" binding:"required"`
	Password  string `json:"password" binding:"required,min=8"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type loginBody struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) SignToken(userID string) (string, error) {
	claims := middleware.Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(168 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "alpha-guard-ai",
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString([]byte(h.JWTSecret))
}

func (h *AuthHandler) Register(c *gin.Context) {
	var b registerBody
	if err := c.ShouldBindJSON(&b); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	email := strings.ToLower(strings.TrimSpace(b.Email))
	firstName := strings.TrimSpace(b.FirstName)
	lastName := strings.TrimSpace(b.LastName)
	if firstName == "" {
		firstName = "User"
	}
	if lastName == "" {
		lastName = "Account"
	}
	if email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid email"})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(b.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "hash error"})
		return
	}
	tok, err := util.RandomHex(24)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token error"})
		return
	}

	var uid string
	err = h.DB.QueryRow(
		`INSERT INTO ag_users (email, password_hash, first_name, last_name, email_verified, verification_token)
		 VALUES ($1, $2, $3, $4, false, $5) RETURNING id`,
		email, string(hash), firstName, lastName, tok,
	).Scan(&uid)
	if err != nil {
		if strings.Contains(err.Error(), "unique") {
			c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}

	verifyURL := buildVerifyURL(tok)
	resp := gin.H{"message": "check email", "user_id": uid}
	if verifyURL != "" && !smtpConfigured() {
		resp["verification_url"] = verifyURL
		resp["hint"] = "SMTP not configured: open verification_url locally"
	} else if err := h.sendVerificationEmail(email, verifyURL); err != nil {
		log.Println("smtp:", err)
		if verifyURL != "" {
			resp["verification_url"] = verifyURL
		}
	}
	c.JSON(http.StatusCreated, resp)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var b loginBody
	if err := c.ShouldBindJSON(&b); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	email := strings.ToLower(strings.TrimSpace(b.Email))

	var uid, hash string
	var verified bool
	err := h.DB.QueryRow(
		`SELECT id::text, password_hash, email_verified FROM ag_users WHERE email=$1`,
		email,
	).Scan(&uid, &hash, &verified)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(b.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	if !verified {
		c.JSON(http.StatusForbidden, gin.H{"error": "email not verified", "verified": false})
		return
	}
	jwtStr, err := h.SignToken(uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "jwt error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": jwtStr})
}

func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	tok := strings.TrimSpace(c.Query("token"))
	if tok == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token required"})
		return
	}
	res, err := h.DB.Exec(
		`UPDATE ag_users SET email_verified=true, verification_token=NULL WHERE verification_token=$1 AND email_verified=false`,
		tok,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or expired token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *AuthHandler) Me(c *gin.Context) {
	uid := middleware.UserID(c)
	var email, firstName, lastName, avatarPath string
	var verified bool
	err := h.DB.QueryRow(`SELECT email, COALESCE(first_name, ''), COALESCE(last_name, ''), email_verified, COALESCE(avatar_path,'') FROM ag_users WHERE id=$1`, uid).
		Scan(&email, &firstName, &lastName, &verified, &avatarPath)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":         uid,
		"email":      email,
		"first_name": firstName,
		"last_name":  lastName,
		"verified":   verified,
		"avatar_url": func() string {
			if strings.TrimSpace(avatarPath) == "" {
				return ""
			}
			return "/api/me/avatar"
		}(),
	})
}

func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	uid := middleware.UserID(c)
	var b struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
	}
	if err := c.ShouldBindJSON(&b); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	first := strings.TrimSpace(b.FirstName)
	last := strings.TrimSpace(b.LastName)
	if first == "" || last == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "first_name and last_name are required"})
		return
	}
	if _, err := h.DB.Exec(`UPDATE ag_users SET first_name=$2, last_name=$3 WHERE id=$1::uuid`, uid, first, last); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "first_name": first, "last_name": last})
}

func (h *AuthHandler) UploadAvatar(c *gin.Context) {
	uid := middleware.UserID(c)
	fh, err := c.FormFile("avatar")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "avatar required"})
		return
	}
	if fh.Size > 10*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "avatar too large"})
		return
	}
	src, err := fh.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "avatar open failed"})
		return
	}
	defer src.Close()
	head := make([]byte, 512)
	n, _ := src.Read(head)
	contentType := http.DetectContentType(head[:n])
	if !strings.HasPrefix(contentType, "image/") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only image files are allowed"})
		return
	}
	exts, _ := mime.ExtensionsByType(contentType)
	ext := ".jpg"
	if len(exts) > 0 {
		ext = exts[0]
	}
	dir := filepath.Join(h.StorageRoot, "avatars")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "storage"})
		return
	}
	rel := filepath.ToSlash(filepath.Join("avatars", uid+ext))
	dstPath := filepath.Join(h.StorageRoot, filepath.FromSlash(rel))
	dst, err := os.Create(dstPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "storage"})
		return
	}
	defer dst.Close()
	if n > 0 {
		_, _ = dst.Write(head[:n])
	}
	_, _ = io.Copy(dst, src)
	if _, err := h.DB.Exec(`UPDATE ag_users SET avatar_path=$2 WHERE id=$1::uuid`, uid, rel); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "avatar_url": "/api/me/avatar"})
}

func (h *AuthHandler) Avatar(c *gin.Context) {
	uid := middleware.UserID(c)
	var rel string
	if err := h.DB.QueryRow(`SELECT COALESCE(avatar_path,'') FROM ag_users WHERE id=$1::uuid`, uid).Scan(&rel); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "avatar not found"})
		return
	}
	rel = strings.TrimSpace(rel)
	if rel == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "avatar not found"})
		return
	}
	fp := filepath.Join(h.StorageRoot, filepath.FromSlash(rel))
	stat, err := os.Stat(fp)
	if err != nil || stat.IsDir() {
		c.JSON(http.StatusNotFound, gin.H{"error": "avatar not found"})
		return
	}
	f, err := os.Open(fp)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "open failed"})
		return
	}
	defer f.Close()
	http.ServeContent(c.Writer, c.Request, filepath.Base(fp), stat.ModTime(), f)
}

func buildVerifyURL(token string) string {
	base := strings.TrimSuffix(os.Getenv("PUBLIC_APP_URL"), "/")
	if base == "" {
		return ""
	}
	return fmt.Sprintf("%s/verify?token=%s", base, token)
}

func smtpConfigured() bool {
	return os.Getenv("SMTP_HOST") != "" && os.Getenv("SMTP_FROM") != ""
}

func (h *AuthHandler) sendVerificationEmail(toEmail, verifyURL string) error {
	if !smtpConfigured() || verifyURL == "" {
		return nil
	}
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")
	if port == "" {
		port = "587"
	}
	from := os.Getenv("SMTP_FROM")
	user := os.Getenv("SMTP_USER")
	pass := os.Getenv("SMTP_PASSWORD")
	subject := "Alpha Guard AI - email verification"
	body := "Follow the link to verify your email:\n" + verifyURL
	msg := []byte(fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n%s",
		from, toEmail, subject, body))
	addr := fmt.Sprintf("%s:%s", host, port)
	var auth smtp.Auth
	if user != "" {
		auth = smtp.PlainAuth("", user, pass, host)
	}
	return smtp.SendMail(addr, auth, from, []string{toEmail}, msg)
}
