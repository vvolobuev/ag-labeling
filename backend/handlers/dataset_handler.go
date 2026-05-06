package handlers

import (
	"archive/zip"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"my-app/middleware"
	"my-app/ylabel"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"gopkg.in/yaml.v3"
)

type DatasetHandler struct {
	DB          *sql.DB
	StorageRoot string
}

func (h *DatasetHandler) ensureDraftVersion(pid string) (string, error) {
	var vid string
	err := h.DB.QueryRow(`
		SELECT id::text FROM ag_dataset_versions
		WHERE project_id=$1::uuid AND is_draft=TRUE
		ORDER BY created_at ASC
		LIMIT 1
	`, pid).Scan(&vid)
	if err == nil {
		return vid, nil
	}
	if err != sql.ErrNoRows {
		return "", err
	}
	yamlSeed := "nc: 0\nnames: []\n"
	if err := h.DB.QueryRow(`
		INSERT INTO ag_dataset_versions (project_id, name, data_yaml, is_draft)
		VALUES ($1::uuid, $2, $3, TRUE)
		RETURNING id::text
	`, pid, "__dataset__", yamlSeed).Scan(&vid); err != nil {
		return "", err
	}
	vRoot := filepath.Join(h.StorageRoot, vid)
	if mkErr := os.MkdirAll(vRoot, 0o755); mkErr != nil {
		return "", mkErr
	}
	return vid, nil
}

func batchFallbackName(ts time.Time) string {
	return "Uploaded on " + ts.Format("01/02/06 at 3:04 pm")
}

func slugForPath(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "batch"
	}
	return out
}

func isSupportedImageExt(ext string) bool {
	switch strings.ToLower(strings.TrimPrefix(strings.TrimSpace(ext), ".")) {
	case "jpg", "jpeg", "png", "bmp", "webp", "avif":
		return true
	default:
		return false
	}
}

func fileSHA256Hex(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func (h *DatasetHandler) userEmailByID(uid string) string {
	if strings.TrimSpace(uid) == "" {
		return ""
	}
	var email string
	_ = h.DB.QueryRow(`SELECT email FROM ag_users WHERE id=$1::uuid`, uid).Scan(&email)
	return email
}

func (h *DatasetHandler) importStagingDir() (string, error) {
	d := filepath.Join(h.StorageRoot, ".import-tmp")
	if err := os.MkdirAll(d, 0o755); err != nil {
		return "", err
	}
	return d, nil
}

func (h *DatasetHandler) datasetVersionNameExists(pid, displayName string) (bool, error) {
	var exists bool
	err := h.DB.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM ag_dataset_versions
			WHERE project_id = $1::uuid AND lower(trim(name)) = lower(trim($2))
		)
	`, pid, displayName).Scan(&exists)
	return exists, err
}

func versionNameProjectToken(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "project"
	}
	var b strings.Builder
	lastUnderscore := false
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore {
			b.WriteByte('_')
			lastUnderscore = true
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		return "project"
	}
	return out
}

func versionNumberFromName(name string) int {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0
	}
	i := strings.LastIndex(name, "_V")
	if i < 0 {
		return 0
	}
	start := i + 2
	end := start
	for end < len(name) && name[end] >= '0' && name[end] <= '9' {
		end++
	}
	if end == start {
		return 0
	}
	n, err := strconv.Atoi(name[start:end])
	if err != nil || n <= 0 {
		return 0
	}
	return n
}

func namesFromSourceForVersion(rawYAML string, src []datasetSourceImage) []string {
	names := yamlClassNamesSlice(rawYAML)
	if len(names) > 0 {
		return names
	}
	maxClass := -1
	for _, it := range src {
		for _, b := range ylabel.CompactBBoxes(it.Label, 1<<20) {
			ci := int(b[0])
			if ci > maxClass {
				maxClass = ci
			}
		}
	}
	if maxClass < 0 {
		return []string{}
	}
	out := make([]string, maxClass+1)
	for i := range out {
		out[i] = fmt.Sprintf("class_%d", i)
	}
	return out
}

func buildDatasetYAMLText(classNames []string, resize int, keepOriginal bool) string {
	quoted := make([]string, 0, len(classNames))
	for _, n := range classNames {
		s := strings.TrimSpace(n)
		if s == "" {
			s = fmt.Sprintf("class_%d", len(quoted))
		}
		s = strings.ReplaceAll(s, `'`, `''`)
		quoted = append(quoted, "'"+s+"'")
	}
	y := fmt.Sprintf(
		"train: ../train/images\nval: ../valid/images\ntest: ../test/images\n\nnc: %d\nnames: [%s]\n",
		len(classNames),
		strings.Join(quoted, ", "),
	)
	if keepOriginal {
		y += "resize: original\n"
	} else if resize > 0 {
		y += fmt.Sprintf("resize: %d\n", resize)
	}
	return y
}

func (h *DatasetHandler) allocateAutoVersionName(pid string) (string, error) {

	if _, err := h.DB.Exec(`ALTER TABLE IF EXISTS ag_projects ADD COLUMN IF NOT EXISTS version_counter INT NOT NULL DEFAULT 0`); err != nil {
		return "", err
	}
	var projectName string
	var versionN int
	if err := h.DB.QueryRow(`
		WITH max_existing AS (
			SELECT COALESCE(MAX((regexp_match(name, '_V([0-9]+)_'))[1]::int), 0) AS mx
			FROM ag_dataset_versions
			WHERE project_id=$1::uuid AND is_draft=FALSE
		)
		UPDATE ag_projects p
		SET version_counter = GREATEST(COALESCE(p.version_counter, 0), (SELECT mx FROM max_existing)) + 1
		WHERE p.id=$1::uuid
		RETURNING p.name, p.version_counter
	`, pid).Scan(&projectName, &versionN); err != nil {
		return "", err
	}
	projectToken := versionNameProjectToken(projectName)
	dateToken := time.Now().Format("20060102")
	for {
		candidate := fmt.Sprintf("%s_V%d_%s", projectToken, versionN, dateToken)
		taken, err := h.datasetVersionNameExists(pid, candidate)
		if err != nil {
			return "", err
		}
		if !taken {
			return candidate, nil
		}
		versionN++
	}
}

