package physics

import (
	"math"

	"github.com/qbradq/redoomed/pkg/wad"
)

// DefaultUseRange is the standard Doom line interaction reach (64 units).
const DefaultUseRange = 64.0

// UseLine casts a ray from the actor's position in their facing direction up to maxDistance,
// returning the usable linedef if one exists within range.
// Following classic Doom PTR_UseTraverse behavior:
// - If a 2-sided line has no special, the use ray passes through it to reach doors/switches behind it.
// - 1-sided lines and lines with specials block the use ray.
func UseLine(mapData *wad.MapData, actor *Actor, maxDistance float64) (lineIdx int, ld *wad.Linedef, dist float64, hit bool) {
	if mapData == nil || actor == nil || len(mapData.Linedefs) == 0 || len(mapData.Vertexes) == 0 {
		return -1, nil, 0, false
	}

	if maxDistance <= 0 {
		maxDistance = DefaultUseRange
	}

	rad := actor.Angle * math.Pi / 180.0
	dirX := math.Cos(rad)
	dirY := math.Sin(rad)

	p1X := actor.X
	p1Y := actor.Y
	p2X := p1X + dirX*maxDistance
	p2Y := p1Y + dirY*maxDistance

	rayMinX := math.Min(p1X, p2X)
	rayMaxX := math.Max(p1X, p2X)
	rayMinY := math.Min(p1Y, p2Y)
	rayMaxY := math.Max(p1Y, p2Y)

	type hitLine struct {
		index int
		line  *wad.Linedef
		dist  float64
	}

	var hits []hitLine

	for i := range mapData.Linedefs {
		l := &mapData.Linedefs[i]
		if int(l.V1) >= len(mapData.Vertexes) || int(l.V2) >= len(mapData.Vertexes) {
			continue
		}

		v1 := mapData.Vertexes[l.V1]
		v2 := mapData.Vertexes[l.V2]

		vx1, vy1 := float64(v1.X), float64(v1.Y)
		vx2, vy2 := float64(v2.X), float64(v2.Y)

		lineMinX := math.Min(vx1, vx2)
		lineMaxX := math.Max(vx1, vx2)
		lineMinY := math.Min(vy1, vy2)
		lineMaxY := math.Max(vy1, vy2)

		// Quick bounding box rejection
		if rayMaxX < lineMinX || rayMinX > lineMaxX || rayMaxY < lineMinY || rayMinY > lineMaxY {
			continue
		}

		t, u, intersects := segmentIntersection(p1X, p1Y, p2X, p2Y, vx1, vy1, vx2, vy2)
		if intersects && t >= 0 && t <= 1 && u >= 0 && u <= 1 {
			d := t * maxDistance
			hits = append(hits, hitLine{
				index: i,
				line:  l,
				dist:  d,
			})
		}
	}

	if len(hits) == 0 {
		return -1, nil, 0, false
	}

	// Sort hits by distance from player (closest first)
	for i := 0; i < len(hits)-1; i++ {
		for j := i + 1; j < len(hits); j++ {
			if hits[j].dist < hits[i].dist {
				hits[i], hits[j] = hits[j], hits[i]
			}
		}
	}

	// Traverse sorted hits in order from closest to farthest
	for _, h := range hits {
		// If line has a special, activate it
		if h.line.Special != 0 {
			return h.index, h.line, h.dist, true
		}

		// 2-sided line with no special: use ray passes through to check behind
		if h.line.LeftSide != 0xFFFF && (h.line.Flags&wad.LinedefTwoSided != 0) {
			continue
		}

		// 1-sided line with no special: blocks the use ray
		return h.index, h.line, h.dist, true
	}

	// If all intersected lines were 2-sided non-special lines, return closest hit
	return hits[0].index, hits[0].line, hits[0].dist, true
}


// segmentIntersection calculates parametric t on segment 1 (p1->p2) and u on segment 2 (v1->v2).
// Returns true if the lines are not parallel.
func segmentIntersection(p1x, p1y, p2x, p2y, v1x, v1y, v2x, v2y float64) (t float64, u float64, ok bool) {
	rx := p2x - p1x
	ry := p2y - p1y
	sx := v2x - v1x
	sy := v2y - v1y

	cross := rx*sy - ry*sx
	if math.Abs(cross) < 1e-9 {
		// Parallel or collinear
		return 0, 0, false
	}

	qpx := v1x - p1x
	qpy := v1y - p1y

	t = (qpx*sy - qpy*sx) / cross
	u = (qpx*ry - qpy*rx) / cross

	return t, u, true
}

// CheckCrossedLines tests for any linedefs intersected by the movement vector (p1x, p1y) -> (p2x, p2y).
// Returns the slice of linedef indices that were crossed.
func CheckCrossedLines(mapData *wad.MapData, p1x, p1y, p2x, p2y float64) []int {
	if mapData == nil || len(mapData.Linedefs) == 0 || len(mapData.Vertexes) == 0 {
		return nil
	}
	if p1x == p2x && p1y == p2y {
		return nil
	}

	pathMinX := math.Min(p1x, p2x)
	pathMaxX := math.Max(p1x, p2x)
	pathMinY := math.Min(p1y, p2y)
	pathMaxY := math.Max(p1y, p2y)

	var crossed []int
	for i := range mapData.Linedefs {
		l := &mapData.Linedefs[i]
		if int(l.V1) >= len(mapData.Vertexes) || int(l.V2) >= len(mapData.Vertexes) {
			continue
		}

		v1 := mapData.Vertexes[l.V1]
		v2 := mapData.Vertexes[l.V2]

		vx1, vy1 := float64(v1.X), float64(v1.Y)
		vx2, vy2 := float64(v2.X), float64(v2.Y)

		lineMinX := math.Min(vx1, vx2)
		lineMaxX := math.Max(vx1, vx2)
		lineMinY := math.Min(vy1, vy2)
		lineMaxY := math.Max(vy1, vy2)

		if pathMaxX < lineMinX || pathMinX > lineMaxX || pathMaxY < lineMinY || pathMinY > lineMaxY {
			continue
		}

		t, u, intersects := segmentIntersection(p1x, p1y, p2x, p2y, vx1, vy1, vx2, vy2)
		if intersects && t >= 0 && t <= 1 && u >= 0 && u <= 1 {
			crossed = append(crossed, i)
		}
	}

	return crossed
}

