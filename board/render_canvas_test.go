package board

import (
	"strings"
	"testing"
)

func TestBoardCanvasPanicsOnOverlap(t *testing.T) {
	tests := []struct {
		name      string
		fragments []boardFragment
		want      string
	}{
		{
			name:      "same position",
			fragments: []boardFragment{{4, 2, "===="}, {4, 2, "PT"}},
			want:      "board canvas: overlapping fragments at column 4, row 2",
		},
		{
			name:      "partial overlap written out of order",
			fragments: []boardFragment{{6, 2, "PT"}, {4, 2, "===="}},
			want:      "board canvas: overlapping fragments at column 6, row 2",
		},
		{
			name:      "negative coordinates",
			fragments: []boardFragment{{-2, -1, "PT"}, {-1, -1, "WD"}},
			want:      "board canvas: overlapping fragments at column -1, row -1",
		},
		{
			name:      "second column of styled emoji",
			fragments: []boardFragment{{4, 2, "\x1b[34m⚓\x1b[0m"}, {5, 2, "PT"}},
			want:      "board canvas: overlapping fragments at column 5, row 2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canvas := newBoardCanvas()
			for _, fragment := range tt.fragments {
				canvas.write(fragment.x, fragment.y, fragment.text)
			}
			defer func() {
				if got := recover(); got != tt.want {
					t.Fatalf("got panic %v, want %q", got, tt.want)
				}
			}()
			canvas.lines()
		})
	}
}

func TestBoardCanvasAllowsAdjacentFragmentsAndSeparateRows(t *testing.T) {
	canvas := newBoardCanvas()
	anchor := "\x1b[34m⚓\x1b[0m"
	canvas.write(3, -1, "3 ")
	canvas.write(-1, 0, "WD")
	canvas.write(1, -1, anchor)
	canvas.write(-1, -1, "PT")
	lines := canvas.lines()
	if !strings.HasPrefix(lines[0], "PT"+anchor+"3 ") {
		t.Fatalf("adjacent fragments were not preserved: %q", lines[0])
	}
	if !strings.HasPrefix(lines[1], "WD") {
		t.Fatalf("fragment on separate row was not preserved: %q", lines[1])
	}
}