func (h *DatasetHandler) allocateImportVersionName(c *gin.Context, pid, postedName string) (string, bool) {
	pref := strings.TrimSpace(postedName)
	if pref != "" {
		taken, err := h.datasetVersionNameExists(pid, pref)
		if err != nil {
			log.Printf("datasetVersionNameExists: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
			return "", false
		}
		if taken {
			c.JSON(http.StatusConflict, gin.H{
				"error": "Версия датасета с таким именем уже есть в этом проекте",
				"code":  "duplicate_version_name",
				"name":  pref,
				"hint":  "Задайте другое имя или оставьте поле пустым — тогда имя сгенерируется автоматически.",
			})
			return "", false
		}
		return pref, true
	}
	auto, err := h.allocateAutoVersionName(pid)
	if err != nil {
		log.Printf("allocateAutoVersionName: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return "", false
	}
	return auto, true
}

func sanitizeZipPath(destRoot, zipName string) (string, bool) {
	name := filepath.ToSlash(zipName)
	name = strings.TrimPrefix(name, "/")
	for _, seg := range strings.Split(name, "/") {
		if seg == ".." {
			return "", false
		}
	}
	out := filepath.Join(destRoot, filepath.FromSlash(name))
	co := filepath.Clean(out)
	r, err := filepath.Rel(filepath.Clean(destRoot), co)
	if err != nil || strings.HasPrefix(r, "..") {
		return "", false
	}
	return co, true
}

func (h *DatasetHandler) unzipToDir(zipPath, destDir string) error {
	t0 := time.Now()
	if st, err := os.Stat(zipPath); err == nil {
		log.Printf("[alpha-guard import] unzip: файл %s, размер %.2f MiB", filepath.Base(zipPath), float64(st.Size())/(1024*1024))
	}
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer zr.Close()
	nEntries := len(zr.File)
	log.Printf("[alpha-guard import] unzip: записей в архиве=%d, распаковка в %s", nEntries, destDir)
	lastLog := time.Now()
	written := 0
	for _, zf := range zr.File {
		path, ok := sanitizeZipPath(destDir, zf.Name)
		if !ok {
			continue
		}
		mode := zf.Mode()
		if mode.IsDir() || strings.HasSuffix(zf.Name, "/") {
			_ = os.MkdirAll(path, 0o755)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		rc, err := zf.Open()
		if err != nil {
			return err
		}
		fw, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			rc.Close()
			return err
		}
		_, err = io.Copy(fw, rc)
		_ = rc.Close()
		_ = fw.Close()
		if err != nil {
			return err
		}
		written++
		if written%800 == 0 || time.Since(lastLog) > 12*time.Second {
			log.Printf("[alpha-guard import] unzip: извлечено файлов %d (~из %d записей архива), прошло %v", written, nEntries, time.Since(t0))
			lastLog = time.Now()
		}
	}
	log.Printf("[alpha-guard import] unzip: готово — извлечено %d файлов за %v (записей в каталоге=%d)", written, time.Since(t0), nEntries)
	return nil
}

func (h *DatasetHandler) ListVersions(c *gin.Context) {
	pid := c.Param("pid")
	if _, ok := RequireProjectViewerOrPublic(c, h.DB, pid); !ok {
		return
	}
	rows, err := h.DB.Query(`
		SELECT id::text, name, EXTRACT(epoch FROM created_at)::bigint FROM ag_dataset_versions
		WHERE project_id=$1::uuid AND is_draft=FALSE ORDER BY created_at DESC
	`, pid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	defer rows.Close()
	var list []gin.H
	for rows.Next() {
		var id, name string
		var ts int64
		if rows.Scan(&id, &name, &ts) != nil {
			continue
		}
		list = append(list, gin.H{"id": id, "name": name, "created_at": ts})
	}
	c.JSON(http.StatusOK, gin.H{"versions": list})
}

func (h *DatasetHandler) DeleteVersion(c *gin.Context) {
	vid := c.Param("vid")
	var pid, vname string
	err := h.DB.QueryRow(`SELECT project_id::text, name FROM ag_dataset_versions WHERE id=$1::uuid`, vid).Scan(&pid, &vname)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	if ok := RequireProjectOwner(c, h.DB, pid); !ok {
		return
	}
	if vn := versionNumberFromName(vname); vn > 0 {
		_, _ = h.DB.Exec(`UPDATE ag_projects SET version_counter = GREATEST(COALESCE(version_counter,0), $2) WHERE id=$1::uuid`, pid, vn)
	}
	vRoot := filepath.Join(h.StorageRoot, vid)
	if err := os.RemoveAll(vRoot); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "storage"})
		return
	}
	res, err := h.DB.Exec(`DELETE FROM ag_dataset_versions WHERE id=$1::uuid`, vid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *DatasetHandler) VersionMeta(c *gin.Context) {
	vid := c.Param("vid")
	var pid, name string
	var yaml sql.NullString
	err := h.DB.QueryRow(`
		SELECT project_id::text, name, data_yaml FROM ag_dataset_versions WHERE id=$1::uuid
	`, vid).Scan(&pid, &name, &yaml)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	if _, ok := RequireProjectViewerOrPublic(c, h.DB, pid); !ok {
		return
	}
	y := ""
	if yaml.Valid {
		y = yaml.String
	}
	c.JSON(http.StatusOK, gin.H{"id": vid, "project_id": pid, "name": name, "data_yaml": y})
}

func (h *DatasetHandler) PatchVersionNames(c *gin.Context) {
	vid := c.Param("vid")
	var pid string
	var yamlNS sql.NullString
	err := h.DB.QueryRow(`SELECT project_id::text, data_yaml FROM ag_dataset_versions WHERE id=$1::uuid`, vid).Scan(&pid, &yamlNS)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	if ok := RequireProjectOwner(c, h.DB, pid); !ok {
		return
	}
	var body struct {
		Names []string `json:"names"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Names == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid"})
		return
	}
	var doc map[string]any
	raw := ""
	if yamlNS.Valid {
		raw = yamlNS.String
	}
	raw = strings.TrimSpace(raw)
	if raw != "" {
		if er := yaml.Unmarshal([]byte(raw), &doc); er != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "yaml parse"})
			return
		}
	}
	if doc == nil {
		doc = map[string]any{}
	}
	clean := make([]string, 0, len(body.Names))
	for _, n := range body.Names {
		s := strings.TrimSpace(n)
		if s != "" {
			clean = append(clean, s)
		}
	}
	doc["nc"] = len(clean)
	doc["names"] = clean
	out, er := yaml.Marshal(doc)
	if er != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "yaml encode"})
		return
	}
	if _, err := h.DB.Exec(`UPDATE ag_dataset_versions SET data_yaml=$1 WHERE id=$2::uuid`, string(out), vid); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data_yaml": strings.TrimSuffix(string(out), "\n")})
}

func (h *DatasetHandler) GetImageJSON(c *gin.Context) {
	iid := c.Param("imgid")
	uid := middleware.UserID(c)
	var vid, pid, stem, ext, rel, lbl, split string
	var isDraft bool
	var w, height int
	err := h.DB.QueryRow(`
		SELECT i.version_id::text, v.project_id::text, v.is_draft, i.stem, i.ext, i.rel_image_path, COALESCE(i.label_text,''), i.width, i.height, i.split
		FROM ag_dataset_images i
		INNER JOIN ag_dataset_versions v ON v.id = i.version_id
		WHERE i.id=$1::uuid
	`, iid).Scan(&vid, &pid, &isDraft, &stem, &ext, &rel, &lbl, &w, &height, &split)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	if _, ok := RequireProjectViewerOrPublic(c, h.DB, pid); !ok {
		return
	}
	canEdit, _ := projectCanEditByUser(h.DB, uid, pid)
	canEdit = canEdit && isDraft
	c.JSON(http.StatusOK, gin.H{
		"id": iid, "version_id": vid, "stem": stem, "ext": ext,
		"label_text": lbl, "width": w, "height": height, "split": split, "can_edit": canEdit,
	})
}

func inferSplit(rel string) string {
	n := strings.ReplaceAll(rel, `\`, `/`)
	for _, sp := range []string{"train", "valid", "test"} {
		if strings.Contains(n, "/"+sp+"/") || strings.HasPrefix(n, sp+"/") {
			return sp
		}
	}
	return "train"
}

func contentType(ext string) string {
	switch strings.ToLower(ext) {
	case "jpg", "jpeg":
		return "image/jpeg"
	case "png":
		return "image/png"
	case "webp":
		return "image/webp"
	case "gif":
		return "image/gif"
	default:
		return "application/octet-stream"
	}
}

func (h *DatasetHandler) GetImageFile(c *gin.Context) {
	iid := c.Param("imgid")
	var pid, vid, rel, ext string
	err := h.DB.QueryRow(`
		SELECT v.project_id::text, i.version_id::text, i.rel_image_path, i.ext FROM ag_dataset_images i
		INNER JOIN ag_dataset_versions v ON v.id = i.version_id
		WHERE i.id=$1::uuid
	`, iid).Scan(&pid, &vid, &rel, &ext)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	if _, ok := RequireProjectViewerOrPublic(c, h.DB, pid); !ok {
		return
	}
	fp := filepath.Join(h.StorageRoot, vid, filepath.FromSlash(strings.ReplaceAll(rel, `\`, `/`)))
	stat, err := os.Stat(fp)
	if err != nil || stat.IsDir() {
		c.JSON(http.StatusNotFound, gin.H{"error": "file missing"})
		return
	}
	f, err := os.Open(fp)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "open"})
		return
	}
	defer f.Close()
	c.Header("Content-Type", contentType(ext))
	c.Header("Cache-Control", "private, max-age=3600")
	http.ServeContent(c.Writer, c.Request, filepath.Base(fp), stat.ModTime(), f)
}

func (h *DatasetHandler) PutLabel(c *gin.Context) {
	iid := c.Param("imgid")
	var vid, pid string
	var isDraft bool
	err := h.DB.QueryRow(`
		SELECT i.version_id::text, v.project_id::text, v.is_draft FROM ag_dataset_images i
		INNER JOIN ag_dataset_versions v ON v.id = i.version_id
		WHERE i.id=$1::uuid
	`, iid).Scan(&vid, &pid, &isDraft)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	if ok := RequireProjectOwner(c, h.DB, pid); !ok {
		return
	}
	if !isDraft {
		c.JSON(http.StatusForbidden, gin.H{"error": "read only version"})
		return
	}
	var body struct {
		LabelText string `json:"label_text"`
		Split     string `json:"split"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid"})
		return
	}
	bc := ylabel.CountBBoxes(body.LabelText)
	split := strings.TrimSpace(body.Split)
	if split == "" {
		split = "train"
	}
	if split != "train" && split != "valid" && split != "test" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid split"})
		return
	}
	if _, err := h.DB.Exec(`UPDATE ag_dataset_images SET label_text=$1, bbox_count=$2, split=$3 WHERE id=$4::uuid`, body.LabelText, bc, split, iid); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	_, _ = h.DB.Exec(`UPDATE ag_projects SET updated_at=now() WHERE id=$1::uuid`, pid)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *DatasetHandler) VersionNameAvailable(c *gin.Context) {
	pid := c.Param("pid")
	if ok := RequireProjectOwner(c, h.DB, pid); !ok {
		return
	}
	q := strings.TrimSpace(c.Query("q"))
	if q == "" {
		c.JSON(http.StatusOK, gin.H{"available": true, "taken": false})
		return
	}
	taken, err := h.datasetVersionNameExists(pid, q)
	if err != nil {
		log.Printf("VersionNameAvailable: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"available": !taken, "taken": taken})
}

func (h *DatasetHandler) CreateEmptyVersion(c *gin.Context) {
	pid := c.Param("pid")
	if ok := RequireProjectOwner(c, h.DB, pid); !ok {
		return
	}
	var b struct {
		Name string `json:"name"`
	}
	_ = c.ShouldBindJSON(&b)
	name, ok := h.allocateImportVersionName(c, pid, b.Name)
	if !ok {
		return
	}
	yamlSeed := `nc: 0
names: []
`
	var vid string
	if err := h.DB.QueryRow(`
		INSERT INTO ag_dataset_versions (project_id, name, data_yaml, is_draft) VALUES ($1::uuid, $2, $3, FALSE) RETURNING id::text
	`, pid, name, yamlSeed).Scan(&vid); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	vRoot := filepath.Join(h.StorageRoot, vid)
	_ = os.RemoveAll(vRoot)
	if err := os.MkdirAll(vRoot, 0o755); err != nil {
		_, _ = h.DB.Exec(`DELETE FROM ag_dataset_versions WHERE id=$1::uuid`, vid)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "storage"})
		return
	}
	_, _ = h.DB.Exec(`UPDATE ag_projects SET updated_at=now() WHERE id=$1::uuid`, pid)
	c.JSON(http.StatusCreated, gin.H{"id": vid, "name": name})
}

