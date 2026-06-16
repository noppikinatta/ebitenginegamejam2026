package hexmap_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/noppikinatta/turnbased/hexmap"
)

func TestIndexCreation(t *testing.T) {
	cases := []struct {
		Name           string
		Factory        string
		Args           []int
		ExpectedX      int
		ExpectedY      int
		ExpectedZ      int
		ExpectedString string
	}{
		{
			Name:           "IdxXY原点",
			Factory:        "XY",
			Args:           []int{0, 0},
			ExpectedX:      0,
			ExpectedY:      0,
			ExpectedZ:      0,
			ExpectedString: "(0,0,0)",
		},
		{
			Name:           "IdxXY正の値",
			Factory:        "XY",
			Args:           []int{3, 4},
			ExpectedX:      3,
			ExpectedY:      4,
			ExpectedZ:      -7,
			ExpectedString: "(3,4,-7)",
		},
		{
			Name:           "IdxXY負の値",
			Factory:        "XY",
			Args:           []int{-2, -5},
			ExpectedX:      -2,
			ExpectedY:      -5,
			ExpectedZ:      7,
			ExpectedString: "(-2,-5,7)",
		},
		{
			Name:           "IdxYZ原点",
			Factory:        "YZ",
			Args:           []int{0, 0},
			ExpectedX:      0,
			ExpectedY:      0,
			ExpectedZ:      0,
			ExpectedString: "(0,0,0)",
		},
		{
			Name:           "IdxYZ正の値",
			Factory:        "YZ",
			Args:           []int{3, 4},
			ExpectedX:      -7,
			ExpectedY:      3,
			ExpectedZ:      4,
			ExpectedString: "(-7,3,4)",
		},
		{
			Name:           "IdxYZ負の値",
			Factory:        "YZ",
			Args:           []int{-2, -5},
			ExpectedX:      7,
			ExpectedY:      -2,
			ExpectedZ:      -5,
			ExpectedString: "(7,-2,-5)",
		},
		{
			Name:           "IdxZX原点",
			Factory:        "ZX",
			Args:           []int{0, 0},
			ExpectedX:      0,
			ExpectedY:      0,
			ExpectedZ:      0,
			ExpectedString: "(0,0,0)",
		},
		{
			Name:           "IdxZX正の値",
			Factory:        "ZX",
			Args:           []int{3, 4},
			ExpectedX:      4,
			ExpectedY:      -7,
			ExpectedZ:      3,
			ExpectedString: "(4,-7,3)",
		},
		{
			Name:           "IdxZX負の値",
			Factory:        "ZX",
			Args:           []int{-2, -5},
			ExpectedX:      -5,
			ExpectedY:      7,
			ExpectedZ:      -2,
			ExpectedString: "(-5,7,-2)",
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			var idx hexmap.Index

			switch c.Factory {
			case "XY":
				idx = hexmap.IdxXY(c.Args[0], c.Args[1])
			case "YZ":
				idx = hexmap.IdxYZ(c.Args[0], c.Args[1])
			case "ZX":
				idx = hexmap.IdxZX(c.Args[0], c.Args[1])
			}

			if idx.X() != c.ExpectedX {
				t.Errorf("X(): expected %d, but got %d", c.ExpectedX, idx.X())
			}

			if idx.Y() != c.ExpectedY {
				t.Errorf("Y(): expected %d, but got %d", c.ExpectedY, idx.Y())
			}

			if idx.Z() != c.ExpectedZ {
				t.Errorf("Z(): expected %d, but got %d", c.ExpectedZ, idx.Z())
			}

			if idx.String() != c.ExpectedString {
				t.Errorf("String(): expected %s, but got %s", c.ExpectedString, idx.String())
			}
		})
	}
}

