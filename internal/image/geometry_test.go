package image

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// The overlap bug: cell height was assumed to be 20 px, so a portrait poster
// scaled to fit a 20 px/cell budget consumed more rows than were reserved on a
// terminal with shorter cells. fitCells must never exceed the row budget for
// any cell size.
func TestFitCellsNeverExceedsBudget(t *testing.T) {
	const maxCols, maxRows = 56, 24

	cases := []struct {
		name         string
		imgW, imgH   int
		cellW, cellH int
	}{
		{"portrait 2:3, short cells", 320, 480, 8, 14},
		{"portrait 2:3, tall cells", 320, 480, 9, 22},
		{"portrait extreme", 200, 1200, 8, 17},
		{"landscape 16:9", 504, 283, 8, 17},
		{"square", 400, 400, 10, 20},
		{"tiny source", 3, 5, 9, 20},
		{"huge source", 4000, 6000, 8, 17},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cols, rows := fitCells(tc.imgW, tc.imgH, tc.cellW, tc.cellH, maxCols, maxRows)
			require.LessOrEqual(t, rows, maxRows)
			require.LessOrEqual(t, cols, maxCols)
			require.Positive(t, rows)
			require.Positive(t, cols)

			// Aspect ratio preserved within one cell of slack on the free axis.
			boxAspect := float64(cols*tc.cellW) / float64(rows*tc.cellH)
			srcAspect := float64(tc.imgW) / float64(tc.imgH)
			slack := float64(tc.cellW) / float64(rows*tc.cellH)
			require.InDelta(t, srcAspect, boxAspect, srcAspect*slack+slack+0.01)
		})
	}
}

func TestFitCellsRejectsDegenerateInput(t *testing.T) {
	cols, rows := fitCells(0, 0, 9, 20, 56, 24)
	require.Zero(t, cols)
	require.Zero(t, rows)

	cols, rows = fitCells(320, 480, 9, 20, 0, 0)
	require.Zero(t, cols)
	require.Zero(t, rows)
}