func (h *DatasetHandler) CreateVersionFromDataset(c *gin.Context) {
	pid := c.Param("pid")
	if ok := RequireProjectOwner(c, h.DB, pid); !ok {
		return
	}
	var b struct {
		Name         string `json:"name"`
		Resize       int    `json:"resize"`
		KeepOriginal bool   `json:"keep_original_size"`
		TrainPct     int    `json:"train_pct"`
		ValidPct     int    `json:"valid_pct"`
		TestPct      int    `json:"test_pct"`
		Rebalance    bool   `json:"rebalance"`
		Notes        string `json:"notes"`
	}
	_ = c.ShouldBindJSON(&b)
	name, ok := h.allocateImportVersionName(c, pid, b.Name)
	if !ok {
		return
	}
	src, srcYAML, err := h.listProjectDatasetSource(pid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	classNames := namesFromSourceForVersion(srcYAML, src)
	resize := b.Resize
	if b.KeepOriginal {
		resize = 0
	}
	if resize <= 0 && !b.KeepOriginal {
		resize = 640
	}
	yamlSeed := buildDatasetYAMLText(classNames, resize, b.KeepOriginal)
	var vid string
	if err := h.DB.QueryRow(`
		INSERT INTO ag_dataset_versions (project_id, name, data_yaml, is_draft)
		VALUES ($1::uuid, $2, $3, FALSE)
		RETURNING id::text
	`, pid, name, yamlSeed).Scan(&vid); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	counts, err := h.buildVersionFromDataset(pid, vid, resize, b.TrainPct, b.ValidPct, b.TestPct, b.Rebalance)
	if err != nil {
		_, _ = h.DB.Exec(`DELETE FROM ag_dataset_versions WHERE id=$1::uuid`, vid)
		_ = os.RemoveAll(filepath.Join(h.StorageRoot, vid))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, _ = h.DB.Exec(`UPDATE ag_projects SET updated_at=now() WHERE id=$1::uuid`, pid)
	c.JSON(http.StatusCreated, gin.H{
		"id":        vid,
		"name":      name,
		"resize":    resize,
		"counts":    counts,
		"rebalance": b.Rebalance,
	})
}

func (h *DatasetHandler) UploadProjectImages(c *gin.Context) {
	pid := c.Param("pid")
	if ok := RequireProjectOwner(c, h.DB, pid); !ok {
		return
	}
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "multipart"})
		return
	}
	defer form.RemoveAll()

	files := make([]*multipart.FileHeader, 0, len(form.File["files"])+len(form.File["file"]))
	files = append(files, form.File["files"]...)
	files = append(files, form.File["file"]...)
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "files required"})
		return
	}

	batchName := ""
	if vs := form.Value["batch_name"]; len(vs) > 0 {
		batchName = strings.TrimSpace(vs[0])
	}
	now := time.Now()
	if batchName == "" {
		batchName = batchFallbackName(now)
	}
	batchSlug := slugForPath(batchName) + "-" + strconv.FormatInt(now.Unix(), 10)

	draftVID, err := h.ensureDraftVersion(pid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	targetDir := filepath.Join(h.StorageRoot, draftVID, "uploads", batchSlug)
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "storage"})
		return
	}

	uploader := h.userEmailByID(middleware.UserID(c))
	existingHashes := map[string]struct{}{}
	exRows, err := h.DB.Query(`
		SELECT i.rel_image_path
		FROM ag_dataset_images i
		WHERE i.version_id=$1::uuid
		  AND i.in_dataset=TRUE
	`, draftVID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	for exRows.Next() {
		var rel string
		if exRows.Scan(&rel) != nil {
			continue
		}
		fp := filepath.Join(h.StorageRoot, draftVID, filepath.FromSlash(strings.ReplaceAll(rel, `\`, `/`)))
		sum, er := fileSHA256Hex(fp)
		if er != nil || strings.TrimSpace(sum) == "" {
			continue
		}
		existingHashes[sum] = struct{}{}
	}
	exRows.Close()
	totalImages := 0
	duplicates := 0
	skippedUnsupported := 0
	failed := 0
	imported := 0
	for i, fh := range files {
		base := filepath.Base(fh.Filename)
		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(base), "."))
		if ext == "" {
			skippedUnsupported++
			continue
		}
		if !isSupportedImageExt(ext) {
			skippedUnsupported++
			continue
		}
		totalImages++
		stem := strings.TrimSuffix(base, filepath.Ext(base))
		stem = strings.TrimSpace(stem)
		if stem == "" {
			stem = "image-" + strconv.Itoa(i+1)
		}
		outName := fmt.Sprintf("%s-%d.%s", slugForPath(stem), time.Now().UnixNano(), ext)
		dst := filepath.Join(targetDir, outName)
		if err := c.SaveUploadedFile(fh, dst); err != nil {
			failed++
			continue
		}
		sum, er := fileSHA256Hex(dst)
		if er != nil {
			_ = os.Remove(dst)
			failed++
			continue
		}
		if _, dup := existingHashes[sum]; dup {
			_ = os.Remove(dst)
			duplicates++
			continue
		}
		rel := filepath.ToSlash(filepath.Join("uploads", batchSlug, outName))
		if _, er := h.DB.Exec(`
			INSERT INTO ag_dataset_images (
				version_id, split, stem, ext, rel_image_path, label_text, width, height, bbox_count,
				batch_name, uploaded_by_email, uploaded_at, in_dataset
			) VALUES (
				$1::uuid, 'train', $2, $3, $4, '', 0, 0, 0, $5, $6, now(), FALSE
			)
		`, draftVID, stem, ext, rel, batchName, uploader); er != nil {
			_ = os.Remove(dst)
			failed++
			continue
		}
		existingHashes[sum] = struct{}{}
		imported++
	}
	if imported > 0 {
		_, _ = h.DB.Exec(`UPDATE ag_projects SET updated_at=now() WHERE id=$1::uuid`, pid)
	}
	c.JSON(http.StatusCreated, gin.H{
		"ok":                  true,
		"batch_name":          batchName,
		"imported":            imported,
		"total_files":         len(files),
		"total_images":        totalImages,
		"duplicates":          duplicates,
		"skipped_unsupported": skippedUnsupported,
		"failed":              failed,
	})
}

func (h *DatasetHandler) DeleteBatch(c *gin.Context) {
	pid := c.Param("pid")
	if ok := RequireProjectOwner(c, h.DB, pid); !ok {
		return
	}
	batch := strings.TrimSpace(c.Param("batch"))
	if batch == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "batch required"})
		return
	}
	rows, err := h.DB.Query(`
		SELECT v.id::text, i.rel_image_path
		FROM ag_dataset_images i
		INNER JOIN ag_dataset_versions v ON v.id=i.version_id
		WHERE v.project_id=$1::uuid
		  AND v.is_draft=TRUE
		  AND COALESCE(NULLIF(i.batch_name,''),'Uploaded')=$2
		  AND i.in_dataset=FALSE
	`, pid, batch)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	paths := make([]string, 0)
	for rows.Next() {
		var vid, rel string
		if rows.Scan(&vid, &rel) == nil {
			fp := filepath.Join(h.StorageRoot, vid, filepath.FromSlash(strings.ReplaceAll(rel, `\`, `/`)))
			paths = append(paths, fp)
		}
	}
	rows.Close()
	res, err := h.DB.Exec(`
		DELETE FROM ag_dataset_images i
		USING ag_dataset_versions v
		WHERE i.version_id=v.id
		  AND v.project_id=$1::uuid
		  AND v.is_draft=TRUE
		  AND COALESCE(NULLIF(i.batch_name,''),'Uploaded')=$2
		  AND i.in_dataset=FALSE
	`, pid, batch)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	for _, fp := range paths {
		if strings.TrimSpace(fp) == "" {
			continue
		}
		_ = os.Remove(fp)
	}
	n, _ := res.RowsAffected()
	c.JSON(http.StatusOK, gin.H{"ok": true, "deleted": n})
}

func (h *DatasetHandler) DeleteProjectImages(c *gin.Context) {
	pid := c.Param("pid")
	if ok := RequireProjectOwner(c, h.DB, pid); !ok {
		return
	}
	var body struct {
		IDs []string `json:"ids"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || len(body.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ids required"})
		return
	}
	clean := make([]string, 0, len(body.IDs))
	for _, id := range body.IDs {
		s := strings.TrimSpace(id)
		if s != "" {
			clean = append(clean, s)
		}
	}
	if len(clean) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ids required"})
		return
	}
	res, err := h.DB.Exec(`
		UPDATE ag_dataset_images i
		SET in_dataset=FALSE
		FROM ag_dataset_versions v
		WHERE i.version_id=v.id
		  AND v.project_id=$1::uuid
		  AND v.is_draft=TRUE
		  AND i.in_dataset=TRUE
		  AND i.id::text = ANY($2::text[])
	`, pid, pq.Array(clean))
	if err != nil {
		log.Printf("DeleteProjectImages: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	_, _ = h.DB.Exec(`UPDATE ag_projects SET updated_at=now() WHERE id=$1::uuid`, pid)
	n, _ := res.RowsAffected()
	c.JSON(http.StatusOK, gin.H{"ok": true, "deleted": n})
}

func (h *DatasetHandler) ListProjectBatches(c *gin.Context) {
	pid := c.Param("pid")
	if _, ok := RequireProjectViewerOrPublic(c, h.DB, pid); !ok {
		return
	}
	rows, err := h.DB.Query(`
		SELECT
			COALESCE(NULLIF(i.batch_name, ''), 'Uploaded'),
			COALESCE(NULLIF(MAX(i.uploaded_by_email), ''), ''),
			MIN(i.uploaded_at),
			COUNT(*)::int,
			SUM(CASE WHEN i.in_dataset THEN 1 ELSE 0 END)::int,
			SUM(CASE WHEN NOT i.in_dataset THEN 1 ELSE 0 END)::int,
			SUM(CASE WHEN NOT i.in_dataset AND i.bbox_count > 0 THEN 1 ELSE 0 END)::int,
			SUM(CASE WHEN NOT i.in_dataset AND i.bbox_count <= 0 THEN 1 ELSE 0 END)::int
		FROM ag_dataset_images i
		INNER JOIN ag_dataset_versions v ON v.id = i.version_id
		WHERE v.project_id = $1::uuid
		  AND v.is_draft=TRUE
		GROUP BY COALESCE(NULLIF(i.batch_name, ''), 'Uploaded')
		ORDER BY MIN(i.uploaded_at) DESC
	`, pid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	defer rows.Close()

	unassigned := make([]gin.H, 0)
	annotating := make([]gin.H, 0)
	dataset := make([]gin.H, 0)
	for rows.Next() {
		var batchName, labeler string
		var uploadedAt time.Time
		var totalCount, inDatasetCount, pendingCount, pendingLabeled, pendingUnlabeled int
		if rows.Scan(&batchName, &labeler, &uploadedAt, &totalCount, &inDatasetCount, &pendingCount, &pendingLabeled, &pendingUnlabeled) != nil {
			continue
		}
		if inDatasetCount > 0 {
			dataset = append(dataset, gin.H{
				"batch_name":    batchName,
				"labeler_email": labeler,
				"uploaded_at":   uploadedAt.Unix(),
				"image_count":   inDatasetCount,
				"in_version":    true,
				"annotated":     inDatasetCount,
				"unannotated":   0,
			})
		}
		if pendingCount > 0 {
			pendingItem := gin.H{
				"batch_name":    batchName,
				"labeler_email": labeler,
				"uploaded_at":   uploadedAt.Unix(),
				"image_count":   pendingCount,
				"in_version":    inDatasetCount > 0,
				"annotated":     pendingLabeled,
				"unannotated":   pendingUnlabeled,
			}
			if pendingLabeled > 0 {
				annotating = append(annotating, pendingItem)
			} else {
				unassigned = append(unassigned, pendingItem)
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"unassigned": unassigned,
		"annotating": annotating,
		"dataset":    dataset,
	})
}

func (h *DatasetHandler) StartBatchAnnotating(c *gin.Context) {
	pid := c.Param("pid")
	if ok := RequireProjectOwner(c, h.DB, pid); !ok {
		return
	}
	batch := strings.TrimSpace(c.Param("batch"))
	if batch == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "batch required"})
		return
	}
	var imgID string
	err := h.DB.QueryRow(`
		SELECT i.id::text
		FROM ag_dataset_images i
		INNER JOIN ag_dataset_versions v ON v.id=i.version_id
		WHERE v.project_id=$1::uuid
		  AND v.is_draft=TRUE
		  AND COALESCE(NULLIF(i.batch_name,''),'Uploaded')=$2
		  AND i.in_dataset=FALSE
		ORDER BY CASE WHEN i.bbox_count > 0 THEN 1 ELSE 0 END ASC, i.uploaded_at ASC, i.id ASC
		LIMIT 1
	`, pid, batch).Scan(&imgID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "batch not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"image_id": imgID})
}

func (h *DatasetHandler) ListBatchImages(c *gin.Context) {
	pid := c.Param("pid")
	if _, ok := RequireProjectViewerOrPublic(c, h.DB, pid); !ok {
		return
	}
	batch := strings.TrimSpace(c.Param("batch"))
	if batch == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "batch required"})
		return
	}
	anno := strings.TrimSpace(strings.ToLower(c.Query("anno")))
	whereAnno := ""
	switch anno {
	case "yes", "1", "true":
		whereAnno = " AND i.bbox_count > 0"
	case "no", "0", "false":
		whereAnno = " AND i.bbox_count = 0"
	}
	baseWhere := `
		FROM ag_dataset_images i
		INNER JOIN ag_dataset_versions v ON v.id=i.version_id
		WHERE v.project_id=$1::uuid
		  AND v.is_draft=TRUE
		  AND COALESCE(NULLIF(i.batch_name,''),'Uploaded')=$2
		  AND i.in_dataset=FALSE
	` + whereAnno
	var total int64
	if err := h.DB.QueryRow(`SELECT COUNT(*) `+baseWhere, pid, batch).Scan(&total); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	rows, err := h.DB.Query(`
		SELECT i.id::text, i.stem, i.ext, i.width, i.height, i.bbox_count, COALESCE(i.label_text,'')
	`+baseWhere+`
		ORDER BY i.uploaded_at ASC, i.id ASC
		LIMIT 500
	`, pid, batch)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	defer rows.Close()
	out := make([]gin.H, 0)
	for rows.Next() {
		var id, stem, ext, lbl string
		var w, height, bc int
		if err := rows.Scan(&id, &stem, &ext, &w, &height, &bc, &lbl); err != nil {
			continue
		}
		boxes := ylabel.CompactBBoxes(lbl, 96)
		bj := make([][]float64, len(boxes))
		for i, b := range boxes {
			bj[i] = []float64{b[0], b[1], b[2], b[3], b[4]}
		}
		out = append(out, gin.H{
			"id":         id,
			"stem":       stem,
			"ext":        ext,
			"width":      w,
			"height":     height,
			"bbox_count": bc,
			"has_label":  bc > 0,
			"boxes":      bj,
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"images": out,
		"total":  total,
		"batch":  batch,
	})
}

func (h *DatasetHandler) AddBatchAnnotatedToDataset(c *gin.Context) {
	pid := c.Param("pid")
	if ok := RequireProjectOwner(c, h.DB, pid); !ok {
		return
	}
	batch := strings.TrimSpace(c.Param("batch"))
	if batch == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "batch required"})
		return
	}
	res, err := h.DB.Exec(`
		UPDATE ag_dataset_images i
		SET in_dataset=TRUE
		FROM ag_dataset_versions v
		WHERE i.version_id=v.id
		  AND v.project_id=$1::uuid
		  AND v.is_draft=TRUE
		  AND COALESCE(NULLIF(i.batch_name,''),'Uploaded')=$2
		  AND i.in_dataset=FALSE
		  AND (i.bbox_count > 0 OR length(trim(COALESCE(i.label_text,''))) > 0)
	`, pid, batch)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	n, _ := res.RowsAffected()
	c.JSON(http.StatusOK, gin.H{"ok": true, "moved": n})
}

