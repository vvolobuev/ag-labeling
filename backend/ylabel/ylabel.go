package ylabel

import (
	"fmt"
	"strconv"
	"strings"
)

func CountBBoxes(label string) int {
	n := 0
	for _, line := range strings.Split(strings.ReplaceAll(label, "\r\n", "\n"), "\n") {
		t := strings.TrimSpace(line)
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		if _, ok := parseLineToBBox(t); ok {
			n++
		}
	}
	return n
}

func CompactBBoxes(label string, maxN int) [][5]float64 {
	if maxN <= 0 {
		return nil
	}
	var out [][5]float64
	for _, line := range strings.Split(strings.ReplaceAll(label, "\r\n", "\n"), "\n") {
		t := strings.TrimSpace(line)
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		if b, ok := parseLineToBBox(t); ok {
			out = append(out, b)
			if len(out) >= maxN {
				break
			}
		}
	}
	return out
}

func isBBoxLine(t string) bool {
	_, ok := parseLineToBBox(t)
	return ok
}

// NormalizeToBBoxLabel converts YOLO segmentation lines
// ("cls x1 y1 x2 y2 ...") into bbox lines ("cls cx cy w h").
// Existing bbox lines are kept as-is (normalized formatting).
func NormalizeToBBoxLabel(label string) string {
	var out []string
	for _, line := range strings.Split(strings.ReplaceAll(label, "\r\n", "\n"), "\n") {
		t := strings.TrimSpace(line)
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		b, ok := parseLineToBBox(t)
		if !ok {
			continue
		}
		out = append(out, formatBBoxLine(b))
	}
	if len(out) == 0 {
		return ""
	}
	return strings.Join(out, "\n") + "\n"
}

func parseLineToBBox(t string) ([5]float64, bool) {
	if b, ok := parseBBoxLine(t); ok {
		return b, true
	}
	return parseSegLineToBBox(t)
}

func parseBBoxLine(t string) ([5]float64, bool) {
	parts := strings.Fields(t)
	if len(parts) != 5 {
		return [5]float64{}, false
	}
	cls, err := strconv.Atoi(parts[0])
	if err != nil {
		return [5]float64{}, false
	}
	var f [5]float64
	f[0] = float64(cls)
	for i := 1; i < 5; i++ {
		v, err := strconv.ParseFloat(parts[i], 64)
		if err != nil {
			return [5]float64{}, false
		}
		f[i] = v
	}
	return f, true
}

func parseSegLineToBBox(t string) ([5]float64, bool) {
	parts := strings.Fields(t)
	if len(parts) < 7 || len(parts)%2 == 0 {
		return [5]float64{}, false
	}
	cls, err := strconv.Atoi(parts[0])
	if err != nil {
		return [5]float64{}, false
	}

	x0, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return [5]float64{}, false
	}
	y0, err := strconv.ParseFloat(parts[2], 64)
	if err != nil {
		return [5]float64{}, false
	}
	minX, maxX := x0, x0
	minY, maxY := y0, y0

	for i := 3; i < len(parts); i += 2 {
		x, err := strconv.ParseFloat(parts[i], 64)
		if err != nil {
			return [5]float64{}, false
		}
		y, err := strconv.ParseFloat(parts[i+1], 64)
		if err != nil {
			return [5]float64{}, false
		}
		if x < minX {
			minX = x
		}
		if x > maxX {
			maxX = x
		}
		if y < minY {
			minY = y
		}
		if y > maxY {
			maxY = y
		}
	}

	w := maxX - minX
	h := maxY - minY
	if w <= 0 || h <= 0 {
		return [5]float64{}, false
	}

	return [5]float64{
		float64(cls),
		minX + w/2,
		minY + h/2,
		w,
		h,
	}, true
}

func formatBBoxLine(b [5]float64) string {
	return fmt.Sprintf("%d %.6f %.6f %.6f %.6f", int(b[0]), b[1], b[2], b[3], b[4])
}
