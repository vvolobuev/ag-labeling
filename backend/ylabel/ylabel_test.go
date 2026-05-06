package ylabel

import "testing"

func TestNormalizeToBBoxLabel_ConvertsSegmentation(t *testing.T) {
	in := "0 0.1 0.2 0.5 0.2 0.5 0.7 0.1 0.7\n"
	got := NormalizeToBBoxLabel(in)
	want := "0 0.300000 0.450000 0.400000 0.500000\n"
	if got != want {
		t.Fatalf("unexpected normalized label\nwant: %q\ngot:  %q", want, got)
	}
}

func TestCompactBBoxes_SupportsSegmentationLines(t *testing.T) {
	in := "1 0.2 0.3 0.4 0.3 0.4 0.8 0.2 0.8\n"
	boxes := CompactBBoxes(in, 5)
	if len(boxes) != 1 {
		t.Fatalf("expected 1 box, got %d", len(boxes))
	}
	b := boxes[0]
	if b[0] != 1 {
		t.Fatalf("expected class 1, got %.0f", b[0])
	}
	if b[1] <= 0 || b[2] <= 0 || b[3] <= 0 || b[4] <= 0 {
		t.Fatalf("invalid bbox values: %#v", b)
	}
}