func (h *DatasetHandler) ImportZip(c *gin.Context) {
	pid := c.Param("pid")
	tReq := time.Now()
	log.Printf("[alpha-guard import] ZIP project_id=%s: запрос принят — дальше MultipartForm (пока тело целиком не прочитано и не разобрано, ответ клиенту не уйдёт)", pid)
	if ok := RequireProjectOwner(c, h.DB, pid); !ok {
		return
	}
	tMultipart := time.Now()
	form, err := c.MultipartForm()
	if err != nil {
		log.Printf("ImportZip MultipartForm: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("multipart: %v", err)})
		return
	}
	defer form.RemoveAll()
	log.Printf("[alpha-guard import] ZIP project_id=%s: MultipartForm завершился за %v (парсинг/сохранение частей multipart в temp)", pid, time.Since(tMultipart))

	fileParts := form.File["file"]
	if len(fileParts) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file required"})
		return
	}
	fh := fileParts[0]
	postedName := ""
	if vs := form.Value["name"]; len(vs) > 0 {
		postedName = vs[0]
	}
	name, ok := h.allocateImportVersionName(c, pid, postedName)
	if !ok {
		return
	}
	ext := strings.ToLower(filepath.Ext(fh.Filename))
	if ext != ".zip" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "expected .zip"})
		return
	}
	staging, err := h.importStagingDir()
	if err != nil {
		log.Printf("importStagingDir: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("staging dir: %v", err)})
		return
	}
	tmpZip, err := os.CreateTemp(staging, "dataset-*.zip")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("temp zip: %v", err)})
		return
	}
	zpath := tmpZip.Name()
	tmpZip.Close()
	defer os.Remove(zpath)

	log.Printf("[alpha-guard import] ZIP project_id=%s: в форме файл %q заголовочный_Size=%d, сохраняем в staging…", pid, fh.Filename, fh.Size)
	tSave := time.Now()
	if err := c.SaveUploadedFile(fh, zpath); err != nil {
		log.Printf("ImportZip SaveUploadedFile: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Не удалось сохранить загруженный zip (%s): %v", h.StorageRoot, err),
			"hint":  "Убедитесь, что на каталог STORAGE_ROOT есть права записи (см. лог сервера с полным путём).",
		})
		return
	}
	log.Printf("[alpha-guard import] ZIP project_id=%s: zip записан за %v -> %s", pid, time.Since(tSave), zpath)

	tmpExtract, err := os.MkdirTemp(staging, "dataset-ex-*")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("temp extract dir: %v", err)})
		return
	}
	log.Printf("[alpha-guard import] ZIP project_id=%s: распаковка во временный каталог %s …", pid, tmpExtract)
	tUnzip := time.Now()
	if err := h.unzipToDir(zpath, tmpExtract); err != nil {
		_ = os.RemoveAll(tmpExtract)
		log.Printf("ImportZip unzipToDir: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("bad zip: %v", err)})
		return
	}
	log.Printf("[alpha-guard import] ZIP project_id=%s: распаковка (только unzip) заняла %v", pid, time.Since(tUnzip))

	if asyncImportRequested(c) {
		jobID := newImportJob(middleware.UserID(c))
		log.Printf("[alpha-guard import] ZIP project_id=%s async=1 job_id=%s: ответ 202 можно слать клиенту, приём+unzip суммарно %v — дальше фон импорт в БД", pid, jobID, time.Since(tReq))
		go func(root string, jid string) {
			defer os.RemoveAll(root)
			h.runImportJob(jid, pid, name, root)
		}(tmpExtract, jobID)
		c.JSON(http.StatusAccepted, gin.H{"job_id": jobID})
		return
	}
	log.Printf("[alpha-guard import] ZIP project_id=%s sync: приём+unzip %v — дальше importDatasetCore в этом же запросе", pid, time.Since(tReq))
	defer os.RemoveAll(tmpExtract)
	h.finishImportExtracted(c, pid, name, tmpExtract)
}

