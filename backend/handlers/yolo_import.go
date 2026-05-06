package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"strings"

	"my-app/ylabel"

	"gopkg.in/yaml.v3"
)

type yoloYAML struct {
	Path   string `yaml:"path"`
	Train  string `yaml:"train"`
	Val    string `yaml:"val"`
	Valid  string `yaml:"valid"`
	Test   string `yaml:"test"`
	NC     int    `yaml:"nc"`
	Names  any    `yaml:"names"`
	RawRob any    `yaml:"roboflow"`
}

func shallowFindDataYAML(root string, maxDepth int) string {
	var walk func(dir string, depth int) string
	walk = func(dir string, depth int) string {
		if depth > maxDepth {
			return ""
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return ""
		}
		for _, e := range entries {
			p := filepath.Join(dir, e.Name())
			if !e.IsDir() && strings.EqualFold(e.Name(), "data.yaml") {
				return p
			}
			if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
				if s := walk(p, depth+1); s != "" {
					return s
				}
			}
		}
		return ""
	}
	return walk(root, 0)
}

func resolveRel(yamlDir, p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	return filepath.Clean(filepath.Join(yamlDir, filepath.FromSlash(p)))
}

func countImageFilesInDir(dir string) int {
	n := 0
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	for _, e := range entries {
		if e.IsDir() || !imageExtOK(e.Name()) {
			continue
		}
		n++
	}
	return n
}

func pickImageSourceDir(candidate string) (string, bool) {
	st, err := os.Stat(candidate)
	if err != nil || !st.IsDir() {
		return "", false
	}
	if countImageFilesInDir(candidate) > 0 {
		return candidate, true
	}
	sub := filepath.Join(candidate, "images")
	if st2, e := os.Stat(sub); e == nil && st2.IsDir() && countImageFilesInDir(sub) > 0 {
		return sub, true
	}
	return "", false
}

func imageDirCandidates(datasetBase, yamlDir, splitKey, raw string) []string {
	raw = filepath.FromSlash(strings.TrimSpace(raw))
	if raw == "" {
		return nil
	}
	seen := map[string]struct{}{}
	var out []string
	add := func(p string) {
		p = filepath.Clean(p)
		if _, ok := seen[p]; ok {
			return
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	add(resolveRel(datasetBase, raw))
	add(resolveRel(yamlDir, raw))

	sk := filepath.FromSlash(strings.TrimSpace(splitKey))
	if sk != "" {
		add(filepath.Join(datasetBase, sk, "images"))
		add(filepath.Join(datasetBase, "images", sk))
		add(filepath.Join(yamlDir, sk, "images"))
		add(filepath.Join(yamlDir, "images", sk))
	}
	return out
}

func imageExtOK(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp", ".bmp":
		return true
	default:
		return false
	}
}

func readImageDims(path string) (int, int) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return 0, 0
	}
	return cfg.Width, cfg.Height
}

type yoloImportTask struct {
	Split string
	Dir   string
}

func planYOLOTasks(datasetRoot string) (tasks []yoloImportTask, yamlText string, yamlPath string, dy *yoloYAML, datasetBase string, err error) {
	yamlPath = shallowFindDataYAML(datasetRoot, 6)
	if yamlPath != "" {
		rawYAML, er := os.ReadFile(yamlPath)
		if er != nil {
			return nil, "", "", nil, "", er
		}
		yamlText = string(rawYAML)
		dy = &yoloYAML{}
		if er := yaml.Unmarshal(rawYAML, dy); er != nil {
			return nil, "", yamlPath, nil, "", er
		}
		yamlDir := filepath.Dir(yamlPath)
		datasetBase = yamlDir
		if pt := strings.TrimSpace(dy.Path); pt != "" {
			datasetBase = resolveRel(yamlDir, pt)
		}
		type splitPath struct {
			key string
			raw string
		}
		sp := []splitPath{
			{"train", dy.Train},
			{"valid", firstNonEmpty(dy.Val, dy.Valid)},
			{"test", dy.Test},
		}
		for _, pair := range sp {
			if pair.raw == "" {
				continue
			}
			var picked string
			for _, cand := range imageDirCandidates(datasetBase, yamlDir, pair.key, pair.raw) {
				if d, ok := pickImageSourceDir(cand); ok {
					picked = d
					break
				}
			}
			if picked != "" {
				tasks = append(tasks, yoloImportTask{Split: pair.key, Dir: picked})
			}
		}
		return tasks, yamlText, yamlPath, dy, datasetBase, nil
	}

	rawYAML := []byte(`nc: 0
names: []
train: train/images
val: valid/images
test: test/images
`)
	yamlText = string(rawYAML)
	for _, folder := range []struct{ split, imgRel string }{
		{"train", "train/images"},
		{"valid", "valid/images"},
		{"test", "test/images"},
	} {
		imgDir := filepath.Join(datasetRoot, filepath.FromSlash(folder.imgRel))
		if st, e := os.Stat(imgDir); e != nil || !st.IsDir() {
			continue
		}
		tasks = append(tasks, yoloImportTask{Split: folder.split, Dir: imgDir})
	}
	return tasks, yamlText, "", nil, datasetBase, nil
}

