package handlers

import (
	"archive/zip"
	"database/sql"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"my-app/ylabel"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

func queryInt(c *gin.Context, key string) (int, bool) {
	s := strings.TrimSpace(c.Query(key))
	if s == "" {
		return 0, false
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	return v, true
}

func queryFloat(c *gin.Context, key string) (float64, bool) {
	s := strings.TrimSpace(c.Query(key))
	if s == "" {
		return 0, false
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

func parseCSVInts(raw string) []int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		v, err := strconv.Atoi(p)
		if err != nil {
			continue
		}
		out = append(out, v)
	}
	return out
}

func ilikeContains(q string) string {
	q = strings.ReplaceAll(q, `\`, `\\`)
	q = strings.ReplaceAll(q, `%`, `\%`)
	q = strings.ReplaceAll(q, `_`, `\_`)
	return "%" + q + "%"
}

func safeExportBase(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "dataset"
	}
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		return "dataset"
	}
	return out
}

func yamlWithoutResize(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	var doc map[string]any
	if err := yaml.Unmarshal([]byte(raw), &doc); err != nil {
		return raw
	}
	names := namesFromYAMLAny(doc["names"])
	if len(names) == 0 {
		if ncv, ok := doc["nc"]; ok {
			n := 0
			switch t := ncv.(type) {
			case int:
				n = t
			case int64:
				n = int(t)
			case float64:
				n = int(t)
			case string:
				n, _ = strconv.Atoi(strings.TrimSpace(t))
			}
			if n > 0 {
				names = make([]string, n)
				for i := 0; i < n; i++ {
					names[i] = fmt.Sprintf("class_%d", i)
				}
			}
		}
	}
	quoted := make([]string, 0, len(names))
	for i, nm := range names {
		s := strings.TrimSpace(nm)
		if s == "" {
			s = fmt.Sprintf("class_%d", i)
		}
		s = strings.ReplaceAll(s, `'`, `''`)
		quoted = append(quoted, "'"+s+"'")
	}
	return fmt.Sprintf(
		"train: ../train/images\nval: ../valid/images\ntest: ../test/images\n\nnc: %d\nnames: [%s]\n",
		len(quoted),
		strings.Join(quoted, ", "),
	)
}

func (h *DatasetHandler) VersionSplitStats(c *gin.Context) {
	vid := c.Param("vid")
	var pid string
	err := h.DB.QueryRow(`SELECT project_id::text FROM ag_dataset_versions WHERE id=$1::uuid`, vid).Scan(&pid)
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
	split := strings.TrimSpace(c.Query("split"))
	if split != "train" && split != "valid" && split != "test" {
		split = "train"
	}
	var total int64
	var wMin, wMax, hMin, hMax sql.NullInt64
	var bbMin, bbMax sql.NullInt64
	var arMin, arMax sql.NullFloat64
	var pxMin, pxMax sql.NullFloat64
	row := h.DB.QueryRow(`
		SELECT COUNT(*)::bigint,
			MIN(width), MAX(width), MIN(height), MAX(height),
			MIN(bbox_count), MAX(bbox_count),
			MIN(CASE WHEN height > 0 THEN width::double precision / height::double precision END),
			MAX(CASE WHEN height > 0 THEN width::double precision / height::double precision END),
			MIN((width::double precision * height::double precision) / 1000000.0),
			MAX((width::double precision * height::double precision) / 1000000.0)
		FROM ag_dataset_images
		WHERE version_id = $1::uuid AND split = $2
	`, vid, split)
	if err := row.Scan(&total, &wMin, &wMax, &hMin, &hMax, &bbMin, &bbMax, &arMin, &arMax, &pxMin, &pxMax); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"split":      split,
		"total":      total,
		"width":      gin.H{"min": nullInt64(wMin), "max": nullInt64(wMax)},
		"height":     gin.H{"min": nullInt64(hMin), "max": nullInt64(hMax)},
		"bbox_count": gin.H{"min": nullInt64(bbMin), "max": nullInt64(bbMax)},
		"aspect":     gin.H{"min": nullFloat64(arMin), "max": nullFloat64(arMax)},
		"megapixels": gin.H{"min": nullFloat64(pxMin), "max": nullFloat64(pxMax)},
	})
}

func nullInt64(v sql.NullInt64) any {
	if !v.Valid {
		return 0
	}
	return v.Int64
}