func (h *DatasetHandler) ImportFolder(c *gin.Context) {
	pid := c.Param("pid")
	tReq := time.Now()
	log.Printf("[alpha-guard import] FOLDER project_id=%s: запрос принят — MultipartForm читает и разбирает все файлы дерева папки одним блоком", pid)
	if ok := RequireProjectOwner(c, h.DB, pid); !ok {
		return
	}
	tMultipart := time.Now()
	form, err := c.MultipartForm()
	if err != nil {
		log.Printf("ImportFolder MultipartForm: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("multipart: %v", err)})
		return
	}
	if form == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "multipart required"})
		return
	}
	defer form.RemoveAll()
	files := form.File["files"]
	nFiles := len(files)
	log.Printf("[alpha-guard import] FOLDER project_id=%s: MultipartForm за %v — внутри частей multipart пришло %d файлов (ещё сохранение на STORAGE_ROOT ниже по циклу)", pid, time.Since(tMultipart), nFiles)
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no files"})
		return
	}
	postedName := ""
	if vs := form.Value["name"]; len(vs) > 0 {
		postedName = vs[0]
	}
	name, ok := h.allocateImportVersionName(c, pid, postedName)
	if !ok {
		return
	}
	staging, err := h.importStagingDir()
	if err != nil {
		log.Printf("importStagingDir: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("staging dir: %v", err)})
		return
	}
	tmpRoot, err := os.MkdirTemp(staging, "folder-import-*")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("temp folder: %v", err)})
		return
	}
	log.Printf("[alpha-guard import] FOLDER project_id=%s: сохранение %d файлов в %s …", pid, nFiles, tmpRoot)
	tSaveLoop := time.Now()
	lastProg := time.Now()
	for idx, fh := range files {
		raw := fh.Filename
		raw = filepath.ToSlash(raw)
		if strings.Contains(raw, "..") {
			continue
		}
		dest := filepath.Join(tmpRoot, filepath.FromSlash(raw))
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "mkdir"})
			return
		}
		if err := c.SaveUploadedFile(fh, dest); err != nil {
			log.Printf("ImportFolder SaveUploadedFile %q: %v", raw, err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Не удалось сохранить файл «%s»: %v", filepath.Base(raw), err),
				"hint":  "Проверьте место на диске и запускайте backend из текущей версии проекта.",
			})
			return
		}
		if idx == 0 || idx+1 == nFiles || (idx+1)%400 == 0 || time.Since(lastProg) > 10*time.Second {
			log.Printf("[alpha-guard import] FOLDER project_id=%s: записано файлов %d/%d за %v", pid, idx+1, nFiles, time.Since(tSaveLoop))
			lastProg = time.Now()
		}
	}
	log.Printf("[alpha-guard import] FOLDER project_id=%s: все файлы сохранены за %v", pid, time.Since(tSaveLoop))

	if asyncImportRequested(c) {
		jobID := newImportJob(middleware.UserID(c))
		log.Printf("[alpha-guard import] FOLDER project_id=%s async=1 job_id=%s: можно отдавать 202, приём папки с запросом %v — дальше фон импорт в БД", pid, jobID, time.Since(tReq))
		go func(root string, jid string) {
			defer os.RemoveAll(root)
			h.runImportJob(jid, pid, name, root)
		}(tmpRoot, jobID)
		c.JSON(http.StatusAccepted, gin.H{"job_id": jobID})
		return
	}
	log.Printf("[alpha-guard import] FOLDER project_id=%s sync: приём %v — синхронный импорт в БД ниже в этом же HTTP-запросе", pid, time.Since(tReq))
	defer os.RemoveAll(tmpRoot)
	h.finishImportExtracted(c, pid, name, tmpRoot)
}