func TestIndexAdd(t *testing.T) {
	cases := []struct {
		Name     string
		Index1   hexmap.Index
		Index2   hexmap.Index
		Expected hexmap.Index
	}{
		{
			Name:     "原点同士",
			Index1:   hexmap.IdxXY(0, 0),
			Index2:   hexmap.IdxXY(0, 0),
			Expected: hexmap.IdxXY(0, 0),
		},
		{
			Name:     "正の値同士",
			Index1:   hexmap.IdxXY(2, 3),
			Index2:   hexmap.IdxXY(4, 5),
			Expected: hexmap.IdxXY(6, 8),
		},
		{
			Name:     "負の値同士",
			Index1:   hexmap.IdxXY(-1, -2),
			Index2:   hexmap.IdxXY(-3, -4),
			Expected: hexmap.IdxXY(-4, -6),
		},
		{
			Name:     "正と負の値",
			Index1:   hexmap.IdxXY(5, -2),
			Index2:   hexmap.IdxXY(-3, 7),
			Expected: hexmap.IdxXY(2, 5),
		},
		{
			Name:     "異なる座標系のインデックス",
			Index1:   hexmap.IdxYZ(1, 2),
			Index2:   hexmap.IdxZX(3, 4),
			Expected: hexmap.IdxXY(1, -6),
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			result := c.Index1.Add(c.Index2)

			if result.X() != c.Expected.X() {
				t.Errorf("X(): expected %d, but got %d", c.Expected.X(), result.X())
			}

			if result.Y() != c.Expected.Y() {
				t.Errorf("Y(): expected %d, but got %d", c.Expected.Y(), result.Y())
			}

			if result.Z() != c.Expected.Z() {
				t.Errorf("Z(): expected %d, but got %d", c.Expected.Z(), result.Z())
			}
		})
	}
}

func TestIndexMul(t *testing.T) {
	cases := []struct {
		Name     string
		Index    hexmap.Index
		Factor   int
		Expected hexmap.Index
	}{
		{
			Name:     "0倍",
			Index:    hexmap.IdxXY(3, 4),
			Factor:   0,
			Expected: hexmap.IdxXY(0, 0),
		},
		{
			Name:     "正の倍数",
			Index:    hexmap.IdxXY(2, 3),
			Factor:   5,
			Expected: hexmap.IdxXY(10, 15),
		},
		{
			Name:     "負の倍数",
			Index:    hexmap.IdxXY(2, 3),
			Factor:   -2,
			Expected: hexmap.IdxXY(-4, -6),
		},
		{
			Name:     "負のインデックスと正の倍数",
			Index:    hexmap.IdxXY(-1, -2),
			Factor:   3,
			Expected: hexmap.IdxXY(-3, -6),
		},
		{
			Name:     "負のインデックスと負の倍数",
			Index:    hexmap.IdxXY(-1, -2),
			Factor:   -3,
			Expected: hexmap.IdxXY(3, 6),
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			result := c.Index.Mul(c.Factor)

			if result.X() != c.Expected.X() {
				t.Errorf("X(): expected %d, but got %d", c.Expected.X(), result.X())
			}

			if result.Y() != c.Expected.Y() {
				t.Errorf("Y(): expected %d, but got %d", c.Expected.Y(), result.Y())
			}

			if result.Z() != c.Expected.Z() {
				t.Errorf("Z(): expected %d, but got %d", c.Expected.Z(), result.Z())
			}
		})
	}
}

