package hexmap

// Size represents the size of a hexagonal map in terms of radius or boundaries.
type Size struct {
	radius int // 中心からの半径（ヘクス数）
}

// NewSize は指定した半径でSizeを作成します。
func NewSize(radius int) Size {
	return Size{radius: radius}
}

// Radius は半径を返します。
func (s Size) Radius() int {
	return s.radius
}

// Contains は指定した座標がサイズ内に含まれるかどうかを返します。
func (s Size) Contains(idx Index) bool {
	origin := IdxXY(0, 0)
	distance := origin.Distance(idx)
	return distance <= s.radius
}