func nullFloat64(v sql.NullFloat64) any {
	if !v.Valid {
		return 0.0
	}
	return v.Float64
}

func (h *DatasetHandler) ListImages(c *gin.Context) {
	vid := c.Param("vid")
	var pid string
	err := h.DB.QueryRow(`SELECT project_id::text FROM ag_dataset_versions WHERE id=$1::uuid`, vid).Scan(&pid)
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
	split := strings.TrimSpace(c.Query("split"))
	if split != "train" && split != "valid" && split != "test" {
		split = "train"
	}
	page := 1
	if p, ok := queryInt(c, "page"); ok && p > 0 {
		page = p
	}
	perPage := 50
	if pp, ok := queryInt(c, "per_page"); ok {
		switch {
		case pp < 50:
			perPage = 50
		case pp > 200:
			perPage = 200
		default:
			perPage = pp
		}
	}
	offset := (page - 1) * perPage

	wb := strings.Builder{}
	wb.WriteString(`WHERE version_id=$1::uuid AND split=$2`)
	args := []any{vid, split}
	argN := 3

	stemSearch := strings.TrimSpace(c.Query("q"))
	if stemSearch != "" {
		fmt.Fprintf(&wb, " AND stem ILIKE $%d ESCAPE '\\'", argN)
		args = append(args, ilikeContains(stemSearch))
		argN++
	}

	if ann := strings.TrimSpace(strings.ToLower(c.Query("anno"))); ann != "" {
		switch ann {
		case "yes", "1", "true":
			wb.WriteString(` AND bbox_count > 0`)
		case "no", "0", "false":
			wb.WriteString(` AND bbox_count = 0`)
		}
	}

	if lo, ok := queryInt(c, "width_min"); ok {
		fmt.Fprintf(&wb, ` AND width >= $%d`, argN)
		args = append(args, lo)
		argN++
	}
	if hi, ok := queryInt(c, "width_max"); ok {
		fmt.Fprintf(&wb, ` AND width <= $%d`, argN)
		args = append(args, hi)
		argN++
	}
	if lo, ok := queryInt(c, "height_min"); ok {
		fmt.Fprintf(&wb, ` AND height >= $%d`, argN)
		args = append(args, lo)
		argN++
	}
	if hi, ok := queryInt(c, "height_max"); ok {
		fmt.Fprintf(&wb, ` AND height <= $%d`, argN)
		args = append(args, hi)
		argN++
	}
	if lo, ok := queryInt(c, "bbox_min"); ok {
		fmt.Fprintf(&wb, ` AND bbox_count >= $%d`, argN)
		args = append(args, lo)
		argN++
	}
	if hi, ok := queryInt(c, "bbox_max"); ok {
		fmt.Fprintf(&wb, ` AND bbox_count <= $%d`, argN)
		args = append(args, hi)
		argN++
	}
	if lo, ok := queryFloat(c, "aspect_min"); ok {
		fmt.Fprintf(&wb, ` AND height > 0 AND width::double precision / height::double precision >= $%d`, argN)
		args = append(args, lo)
		argN++
	}
	if hi, ok := queryFloat(c, "aspect_max"); ok {
		fmt.Fprintf(&wb, ` AND height > 0 AND width::double precision / height::double precision <= $%d`, argN)
		args = append(args, hi)
		argN++
	}
	if lo, ok := queryFloat(c, "mp_min"); ok {
		fmt.Fprintf(&wb, ` AND (width::double precision * height::double precision)/1000000.0 >= $%d`, argN)
		args = append(args, lo)
		argN++
	}
	if hi, ok := queryFloat(c, "mp_max"); ok {
		fmt.Fprintf(&wb, ` AND (width::double precision * height::double precision)/1000000.0 <= $%d`, argN)
		args = append(args, hi)
		argN++
	}

	whereSQL := wb.String()

	var total int64
	cq := `SELECT COUNT(*) FROM ag_dataset_images ` + whereSQL
	if err := h.DB.QueryRow(cq, args...).Scan(&total); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}

	totalPages := int((total + int64(perPage) - 1) / int64(perPage))
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
		offset = (page - 1) * perPage
	}

	dataQ := `SELECT id::text, stem, ext, width, height, bbox_count, COALESCE(label_text,'') FROM ag_dataset_images ` +
		whereSQL + ` ORDER BY stem ASC, id ASC LIMIT $` + strconv.Itoa(argN) + ` OFFSET $` + strconv.Itoa(argN+1)
	argsWithPage := append(args, perPage, offset)
	rows, err := h.DB.Query(dataQ, argsWithPage...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	defer rows.Close()
	out := make([]gin.H, 0, perPage)
	for rows.Next() {
		var id, stem, ext, lbl string
		var w, h, bc int
		if err := rows.Scan(&id, &stem, &ext, &w, &h, &bc, &lbl); err != nil {
			continue
		}
		boxes := ylabel.CompactBBoxes(lbl, 96)
		bj := make([][]float64, len(boxes))
		for i, b := range boxes {
			bj[i] = []float64{b[0], b[1], b[2], b[3], b[4]}
		}
		out = append(out, gin.H{
			"id": id, "stem": stem, "ext": ext, "width": w, "height": h,
			"has_label": bc > 0, "bbox_count": bc, "boxes": bj,
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"images":      out,
		"split":       split,
		"total":       total,
		"total_pages": totalPages,
		"page":        page,
		"per_page":    perPage,
	})
}

func (h *DatasetHandler) ListProjectImages(c *gin.Context) {
	pid := c.Param("pid")
	if _, ok := RequireProjectViewerOrPublic(c, h.DB, pid); !ok {
		return
	}

	split := strings.TrimSpace(c.Query("split"))
	if split != "train" && split != "valid" && split != "test" {
		split = "all"
	}

	page := 1
	if p, ok := queryInt(c, "page"); ok && p > 0 {
		page = p
	}
	perPage := 50
	if pp, ok := queryInt(c, "per_page"); ok {
		switch {
		case pp < 50:
			perPage = 50
		case pp > 200:
			perPage = 200
		default:
			perPage = pp
		}
	}
	offset := (page - 1) * perPage

	wb := strings.Builder{}
	wb.WriteString(`WHERE v.project_id=$1::uuid AND v.is_draft=TRUE AND i.in_dataset=TRUE`)
	args := []any{pid}
	argN := 2

	if split != "all" {
		fmt.Fprintf(&wb, " AND i.split=$%d", argN)
		args = append(args, split)
		argN++
	}

	stemSearch := strings.TrimSpace(c.Query("q"))
	if stemSearch != "" {
		fmt.Fprintf(&wb, " AND i.stem ILIKE $%d ESCAPE '\\'", argN)
		args = append(args, ilikeContains(stemSearch))
		argN++
	}
	batchName := strings.TrimSpace(c.Query("batch"))
	if batchName != "" {
		fmt.Fprintf(&wb, " AND COALESCE(NULLIF(i.batch_name,''),'Uploaded')=$%d", argN)
		args = append(args, batchName)
		argN++
	}
	if ann := strings.TrimSpace(strings.ToLower(c.Query("anno"))); ann != "" {
		switch ann {
		case "yes", "1", "true":
			wb.WriteString(` AND i.bbox_count > 0`)
		case "no", "0", "false":
			wb.WriteString(` AND i.bbox_count = 0`)
		}
	}
	if lo, ok := queryInt(c, "width_min"); ok {
		fmt.Fprintf(&wb, ` AND i.width >= $%d`, argN)
		args = append(args, lo)
		argN++
	}
	if hi, ok := queryInt(c, "width_max"); ok {
		fmt.Fprintf(&wb, ` AND i.width <= $%d`, argN)
		args = append(args, hi)
		argN++
	}
	if lo, ok := queryInt(c, "height_min"); ok {
		fmt.Fprintf(&wb, ` AND i.height >= $%d`, argN)
		args = append(args, lo)
		argN++
	}
	if hi, ok := queryInt(c, "height_max"); ok {
		fmt.Fprintf(&wb, ` AND i.height <= $%d`, argN)
		args = append(args, hi)
		argN++
	}
	if lo, ok := queryInt(c, "bbox_min"); ok {
		fmt.Fprintf(&wb, ` AND i.bbox_count >= $%d`, argN)
		args = append(args, lo)
		argN++
	}
	if hi, ok := queryInt(c, "bbox_max"); ok {
		fmt.Fprintf(&wb, ` AND i.bbox_count <= $%d`, argN)
		args = append(args, hi)
		argN++
	}
	if lo, ok := queryFloat(c, "aspect_min"); ok {
		fmt.Fprintf(&wb, ` AND i.height > 0 AND i.width::double precision / i.height::double precision >= $%d`, argN)
		args = append(args, lo)
		argN++
	}
	if hi, ok := queryFloat(c, "aspect_max"); ok {
		fmt.Fprintf(&wb, ` AND i.height > 0 AND i.width::double precision / i.height::double precision <= $%d`, argN)
		args = append(args, hi)
		argN++
	}
	if lo, ok := queryFloat(c, "mp_min"); ok {
		fmt.Fprintf(&wb, ` AND (i.width::double precision * i.height::double precision)/1000000.0 >= $%d`, argN)
		args = append(args, lo)
		argN++
	}
	if hi, ok := queryFloat(c, "mp_max"); ok {
		fmt.Fprintf(&wb, ` AND (i.width::double precision * i.height::double precision)/1000000.0 <= $%d`, argN)
		args = append(args, hi)
		argN++
	}
	for _, classID := range parseCSVInts(c.Query("class_present")) {
		fmt.Fprintf(&wb, ` AND i.bbox_count > 0 AND EXISTS (
			SELECT 1 FROM regexp_split_to_table(COALESCE(i.label_text,''), E'\\r?\\n') AS ln
			CROSS JOIN LATERAL regexp_split_to_array(BTRIM(ln), E'\\s+') AS tok
			WHERE array_length(tok, 1) >= 5
			  AND tok[1] ~ '^[+-]?[0-9]+(\\.[0]+)?$'
			  AND tok[1]::numeric = $%d::numeric
		)`, argN)
		args = append(args, classID)
		argN++
	}
	for _, classID := range parseCSVInts(c.Query("class_absent")) {
		fmt.Fprintf(&wb, ` AND (i.bbox_count = 0 OR NOT EXISTS (
			SELECT 1 FROM regexp_split_to_table(COALESCE(i.label_text,''), E'\\r?\\n') AS ln
			CROSS JOIN LATERAL regexp_split_to_array(BTRIM(ln), E'\\s+') AS tok
			WHERE array_length(tok, 1) >= 5
			  AND tok[1] ~ '^[+-]?[0-9]+(\\.[0]+)?$'
			  AND tok[1]::numeric = $%d::numeric
		))`, argN)
		args = append(args, classID)
		argN++
	}

	whereSQL := wb.String()
	sortBy := strings.TrimSpace(strings.ToLower(c.Query("sort")))
	orderSQL := "i.stem ASC, i.id ASC"
	switch sortBy {
	case "newest", "recent":
		orderSQL = "COALESCE(i.uploaded_at, i.created_at) DESC, i.id DESC"
	case "oldest":
		orderSQL = "COALESCE(i.uploaded_at, i.created_at) ASC, i.id ASC"
	case "objects_desc", "bbox_desc":
		orderSQL = "i.bbox_count DESC, i.stem ASC, i.id ASC"
	case "objects_asc", "bbox_asc":
		orderSQL = "i.bbox_count ASC, i.stem ASC, i.id ASC"
	}

	var total int64
	cq := `SELECT COUNT(*) FROM ag_dataset_images i INNER JOIN ag_dataset_versions v ON v.id=i.version_id ` + whereSQL
	if err := h.DB.QueryRow(cq, args...).Scan(&total); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	totalPages := int((total + int64(perPage) - 1) / int64(perPage))
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
		offset = (page - 1) * perPage
	}

	dataQ := `SELECT i.id::text, i.stem, i.ext, i.width, i.height, i.bbox_count, COALESCE(i.label_text,'')
		FROM ag_dataset_images i
		INNER JOIN ag_dataset_versions v ON v.id=i.version_id ` +
		whereSQL + ` ORDER BY ` + orderSQL + ` LIMIT $` + strconv.Itoa(argN) + ` OFFSET $` + strconv.Itoa(argN+1)
	argsWithPage := append(args, perPage, offset)
	rows, err := h.DB.Query(dataQ, argsWithPage...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	defer rows.Close()

	out := make([]gin.H, 0, perPage)
	for rows.Next() {
		var id, stem, ext, lbl string
		var w, h, bc int
		if err := rows.Scan(&id, &stem, &ext, &w, &h, &bc, &lbl); err != nil {
			continue
		}
		boxes := ylabel.CompactBBoxes(lbl, 96)
		bj := make([][]float64, len(boxes))
		for i, b := range boxes {
			bj[i] = []float64{b[0], b[1], b[2], b[3], b[4]}
		}
		out = append(out, gin.H{
			"id": id, "stem": stem, "ext": ext, "width": w, "height": h,
			"has_label": bc > 0, "bbox_count": bc, "boxes": bj,
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"images":      out,
		"split":       split,
		"total":       total,
		"total_pages": totalPages,
		"page":        page,
		"per_page":    perPage,
	})
}

func (h *DatasetHandler) PutVersionDataYAML(c *gin.Context) {
	vid := c.Param("vid")
	var pid string
	err := h.DB.QueryRow(`SELECT project_id::text FROM ag_dataset_versions WHERE id=$1::uuid`, vid).Scan(&pid)
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
		DataYAML string `json:"data_yaml"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid"})
		return
	}
	raw := body.DataYAML
	if strings.TrimSpace(raw) != "" {
		var probe map[string]any
		if err := yaml.Unmarshal([]byte(raw), &probe); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "yaml"})
			return
		}
	}
	if _, err := h.DB.Exec(`UPDATE ag_dataset_versions SET data_yaml=$1 WHERE id=$2::uuid`, raw, vid); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "data_yaml": raw})
}

func (h *DatasetHandler) ExportVersionZIP(c *gin.Context) {
	vid := c.Param("vid")
	var pid, vname string
	var yamlNS sql.NullString
	err := h.DB.QueryRow(`SELECT project_id::text, name, data_yaml FROM ag_dataset_versions WHERE id=$1::uuid`, vid).
		Scan(&pid, &vname, &yamlNS)
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
	rows, err := h.DB.Query(`
		SELECT split, stem, ext, rel_image_path, COALESCE(label_text,'')
		FROM ag_dataset_images WHERE version_id=$1::uuid ORDER BY split, stem
	`, vid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	defer rows.Close()
	var imageTotal int
	if err := h.DB.QueryRow(`SELECT COUNT(*) FROM ag_dataset_images WHERE version_id=$1::uuid`, vid).Scan(&imageTotal); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	if imageTotal <= 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "version is not ready yet"})
		return
	}
	root := safeExportBase(vname)
	yamlStr := ""
	if yamlNS.Valid {
		yamlStr = yamlNS.String
	}
	yamlStr = yamlWithoutResize(yamlStr)
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", `attachment; filename="`+root+`.zip"`)
	zw := zip.NewWriter(c.Writer)
	defer zw.Close()
	yw, err := zw.Create(root + "/data.yaml")
	if err != nil {
		return
	}
	if _, err := io.WriteString(yw, yamlStr); err != nil {
		return
	}
	for rows.Next() {
		var split, stem, ext, rel, lbl string
		if rows.Scan(&split, &stem, &ext, &rel, &lbl) != nil {
			continue
		}
		relNorm := filepath.ToSlash(strings.ReplaceAll(rel, `\`, `/`))
		fp := filepath.Join(h.StorageRoot, vid, filepath.FromSlash(relNorm))
		bin, er := os.ReadFile(fp)
		if er != nil {
			continue
		}
		zp := root + "/" + relNorm
		fw, er := zw.Create(strings.ReplaceAll(zp, `\`, `/`))
		if er != nil {
			continue
		}
		if _, er := fw.Write(bin); er != nil {
			continue
		}
		lzp := fmt.Sprintf("%s/%s/labels/%s.txt", root, split, stem)
		lw, er := zw.Create(strings.ReplaceAll(lzp, `\`, `/`))
		if er != nil {
			continue
		}
		_, _ = io.WriteString(lw, lbl)
	}
}

func yamlClassNamesSlice(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var doc map[string]any
	if err := yaml.Unmarshal([]byte(raw), &doc); err != nil {
		return nil
	}
	return namesFromYAMLAny(doc["names"])
}

func namesFromYAMLAny(v any) []string {
	switch t := v.(type) {
	case []any:
		out := make([]string, 0, len(t))
		for _, x := range t {
			switch s := x.(type) {
			case string:
				out = append(out, strings.TrimSpace(s))
			default:
				vs := strings.TrimSpace(fmt.Sprint(x))
				if vs != "" && vs != "<nil>" {
					out = append(out, vs)
				}
			}
		}
		return out
	case map[string]any:
		tmp := map[int]string{}
		maxK := -1
		for k, val := range t {
			ki, err := strconv.Atoi(strings.TrimSpace(k))
			if err != nil {
				continue
			}
			if s, ok := val.(string); ok {
				tmp[ki] = strings.TrimSpace(s)
				if ki > maxK {
					maxK = ki
				}
			}
		}
		if maxK < 0 {
			return nil
		}
		out := make([]string, maxK+1)
		for i := 0; i <= maxK; i++ {
			out[i] = tmp[i]
		}
		return out
	case map[any]any:
		tmp := map[int]string{}
		maxK := -1
		for k, val := range t {
			ki, err := strconv.Atoi(strings.TrimSpace(fmt.Sprint(k)))
			if err != nil {
				continue
			}
			vs := strings.TrimSpace(fmt.Sprint(val))
			tmp[ki] = vs
			if ki > maxK {
				maxK = ki
			}
		}
		if maxK < 0 {
			return nil
		}
		out := make([]string, maxK+1)
		for i := 0; i <= maxK; i++ {
			out[i] = tmp[i]
		}
		return out
	default:
		return nil
	}
}

func (h *DatasetHandler) VersionClassStats(c *gin.Context) {
	vid := c.Param("vid")
	var pid, yamlRaw string
	err := h.DB.QueryRow(`
		SELECT project_id::text, COALESCE(data_yaml,'') FROM ag_dataset_versions WHERE id=$1::uuid
	`, vid).Scan(&pid, &yamlRaw)
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
	rows, err := h.DB.Query(`SELECT COALESCE(label_text,'') FROM ag_dataset_images WHERE version_id=$1::uuid`, vid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	defer rows.Close()

	counts := map[int]int{}
	heatmap := make([][]int, 24)
	for i := range heatmap {
		heatmap[i] = make([]int, 24)
	}
	for rows.Next() {
		var lbl string
		if rows.Scan(&lbl) != nil {
			continue
		}
		for _, b := range ylabel.CompactBBoxes(lbl, 1<<20) {
			ci := int(b[0])
			if ci < 0 {
				continue
			}
			counts[ci]++
			cx, cy := b[1], b[2]
			if cx >= 0 && cx <= 1 && cy >= 0 && cy <= 1 {
				ix := int(cx * 24)
				iy := int(cy * 24)
				if ix > 23 {
					ix = 23
				}
				if iy > 23 {
					iy = 23
				}
				if ix < 0 {
					ix = 0
				}
				if iy < 0 {
					iy = 0
				}
				heatmap[iy][ix]++
			}
		}
	}
	names := yamlClassNamesSlice(yamlRaw)
	classIDs := make([]int, 0, len(counts))
	for k := range counts {
		classIDs = append(classIDs, k)
	}
	sort.Ints(classIDs)
	classes := make([]gin.H, 0, len(classIDs))
	for _, id := range classIDs {
		nm := ""
		if id >= 0 && id < len(names) {
			nm = names[id]
		}
		classes = append(classes, gin.H{
			"class_id": id, "name": nm, "count": counts[id],
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"version_id":   vid,
		"classes":      classes,
		"heatmap":      heatmap,
		"heatmap_size": gin.H{"rows": 24, "cols": 24},
	})
}

func (h *DatasetHandler) ProjectClassStats(c *gin.Context) {
	pid := c.Param("pid")
	if _, ok := RequireProjectViewerOrPublic(c, h.DB, pid); !ok {
		return
	}

	var yamlRaw string
	_ = h.DB.QueryRow(`
		SELECT COALESCE(data_yaml, '') FROM ag_dataset_versions
		WHERE project_id=$1::uuid
		  AND is_draft=TRUE
		ORDER BY created_at DESC
		LIMIT 1
	`, pid).Scan(&yamlRaw)

	rows, err := h.DB.Query(`
		SELECT i.split, COALESCE(i.label_text,''), i.width, i.height
		FROM ag_dataset_images i
		INNER JOIN ag_dataset_versions v ON v.id=i.version_id
		WHERE v.project_id=$1::uuid
		  AND v.is_draft=TRUE
		  AND i.in_dataset=TRUE
	`, pid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	defer rows.Close()

	type classAgg struct {
		count       int
		splitTrain  int
		splitValid  int
		splitTest   int
		aspectSum   float64
		aspectCount int
		areaSum     float64
	}
	classMap := map[int]*classAgg{}
	splitImages := map[string]int{"train": 0, "valid": 0, "test": 0}
	splitObjects := map[string]int{"train": 0, "valid": 0, "test": 0}
	totalImages := 0
	unannotatedImages := 0
	var minW, maxW, minH, maxH int
	dimCount := 0
	sumW, sumH := 0.0, 0.0
	aspectImgSum := 0.0
	aspectImgCount := 0
	for rows.Next() {
		var split, lbl string
		var w, h int
		if rows.Scan(&split, &lbl, &w, &h) != nil {
			continue
		}
		totalImages++
		if split != "train" && split != "valid" && split != "test" {
			split = "train"
		}
		splitImages[split]++
		if w > 0 && h > 0 {
			if dimCount == 0 {
				minW, maxW, minH, maxH = w, w, h, h
			} else {
				if w < minW {
					minW = w
				}
				if w > maxW {
					maxW = w
				}
				if h < minH {
					minH = h
				}
				if h > maxH {
					maxH = h
				}
			}
			dimCount++
			sumW += float64(w)
			sumH += float64(h)
		}
		if h > 0 {
			aspectImgSum += float64(w) / float64(h)
			aspectImgCount++
		}

		boxes := ylabel.CompactBBoxes(lbl, 1<<20)
		if len(boxes) == 0 {
			unannotatedImages++
		}
		for _, b := range boxes {
			cls := int(b[0])
			if cls < 0 {
				continue
			}
			ag := classMap[cls]
			if ag == nil {
				ag = &classAgg{}
				classMap[cls] = ag
			}
			ag.count++
			switch split {
			case "train":
				ag.splitTrain++
			case "valid":
				ag.splitValid++
			case "test":
				ag.splitTest++
			}
			bw := b[3]
			bh := b[4]
			if bw > 0 && bh > 0 {
				ag.aspectSum += bw / bh
				ag.aspectCount++
				ag.areaSum += bw * bh
			}
			splitObjects[split]++
		}
	}

	names := yamlClassNamesSlice(yamlRaw)
	classIDs := make([]int, 0, len(classMap))
	for k := range classMap {
		classIDs = append(classIDs, k)
	}
	sort.Ints(classIDs)
	classes := make([]gin.H, 0, len(classIDs))
	for _, id := range classIDs {
		ag := classMap[id]
		nm := ""
		if id >= 0 && id < len(names) {
			nm = names[id]
		}
		avgAspect := 0.0
		avgAreaPct := 0.0
		if ag.aspectCount > 0 {
			avgAspect = ag.aspectSum / float64(ag.aspectCount)
		}
		if ag.count > 0 {
			avgAreaPct = (ag.areaSum / float64(ag.count)) * 100.0
		}
		avgAspect = math.Round(avgAspect*1000) / 1000
		avgAreaPct = math.Round(avgAreaPct*1000) / 1000
		classes = append(classes, gin.H{
			"class_id": id,
			"name":     nm,
			"count":    ag.count,
			"by_split": gin.H{
				"train": ag.splitTrain,
				"valid": ag.splitValid,
				"test":  ag.splitTest,
			},
			"avg_bbox_aspect_ratio": avgAspect,
			"avg_bbox_area_pct":     avgAreaPct,
		})
	}

	avgW := 0.0
	avgH := 0.0
	avgAspectImg := 0.0
	if dimCount > 0 {
		avgW = sumW / float64(dimCount)
		avgH = sumH / float64(dimCount)
	}
	if aspectImgCount > 0 {
		avgAspectImg = aspectImgSum / float64(aspectImgCount)
	}
	avgW = math.Round(avgW*100) / 100
	avgH = math.Round(avgH*100) / 100
	avgAspectImg = math.Round(avgAspectImg*1000) / 1000

	c.JSON(http.StatusOK, gin.H{
		"project_id": pid,
		"summary": gin.H{
			"total_images":       totalImages,
			"total_classes":      len(classes),
			"unannotated_images": unannotatedImages,
		},
		"split_distribution": gin.H{
			"images":  splitImages,
			"objects": splitObjects,
		},
		"image_size": gin.H{
			"width":  gin.H{"min": minW, "avg": avgW, "max": maxW},
			"height": gin.H{"min": minH, "avg": avgH, "max": maxH},
		},
		"image_aspect_ratio": gin.H{
			"avg": avgAspectImg,
		},
		"classes": classes,
	})
}
