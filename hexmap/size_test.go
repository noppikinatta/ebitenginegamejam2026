package hexmap_test

import (
	"testing"

	"github.com/noppikinatta/ebitenginegamejam2026/hexmap"
)

func TestSizeCreation(t *testing.T) {
	cases := []struct {
		name           string
		radius         int
		expectedRadius int
	}{
		{
			name:           "半径0のサイズ",
			radius:         0,
			expectedRadius: 0,
		},
		{
			name:           "半径1のサイズ",
			radius:         1,
			expectedRadius: 1,
		},
		{
			name:           "半径5のサイズ",
			radius:         5,
			expectedRadius: 5,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			size := hexmap.NewSize(c.radius)
			if size.Radius() != c.expectedRadius {
				t.Errorf("期待する半径 %d, 実際の半径 %d", c.expectedRadius, size.Radius())
			}
		})
	}
}

func TestSizeContains(t *testing.T) {
	cases := []struct {
		name     string
		radius   int
		index    hexmap.Index
		expected bool
	}{
		{
			name:     "半径0_原点",
			radius:   0,
			index:    hexmap.IdxXY(0, 0),
			expected: true,
		},
		{
			name:     "半径0_隣接",
			radius:   0,
			index:    hexmap.IdxXY(1, 0),
			expected: false,
		},
		{
			name:     "半径1_原点",
			radius:   1,
			index:    hexmap.IdxXY(0, 0),
			expected: true,
		},
		{
			name:     "半径1_隣接_含まれる",
			radius:   1,
			index:    hexmap.IdxXY(1, 0),
			expected: true,
		},
		{
			name:     "半径1_距離2_含まれない",
			radius:   1,
			index:    hexmap.IdxXY(2, 0),
			expected: false,
		},
		{
			name:     "半径2_距離2_含まれる",
			radius:   2,
			index:    hexmap.IdxXY(2, 0),
			expected: true,
		},
		{
			name:     "半径2_距離3_含まれない",
			radius:   2,
			index:    hexmap.IdxXY(3, 0),
			expected: false,
		},
		{
			name:     "負の座標_半径内",
			radius:   2,
			index:    hexmap.IdxXY(-1, -1),
			expected: true,
		},
		{
			name:     "負の座標_半径外",
			radius:   1,
			index:    hexmap.IdxXY(-2, -2),
			expected: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			size := hexmap.NewSize(c.radius)
			if size.Contains(c.index) != c.expected {
				t.Errorf("Contains(%s): 期待値 %v, 実際の値 %v", c.index.String(), c.expected, size.Contains(c.index))
			}
		})
	}
}