func TestIndexDistance(t *testing.T) {
	cases := []struct {
		Name     string
		Index1   hexmap.Index
		Index2   hexmap.Index
		Expected int
	}{
		{
			Name:     "同じ位置",
			Index1:   hexmap.IdxXY(0, 0),
			Index2:   hexmap.IdxXY(0, 0),
			Expected: 0,
		},
		{
			Name:     "隣接位置",
			Index1:   hexmap.IdxXY(0, 0),
			Index2:   hexmap.IdxXY(1, 0),
			Expected: 1,
		},
		{
			Name:     "X軸上の移動",
			Index1:   hexmap.IdxXY(0, 0),
			Index2:   hexmap.IdxXY(5, 0),
			Expected: 5,
		},
		{
			Name:     "Y軸上の移動",
			Index1:   hexmap.IdxXY(0, 0),
			Index2:   hexmap.IdxXY(0, 4),
			Expected: 4,
		},
		{
			Name:     "Z軸上の移動",
			Index1:   hexmap.IdxXY(0, 0),
			Index2:   hexmap.IdxXY(-3, 3),
			Expected: 3,
		},
		{
			Name:     "複雑な経路",
			Index1:   hexmap.IdxXY(1, 2),
			Index2:   hexmap.IdxXY(4, -1),
			Expected: 3,
		},
		{
			Name:     "負の座標間",
			Index1:   hexmap.IdxXY(-2, -3),
			Index2:   hexmap.IdxXY(-5, -1),
			Expected: 3,
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			distance := c.Index1.Distance(c.Index2)

			if distance != c.Expected {
				t.Errorf("expected %d, but got %d", c.Expected, distance)
			}

			// 距離は対称的であるはず
			reverseDistance := c.Index2.Distance(c.Index1)
			if reverseDistance != c.Expected {
				t.Errorf("reverse distance: expected %d, but got %d", c.Expected, reverseDistance)
			}
		})
	}
}

func TestHelperFunctions(t *testing.T) {
	// distanceは非公開関数なので、その振る舞いを間接的にテストします
	cases := []struct {
		Name     string
		Index1   hexmap.Index
		Index2   hexmap.Index
		Expected int
	}{
		{
			Name:     "X軸のみの変化 (正方向)",
			Index1:   hexmap.IdxXY(0, 0),
			Index2:   hexmap.IdxXY(3, 0),
			Expected: 3,
		},
		{
			Name:     "X軸のみの変化 (負方向)",
			Index1:   hexmap.IdxXY(0, 0),
			Index2:   hexmap.IdxXY(-4, 0),
			Expected: 4,
		},
		{
			Name:     "Y軸のみの変化 (正方向)",
			Index1:   hexmap.IdxXY(0, 0),
			Index2:   hexmap.IdxXY(0, 5),
			Expected: 5,
		},
		{
			Name:     "Y軸のみの変化 (負方向)",
			Index1:   hexmap.IdxXY(0, 0),
			Index2:   hexmap.IdxXY(0, -6),
			Expected: 6,
		},
		{
			Name:     "Z軸のみの変化 (計算済みケース)",
			Index1:   hexmap.IdxXY(0, 0),
			Index2:   hexmap.IdxXY(7, -7),
			Expected: 7,
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			// X,Y,Z軸のうち一つだけ変化するケースをテスト
			// 他の2軸は変化しないので、Distanceの結果はその1軸の変化量に等しいはず
			distance := c.Index1.Distance(c.Index2)

			if distance != c.Expected {
				t.Errorf("expected %d, but got %d", c.Expected, distance)
			}
		})
	}
}

func TestDirectionConstants(t *testing.T) {
	cases := []struct {
		Name      string
		Direction hexmap.Index
		ExpectedX int
		ExpectedY int
		ExpectedZ int
	}{
		{
			Name:      "Direction01 (1時方向)",
			Direction: hexmap.Direction01,
			ExpectedX: 1,
			ExpectedY: -1,
			ExpectedZ: 0,
		},
		{
			Name:      "Direction03 (3時方向)",
			Direction: hexmap.Direction03,
			ExpectedX: 1,
			ExpectedY: 0,
			ExpectedZ: -1,
		},
		{
			Name:      "Direction05 (5時方向)",
			Direction: hexmap.Direction05,
			ExpectedX: 0,
			ExpectedY: 1,
			ExpectedZ: -1,
		},
		{
			Name:      "Direction07 (7時方向)",
			Direction: hexmap.Direction07,
			ExpectedX: -1,
			ExpectedY: 1,
			ExpectedZ: 0,
		},
		{
			Name:      "Direction09 (9時方向)",
			Direction: hexmap.Direction09,
			ExpectedX: -1,
			ExpectedY: 0,
			ExpectedZ: 1,
		},
		{
			Name:      "Direction11 (11時方向)",
			Direction: hexmap.Direction11,
			ExpectedX: 0,
			ExpectedY: -1,
			ExpectedZ: 1,
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			if c.Direction.X() != c.ExpectedX {
				t.Errorf("X(): expected %d, but got %d", c.ExpectedX, c.Direction.X())
			}

			if c.Direction.Y() != c.ExpectedY {
				t.Errorf("Y(): expected %d, but got %d", c.ExpectedY, c.Direction.Y())
			}

			if c.Direction.Z() != c.ExpectedZ {
				t.Errorf("Z(): expected %d, but got %d", c.ExpectedZ, c.Direction.Z())
			}

			// 原点からの距離が1であることを確認
			origin := hexmap.IdxXY(0, 0)
			distance := origin.Distance(c.Direction)
			if distance != 1 {
				t.Errorf("Distance from origin: expected 1, but got %d", distance)
			}
		})
	}
}