func ImportYOLOFromDir(db *sql.DB, datasetRoot, versionRoot, versionID string, onImage func(done, total int), shouldStop func() bool) (yamlText string, perSplit map[string]int, err error) {
	perSplit = map[string]int{"train": 0, "valid": 0, "test": 0}

	tasks, yamlText, yamlPath, dy, datasetBase, err := planYOLOTasks(datasetRoot)
	if err != nil {
		return "", nil, err
	}

	totalImages := 0
	for _, t := range tasks {
		totalImages += countImageFilesInDir(t.Dir)
	}

	const maxAttempts = 3
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		counts, er := importYOLOAttempt(db, tasks, versionID, versionRoot, totalImages, onImage, shouldStop)
		if er != nil {
			if isTransientDBErr(er) && attempt < maxAttempts {
				continue
			}
			return "", nil, er
		}
		perSplit = counts

		if perSplit["train"]+perSplit["valid"]+perSplit["test"] == 0 {
			if yamlPath != "" && dy != nil {
				return "", nil, fmt.Errorf(
					"no images from data.yaml (yaml file: %s). train=%q val|valid=%q test=%q YAML path-field=%q resolvedBase=%s",
					yamlPath,
					strings.TrimSpace(dy.Train),
					firstNonEmpty(dy.Val, dy.Valid),
					strings.TrimSpace(dy.Test),
					strings.TrimSpace(dy.Path),
					datasetBase,
				)
			}
			return "", nil, fmt.Errorf("no YOLO images under train/valid/test")
		}
		return yamlText, perSplit, nil
	}

	return "", nil, fmt.Errorf("import failed after retries")
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

func importYOLOAttempt(
	db *sql.DB,
	tasks []yoloImportTask,
	versionID, versionRoot string,
	totalImages int,
	onImage func(done, total int),
	shouldStop func() bool,
) (map[string]int, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(`SET LOCAL lock_timeout = '5s'`); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(`SET LOCAL idle_in_transaction_session_timeout = '60s'`); err != nil {
		return nil, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	done := 0
	fire := func() {
		if onImage == nil {
			return
		}
		onImage(done, totalImages)
	}

	for _, t := range tasks {
		if err := importImageDir(tx, versionID, t.Split, t.Dir, versionRoot, func() {
			done++
			fire()
		}, shouldStop); err != nil {
			return nil, err
		}
	}

	perSplit := map[string]int{"train": 0, "valid": 0, "test": 0}
	for _, spl := range []string{"train", "valid", "test"} {
		rows, qerr := tx.Query(`SELECT COUNT(*) FROM ag_dataset_images WHERE version_id=$1::uuid AND split=$2`, versionID, spl)
		if qerr != nil {
			return nil, qerr
		}
		var n int
		if rows.Next() {
			_ = rows.Scan(&n)
		}
		rows.Close()
		perSplit[spl] = n
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	committed = true
	return perSplit, nil
}

func importImageDir(tx *sql.Tx, versionID, split, imgDir, versionRoot string, afterEach func(), shouldStop func() bool) error {
	if afterEach == nil {
		afterEach = func() {}
	}
	entries, err := os.ReadDir(imgDir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if shouldStop != nil && shouldStop() {
			return errors.New("import cancelled")
		}
		if e.IsDir() || !imageExtOK(e.Name()) {
			continue
		}
		stem := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(e.Name())), ".")

		src := filepath.Join(imgDir, e.Name())
		destDir := filepath.Join(versionRoot, split, "images")
		if err := os.MkdirAll(destDir, 0o755); err != nil {
			return err
		}
		dest := filepath.Join(destDir, e.Name())
		if err := copyFile(src, dest); err != nil {
			return err
		}

		w, h := readImageDims(dest)
		rel := fmt.Sprintf("%s/images/%s", split, filepath.Base(dest))
		label := readLabelForImage(imgDir, stem)
		label = ylabel.NormalizeToBBoxLabel(label)
		bc := ylabel.CountBBoxes(label)

		_, err = tx.Exec(`
			INSERT INTO ag_dataset_images (
				version_id, split, stem, ext, rel_image_path, label_text, width, height, bbox_count, in_dataset
			) VALUES (
				$1::uuid, $2, $3, $4, $5, $6, $7, $8, $9, TRUE
			)
			ON CONFLICT (version_id, split, stem) DO UPDATE SET
				label_text = EXCLUDED.label_text,
				rel_image_path = EXCLUDED.rel_image_path,
				ext = EXCLUDED.ext,
				width = EXCLUDED.width,
				height = EXCLUDED.height,
				bbox_count = EXCLUDED.bbox_count,
				in_dataset = EXCLUDED.in_dataset
		`, versionID, split, stem, ext, rel, label, w, h, bc)
		if err != nil {
			return err
		}
		afterEach()
	}
	return nil
}

func readLabelForImage(imgDir, stem string) string {
	if strings.TrimSpace(stem) == "" {
		return ""
	}

	parent := filepath.Dir(imgDir)
	base := filepath.Base(imgDir)
	grand := filepath.Dir(parent)
	candidates := []string{
		filepath.Join(parent, "labels", stem+".txt"),
		filepath.Join(imgDir, "labels", stem+".txt"),
		filepath.Join(grand, "labels", base, stem+".txt"),
		filepath.Join(grand, "labels", stem+".txt"),
	}
	seen := map[string]struct{}{}
	for _, p := range candidates {
		p = filepath.Clean(p)
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		if b, err := os.ReadFile(p); err == nil {
			return string(b)
		}
	}
	return ""
}

func isTransientDBErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, sql.ErrConnDone) {
		return true
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "driver: bad connection"),
		strings.Contains(msg, "broken pipe"),
		strings.Contains(msg, "connection reset"),
		strings.Contains(msg, "i/o timeout"),
		strings.Contains(msg, "connection refused"):
		return true
	default:
		return false
	}
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