func (h *DatasetHandler) runImportJob(jobID, pid, versionName, extractRoot string) {
	t0 := time.Now()
	job, ok := importJobByID(jobID)
	if !ok {
		log.Printf("[alpha-guard import] job_id=%s: запись задачи уже не найдена (TTL/перезапуск процесса?)", jobID)
		return
	}
	log.Printf("[alpha-guard import] job_id=%s project_id=%s extract=%s : старт importDatasetCore (фон)", jobID, pid, extractRoot)

	payload, httpErr, errBody := h.importDatasetCore(pid, versionName, extractRoot, func(phase string, pct int, detail string) {
		job.applyProgress(phase, pct, detail)
	}, func() bool {
		return job.isCancelRequested()
	})
	if httpErr != 0 {
		msg := ginErrPrimaryMessage(errBody)
		if strings.TrimSpace(msg) == "" {
			msg = "ошибка импорта"
		}
		log.Printf("[alpha-guard import] job_id=%s: ошибка http=%d %q за %v", jobID, httpErr, msg, time.Since(t0))
		job.finishErr(httpErr, msg, errBody)
		return
	}
	log.Printf("[alpha-guard import] job_id=%s: успех за %v version_id=%v", jobID, time.Since(t0), payload["id"])
	job.finishOK(payload)
}