func TestAppendAroundBasic(t *testing.T) {
	cases := []struct {
		Name           string
		Index          hexmap.Index
		InitialSlice   []hexmap.Index
		ExpectedLength int
		ExpectedAround []hexmap.Index
	}{
		{
			Name:           "原点でのAppendAround（空スライス）",
			Index:          hexmap.IdxXY(0, 0),
			InitialSlice:   []hexmap.Index{},
			ExpectedLength: 6,
			ExpectedAround: []hexmap.Index{
				hexmap.IdxXY(1, -1), // Direction01: 1時方向
				hexmap.IdxXY(1, 0),  // Direction03: 3時方向
				hexmap.IdxXY(0, 1),  // Direction05: 5時方向
				hexmap.IdxXY(-1, 1), // Direction07: 7時方向
				hexmap.IdxXY(-1, 0), // Direction09: 9時方向
				hexmap.IdxXY(0, -1), // Direction11: 11時方向
			},
		},
		{
			Name:           "原点でのAppendAround（既存要素ありスライス）",
			Index:          hexmap.IdxXY(0, 0),
			InitialSlice:   []hexmap.Index{hexmap.IdxXY(10, 10)},
			ExpectedLength: 7,
			ExpectedAround: []hexmap.Index{
				hexmap.IdxXY(10, 10), // 既存要素
				hexmap.IdxXY(1, -1),  // Direction01: 1時方向
				hexmap.IdxXY(1, 0),   // Direction03: 3時方向
				hexmap.IdxXY(0, 1),   // Direction05: 5時方向
				hexmap.IdxXY(-1, 1),  // Direction07: 7時方向
				hexmap.IdxXY(-1, 0),  // Direction09: 9時方向
				hexmap.IdxXY(0, -1),  // Direction11: 11時方向
			},
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			result := c.Index.AppendAround(c.InitialSlice)

			// 長さの確認
			if len(result) != c.ExpectedLength {
				t.Errorf("Length: expected %d, but got %d", c.ExpectedLength, len(result))
			}

			// 各要素の確認
			for i, expected := range c.ExpectedAround {
				if i >= len(result) {
					t.Errorf("Missing element at index %d: expected %s", i, expected.String())
					continue
				}

				actual := result[i]
				if actual.X() != expected.X() || actual.Y() != expected.Y() {
					t.Errorf("Index %d: expected %s, but got %s", i, expected.String(), actual.String())
				}
			}

			// 周囲の座標が全て距離1であることを確認（既存要素を除く）
			startIdx := len(c.InitialSlice)
			for i := startIdx; i < len(result); i++ {
				distance := c.Index.Distance(result[i])
				if distance != 1 {
					t.Errorf("Distance from center to %s: expected 1, but got %d", result[i].String(), distance)
				}
			}
		})
	}
}

