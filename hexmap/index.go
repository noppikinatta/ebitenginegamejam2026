package hexmap

import "fmt"

// Index is a hexmap coordinate expressed in X,Y,Z.
type Index struct {
	x int
	y int
}

func IdxXY(x, y int) Index {
	return Index{x: x, y: y}
}

func IdxYZ(y, z int) Index {
	return Index{x: 0 - y - z, y: y}
}

func IdxZX(z, x int) Index {
	return Index{x: x, y: 0 - z - x}
}

func (i Index) X() int {
	return i.x
}

func (i Index) Y() int {
	return i.y
}

func (i Index) Z() int {
	return 0 - i.x - i.y
}

func (i Index) Add(other Index) Index {
	return IdxXY(i.x+other.x, i.y+other.y)
}

func (i Index) Mul(v int) Index {
	return IdxXY(i.x*v, i.y*v)
}

func (i Index) Distance(other Index) int {
	dx := distance(i.x, other.x)
	dy := distance(i.y, other.y)
	dz := distance(i.Z(), other.Z())

	return (dx + dy + dz) / 2
}

func distance(v1, v2 int) int {
	a := v1 - v2
	if a >= 0 {
		return a
	} else {
		return -a
	}
}

func (i Index) String() string {
	return fmt.Sprintf("(%d,%d,%d)", i.X(), i.Y(), i.Z())
}

var (
	Direction01 = Index{x: 1, y: -1} // Direction01 is 1 o'clock
	Direction03 = Index{x: 1, y: 0}  // Direction03 is 3 o'clock
	Direction05 = Index{x: 0, y: 1}  // Direction05 is 5 o'clock
	Direction07 = Index{x: -1, y: 1} // Direction07 is 7 o'clock
	Direction09 = Index{x: -1, y: 0} // Direction09 is 9 o'clock
	Direction11 = Index{x: 0, y: -1} // Direction11 is 11 o'clock
)

func (i Index) AppendAround(around []Index) []Index {
	around = append(around, i.Add(Direction01))
	around = append(around, i.Add(Direction03))
	around = append(around, i.Add(Direction05))
	around = append(around, i.Add(Direction07))
	around = append(around, i.Add(Direction09))
	around = append(around, i.Add(Direction11))
	return around
}