func ginErrPrimaryMessage(body gin.H) string {
	if body == nil {
		return ""
	}
	if e, ok := body["error"].(string); ok {
		return e
	}
	return ""
}

func (h *DatasetHandler) importDatasetCore(pid, versionName, extractRoot string, rep func(phase string, pct int, detail string), shouldStop func() bool) (payload gin.H, httpErr int, errBody gin.H) {
	if rep == nil {
		rep = func(string, int, string) {}
	}
	if shouldStop == nil {
		shouldStop = func() bool { return false }
	}

	tCore := time.Now()
	log.Printf("[alpha-guard import] importDatasetCore START project_id=%s version=%q extractRoot=%s", pid, versionName, extractRoot)

	lastPh := ""
	var lastProgAt time.Time
	repLogged := func(phase string, pct int, detail string) {
		n := time.Now()
		if phase != lastPh || n.Sub(lastProgAt) > 12*time.Second {
			log.Printf("[alpha-guard import] importDatasetCore project_id=%s phase=%s pct=%d detail=%s (с старта импорта %v)",
				pid, phase, pct, detail, n.Sub(tCore))
			lastPh = phase
			lastProgAt = n
		}
		rep(phase, pct, detail)
	}

	repLogged("version", 5, "создание черновой версии в базе данных")
	var vid string
	if err := h.DB.QueryRow(`
		INSERT INTO ag_dataset_versions (project_id, name, data_yaml, is_draft) VALUES ($1::uuid, $2, '', TRUE)
		RETURNING id::text
	`, pid, versionName).Scan(&vid); err != nil {
		log.Printf("importDatasetCore INSERT: %v", err)
		em := strings.ToLower(err.Error())
		if strings.Contains(em, "duplicate key") || strings.Contains(em, "unique constraint") {
			return nil, http.StatusConflict, gin.H{
				"error": "Версия датасета с таким именем уже есть в этом проекте",
				"code":  "duplicate_version_name",
				"name":  versionName,
			}
		}
		return nil, http.StatusInternalServerError, gin.H{
			"error": "db",
			"hint":  err.Error(),
		}
	}

	repLogged("storage", 12, "подготовка каталога на диске")
	vRoot := filepath.Join(h.StorageRoot, vid)
	if err := os.RemoveAll(vRoot); err != nil {
		_, _ = h.DB.Exec(`DELETE FROM ag_dataset_versions WHERE id=$1::uuid`, vid)
		return nil, http.StatusInternalServerError, gin.H{"error": "storage"}
	}
	if err := os.MkdirAll(vRoot, 0o755); err != nil {
		_, _ = h.DB.Exec(`DELETE FROM ag_dataset_versions WHERE id=$1::uuid`, vid)
		return nil, http.StatusInternalServerError, gin.H{"error": "storage"}
	}

	repLogged("import", 18, "копирование файлов и запись в базу данных")
	lastPct := 18
	yamlText, counts, err := ImportYOLOFromDir(h.DB, extractRoot, vRoot, vid, func(done, total int) {
		var pct int
		if total <= 0 {
			pct = 18 + min(done, 50)
			if pct > 88 {
				pct = 88
			}
		} else {
			pct = 18 + (72*done)/total
			if pct > 90 {
				pct = 90
			}
		}
		if pct < lastPct {
			pct = lastPct
		}
		lastPct = pct
		detail := fmt.Sprintf("изображений в базе: %d", done)
		if total > 0 {
			detail = fmt.Sprintf("изображения %d из %d в базу данных", done, total)
		}
		repLogged("import", pct, detail)
	}, shouldStop)
	if err != nil {
		_, _ = h.DB.Exec(`DELETE FROM ag_dataset_versions WHERE id=$1::uuid`, vid)
		_ = os.RemoveAll(vRoot)
		if strings.Contains(strings.ToLower(err.Error()), "cancelled") {
			return nil, 499, gin.H{"error": "import cancelled"}
		}
		return nil, http.StatusBadRequest, gin.H{"error": err.Error()}
	}
	sum := counts["train"] + counts["valid"] + counts["test"]
	if sum == 0 {
		_, _ = h.DB.Exec(`DELETE FROM ag_dataset_versions WHERE id=$1::uuid`, vid)
		_ = os.RemoveAll(vRoot)
		return nil, http.StatusBadRequest, gin.H{"error": "no images found"}
	}

	repLogged("finalize", 94, "сохранение data.yaml")
	if _, err := h.DB.Exec(`UPDATE ag_dataset_versions SET data_yaml=$2 WHERE id=$1::uuid`, vid, yamlText); err != nil {
		return nil, http.StatusInternalServerError, gin.H{
			"error": "db",
			"hint":  err.Error(),
		}
	}
	repLogged("done", 100, "готово")
	_, _ = h.DB.Exec(`UPDATE ag_projects SET updated_at=now() WHERE id=$1::uuid`, pid)
	log.Printf("[alpha-guard import] importDatasetCore OK project_id=%s version_id=%s images=%v за %v",
		pid, vid, counts, time.Since(tCore))
	return gin.H{"id": vid, "name": versionName, "counts": counts}, 0, nil
}

func (h *DatasetHandler) finishImportExtracted(c *gin.Context, pid, versionName, extractRoot string) {
	payload, httpErr, errBody := h.importDatasetCore(pid, versionName, extractRoot, nil, nil)
	if httpErr != 0 {
		c.JSON(httpErr, errBody)
		return
	}
	c.JSON(http.StatusCreated, payload)
}