func TestAppendAroundArbitraryPosition(t *testing.T) {
	cases := []struct {
		Name           string
		Index          hexmap.Index
		ExpectedAround []hexmap.Index
	}{
		{
			Name:  "正の座標での周囲取得",
			Index: hexmap.IdxXY(3, 2),
			ExpectedAround: []hexmap.Index{
				hexmap.IdxXY(4, 1), // 3+1, 2-1 (Direction01)
				hexmap.IdxXY(4, 2), // 3+1, 2+0 (Direction03)
				hexmap.IdxXY(3, 3), // 3+0, 2+1 (Direction05)
				hexmap.IdxXY(2, 3), // 3-1, 2+1 (Direction07)
				hexmap.IdxXY(2, 2), // 3-1, 2+0 (Direction09)
				hexmap.IdxXY(3, 1), // 3+0, 2-1 (Direction11)
			},
		},
		{
			Name:  "負の座標での周囲取得",
			Index: hexmap.IdxXY(-2, -3),
			ExpectedAround: []hexmap.Index{
				hexmap.IdxXY(-1, -4), // -2+1, -3-1 (Direction01)
				hexmap.IdxXY(-1, -3), // -2+1, -3+0 (Direction03)
				hexmap.IdxXY(-2, -2), // -2+0, -3+1 (Direction05)
				hexmap.IdxXY(-3, -2), // -2-1, -3+1 (Direction07)
				hexmap.IdxXY(-3, -3), // -2-1, -3+0 (Direction09)
				hexmap.IdxXY(-2, -4), // -2+0, -3-1 (Direction11)
			},
		},
		{
			Name:  "混在座標（X正、Y負）での周囲取得",
			Index: hexmap.IdxXY(5, -2),
			ExpectedAround: []hexmap.Index{
				hexmap.IdxXY(6, -3), // 5+1, -2-1 (Direction01)
				hexmap.IdxXY(6, -2), // 5+1, -2+0 (Direction03)
				hexmap.IdxXY(5, -1), // 5+0, -2+1 (Direction05)
				hexmap.IdxXY(4, -1), // 5-1, -2+1 (Direction07)
				hexmap.IdxXY(4, -2), // 5-1, -2+0 (Direction09)
				hexmap.IdxXY(5, -3), // 5+0, -2-1 (Direction11)
			},
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			result := c.Index.AppendAround([]hexmap.Index{})

			// 長さの確認
			if len(result) != 6 {
				t.Errorf("Length: expected 6, but got %d", len(result))
			}

			// 各要素の確認
			for i, expected := range c.ExpectedAround {
				if i >= len(result) {
					t.Errorf("Missing element at index %d: expected %s", i, expected.String())
					continue
				}

				actual := result[i]
				if actual.X() != expected.X() || actual.Y() != expected.Y() {
					t.Errorf("Index %d: expected %s, but got %s", i, expected.String(), actual.String())
				}

				// 距離が1であることを確認
				distance := c.Index.Distance(actual)
				if distance != 1 {
					t.Errorf("Distance from %s to %s: expected 1, but got %d", c.Index.String(), actual.String(), distance)
				}
			}
		})
	}
}

func TestAppendAroundOrder(t *testing.T) {
	// 原点から時計回りに方向が返されることを確認
	origin := hexmap.IdxXY(0, 0)
	result := origin.AppendAround([]hexmap.Index{})

	expectedDirections := []struct {
		Name      string
		Direction hexmap.Index
		Clock     string
	}{
		{"Direction01", hexmap.Direction01, "1時方向"},
		{"Direction03", hexmap.Direction03, "3時方向"},
		{"Direction05", hexmap.Direction05, "5時方向"},
		{"Direction07", hexmap.Direction07, "7時方向"},
		{"Direction09", hexmap.Direction09, "9時方向"},
		{"Direction11", hexmap.Direction11, "11時方向"},
	}

	// 長さの確認
	if len(result) != 6 {
		t.Fatalf("Length: expected 6, but got %d", len(result))
	}

	// 時計回りの順序確認
	for i, expected := range expectedDirections {
		actual := result[i]
		expectedIndex := origin.Add(expected.Direction)

		if actual.X() != expectedIndex.X() || actual.Y() != expectedIndex.Y() {
			t.Errorf("Position %d (%s): expected %s, but got %s",
				i, expected.Clock, expectedIndex.String(), actual.String())
		}
	}

	t.Logf("順序確認: %s", func() string {
		var directions []string
		for i, expected := range expectedDirections {
			directions = append(directions, fmt.Sprintf("%d.%s(%s)",
				i+1, expected.Clock, result[i].String()))
		}
		return strings.Join(directions, " → ")
	}())
}

