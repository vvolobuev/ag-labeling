package handlers

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	xdraw "golang.org/x/image/draw"

	"github.com/gin-gonic/gin"
	"my-app/ylabel"
)

type datasetSourceImage struct {
	ID       string
	Version  string
	Split    string
	Stem     string
	Ext      string
	RelPath  string
	Label    string
	Width    int
	Height   int
	BBox     int
	Inserted int64
}

func parsePercent(v, fallback int) int {
	if v < 0 {
		return fallback
	}
	if v > 100 {
		return 100
	}
	return v
}

func normalizeSplitPercents(train, valid, test int) (int, int, int) {
	train = parsePercent(train, 75)
	valid = parsePercent(valid, 18)
	test = parsePercent(test, 7)
	sum := train + valid + test
	if sum <= 0 {
		return 75, 18, 7
	}
	train = (train * 100) / sum
	valid = (valid * 100) / sum
	test = 100 - train - valid
	return train, valid, test
}

func distributeCounts(total, trainPct, validPct int) (int, int, int) {
	if total <= 0 {
		return 0, 0, 0
	}
	trainN := int(float64(total) * float64(trainPct) / 100.0)
	validN := int(float64(total) * float64(validPct) / 100.0)
	if trainN < 0 {
		trainN = 0
	}
	if validN < 0 {
		validN = 0
	}
	if trainN+validN > total {
		validN = total - trainN
	}
	testN := total - trainN - validN
	return trainN, validN, testN
}

func (h *DatasetHandler) listProjectDatasetSource(pid string) ([]datasetSourceImage, string, error) {
	var yaml string
	_ = h.DB.QueryRow(`
		SELECT COALESCE(data_yaml,'')
		FROM ag_dataset_versions
		WHERE project_id=$1::uuid
		ORDER BY is_draft DESC, created_at DESC
		LIMIT 1
	`, pid).Scan(&yaml)

	rows, err := h.DB.Query(`
		SELECT
			i.id::text,
			i.version_id::text,
			i.split,
			i.stem,
			i.ext,
			i.rel_image_path,
			COALESCE(i.label_text,''),
			i.width,
			i.height,
			i.bbox_count,
			EXTRACT(EPOCH FROM COALESCE(i.uploaded_at, i.created_at))::bigint
		FROM ag_dataset_images i
		INNER JOIN ag_dataset_versions v ON v.id=i.version_id
		WHERE v.project_id=$1::uuid
		  AND v.is_draft=TRUE
		  AND i.in_dataset=TRUE
		ORDER BY COALESCE(i.uploaded_at, i.created_at) ASC, i.id ASC
	`, pid)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	out := make([]datasetSourceImage, 0)
	for rows.Next() {
		var it datasetSourceImage
		if err := rows.Scan(
			&it.ID, &it.Version, &it.Split, &it.Stem, &it.Ext, &it.RelPath, &it.Label,
			&it.Width, &it.Height, &it.BBox, &it.Inserted,
		); err != nil {
			continue
		}
		out = append(out, it)
	}
	return out, yaml, nil
}

func (h *DatasetHandler) VersionSourceStats(c *gin.Context) {
	pid := c.Param("pid")
	if ok := RequireProjectOwner(c, h.DB, pid); !ok {
		return
	}
	src, _, err := h.listProjectDatasetSource(pid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db"})
		return
	}
	total := len(src)
	unannotated := 0
	cls := map[int]struct{}{}
	splits := map[string]int{"train": 0, "valid": 0, "test": 0}
	for _, it := range src {
		if it.BBox <= 0 && strings.TrimSpace(it.Label) == "" {
			unannotated++
		}
		splits[it.Split]++
		for _, b := range ylabel.CompactBBoxes(it.Label, 1<<20) {
			cls[int(b[0])] = struct{}{}
		}
	}
	trainPct, validPct, testPct := 75, 18, 7
	if total > 0 {
		trainPct = int(float64(splits["train"]) * 100.0 / float64(total))
		validPct = int(float64(splits["valid"]) * 100.0 / float64(total))
		testPct = 100 - trainPct - validPct
	}
	c.JSON(http.StatusOK, gin.H{
		"total_images":       total,
		"unannotated_images": unannotated,
		"class_count":        len(cls),
		"splits":             splits,
		"suggested": gin.H{
			"train_pct": trainPct,
			"valid_pct": validPct,
			"test_pct":  testPct,
		},
	})
}

func decodeImage(buf []byte) (image.Image, string, error) {
	img, format, err := image.Decode(bytes.NewReader(buf))
	if err != nil {
		return nil, "", err
	}
	return img, strings.ToLower(strings.TrimSpace(format)), nil
}