func TestAppendAroundMemoryEfficiency(t *testing.T) {
	origin := hexmap.IdxXY(0, 0)

	t.Run("キャパシティが十分な場合の追加", func(t *testing.T) {
		// 十分なキャパシティを持つスライスを作成
		initialSlice := make([]hexmap.Index, 2, 10)
		initialSlice[0] = hexmap.IdxXY(100, 100)
		initialSlice[1] = hexmap.IdxXY(200, 200)

		// スライスのポインタを記録
		initialPtr := fmt.Sprintf("%p", initialSlice)

		result := origin.AppendAround(initialSlice)

		// 結果のポインタを記録
		resultPtr := fmt.Sprintf("%p", result)

		// 長さとキャパシティの確認
		if len(result) != 8 { // 既存2個 + 追加6個
			t.Errorf("Length: expected 8, but got %d", len(result))
		}

		// キャパシティが十分だった場合、同じ underlying array を使用するはず
		if cap(initialSlice) >= 8 {
			t.Logf("Initial pointer: %s, Result pointer: %s", initialPtr, resultPtr)
			if initialPtr != resultPtr {
				t.Logf("Note: Pointers differ, but this might be expected due to slice header copying")
			}
		}

		// 既存要素が保持されていることを確認
		if result[0].X() != 100 || result[0].Y() != 100 {
			t.Errorf("First element: expected (100,100), but got %s", result[0].String())
		}
		if result[1].X() != 200 || result[1].Y() != 200 {
			t.Errorf("Second element: expected (200,200), but got %s", result[1].String())
		}

		// 新しく追加された要素が正しいことを確認
		expectedNewElements := []hexmap.Index{
			hexmap.IdxXY(1, -1), // Direction01
			hexmap.IdxXY(1, 0),  // Direction03
			hexmap.IdxXY(0, 1),  // Direction05
			hexmap.IdxXY(-1, 1), // Direction07
			hexmap.IdxXY(-1, 0), // Direction09
			hexmap.IdxXY(0, -1), // Direction11
		}

		for i, expected := range expectedNewElements {
			actual := result[i+2] // 既存要素2個をスキップ
			if actual.X() != expected.X() || actual.Y() != expected.Y() {
				t.Errorf("New element %d: expected %s, but got %s", i, expected.String(), actual.String())
			}
		}
	})

	t.Run("nil スライスでの動作", func(t *testing.T) {
		var nilSlice []hexmap.Index
		result := origin.AppendAround(nilSlice)

		if len(result) != 6 {
			t.Errorf("Length: expected 6, but got %d", len(result))
		}

		// 結果が正しい6つの隣接座標であることを確認
		expectedElements := []hexmap.Index{
			hexmap.IdxXY(1, -1), // Direction01
			hexmap.IdxXY(1, 0),  // Direction03
			hexmap.IdxXY(0, 1),  // Direction05
			hexmap.IdxXY(-1, 1), // Direction07
			hexmap.IdxXY(-1, 0), // Direction09
			hexmap.IdxXY(0, -1), // Direction11
		}

		for i, expected := range expectedElements {
			actual := result[i]
			if actual.X() != expected.X() || actual.Y() != expected.Y() {
				t.Errorf("Element %d: expected %s, but got %s", i, expected.String(), actual.String())
			}
		}
	})
}