func resizeLetterbox(src image.Image, size int) image.Image {
	if size <= 0 {
		return src
	}
	sb := src.Bounds()
	sw := sb.Dx()
	sh := sb.Dy()
	if sw <= 0 || sh <= 0 {
		return src
	}
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{C: color.Black}, image.Point{}, draw.Src)
	scale := float64(size) / float64(sw)
	if sh > sw {
		scale = float64(size) / float64(sh)
	}
	nw := int(float64(sw) * scale)
	nh := int(float64(sh) * scale)
	if nw < 1 {
		nw = 1
	}
	if nh < 1 {
		nh = 1
	}
	offX := (size - nw) / 2
	offY := (size - nh) / 2
	xdraw.CatmullRom.Scale(dst, image.Rect(offX, offY, offX+nw, offY+nh), src, sb, xdraw.Over, nil)
	return dst
}

func encodeImage(dstPath, ext string, img image.Image) error {
	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return err
	}
	f, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer f.Close()
	ext = strings.ToLower(strings.TrimPrefix(ext, "."))
	switch ext {
	case "jpg", "jpeg":
		return jpeg.Encode(f, img, &jpeg.Options{Quality: 90})
	case "gif":
		return gif.Encode(f, img, &gif.Options{NumColors: 256})
	default:
		return png.Encode(f, img)
	}
}

func copyOrResizeImage(srcPath, dstPath, ext string, resize int, fallbackW, fallbackH int) (int, int, error) {
	raw, err := os.ReadFile(srcPath)
	if err != nil {
		return 0, 0, err
	}
	img, format, err := decodeImage(raw)
	if err != nil {

		if mkErr := os.MkdirAll(filepath.Dir(dstPath), 0o755); mkErr != nil {
			return 0, 0, mkErr
		}
		if wrErr := os.WriteFile(dstPath, raw, 0o644); wrErr != nil {
			return 0, 0, wrErr
		}
		return fallbackW, fallbackH, nil
	}
	if ext == "" {
		ext = format
	}
	out := img
	if resize > 0 {
		out = resizeLetterbox(img, resize)
	}
	if err := encodeImage(dstPath, ext, out); err != nil {
		return 0, 0, err
	}
	b := out.Bounds()
	return b.Dx(), b.Dy(), nil
}

func (h *DatasetHandler) buildVersionFromDataset(pid, vid string, resize int, trainPct, validPct, testPct int, rebalance bool) (map[string]int, error) {
	src, _, err := h.listProjectDatasetSource(pid)
	if err != nil {
		return nil, err
	}
	if len(src) == 0 {
		return nil, fmt.Errorf("dataset is empty")
	}
	trainPct, validPct, testPct = normalizeSplitPercents(trainPct, validPct, testPct)
	_ = testPct
	trainN, validN, _ := distributeCounts(len(src), trainPct, validPct)

	ordered := src
	if rebalance {
		ordered = append([]datasetSourceImage(nil), src...)
		sort.SliceStable(ordered, func(i, j int) bool {
			if ordered[i].BBox != ordered[j].BBox {
				return ordered[i].BBox > ordered[j].BBox
			}
			return ordered[i].ID < ordered[j].ID
		})
	}

	targetRoot := filepath.Join(h.StorageRoot, vid)
	_ = os.MkdirAll(targetRoot, 0o755)
	seen := map[string]int{}
	counts := map[string]int{"train": 0, "valid": 0, "test": 0}

	for idx, it := range ordered {
		targetSplit := "test"
		switch {
		case idx < trainN:
			targetSplit = "train"
		case idx < trainN+validN:
			targetSplit = "valid"
		}

		srcPath := filepath.Join(h.StorageRoot, it.Version, filepath.FromSlash(strings.ReplaceAll(it.RelPath, `\`, `/`)))
		ext := strings.ToLower(strings.TrimSpace(it.Ext))
		if ext == "" {
			ext = "png"
		}
		stem := strings.TrimSpace(it.Stem)
		if stem == "" {
			stem = "image-" + strconv.Itoa(idx+1)
		}
		key := targetSplit + "/" + stem
		seen[key]++
		if seen[key] > 1 {
			stem = fmt.Sprintf("%s-%d", stem, seen[key])
		}
		filename := stem + "." + ext
		relPath := filepath.ToSlash(filepath.Join(targetSplit, "images", filename))
		dstPath := filepath.Join(targetRoot, filepath.FromSlash(relPath))
		w, hh, err := copyOrResizeImage(srcPath, dstPath, ext, resize, it.Width, it.Height)
		if err != nil {
			return nil, fmt.Errorf("failed to process source image %s: %w", it.ID, err)
		}
		bc := ylabel.CountBBoxes(it.Label)
		if _, err := h.DB.Exec(`
			INSERT INTO ag_dataset_images (
				version_id, split, stem, ext, rel_image_path, label_text, width, height, bbox_count, in_dataset
			) VALUES (
				$1::uuid, $2, $3, $4, $5, $6, $7, $8, $9, FALSE
			)
		`, vid, targetSplit, stem, ext, relPath, it.Label, w, hh, bc); err != nil {
			return nil, err
		}
		counts[targetSplit]++
	}
	if counts["train"]+counts["valid"]+counts["test"] != len(ordered) {
		return nil, fmt.Errorf("version snapshot is incomplete")
	}
	return counts, nil
}
