package render

import (
	"image/color"
	"math"
	"sort"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/qbradq/redoomed/pkg/wad"
)

const (
	// DefaultViewWidth is the horizontal resolution of the 2.5D renderer.
	DefaultViewWidth = 320
	// DefaultViewHeight is the active 2.5D viewport height (leaving 32px for status bar).
	DefaultViewHeight = 168
	// DefaultBufferHeight is the full 320x200 game buffer height.
	DefaultBufferHeight = 200
	// DefaultFOV is the Doom 90-degree field of view in degrees.
	DefaultFOV = 90.0
	// DefaultPlayerEyeHeight is the player eye height above the sector floor in Doom units.
	DefaultPlayerEyeHeight = 41.0
)

// Camera represents the 2.5D camera viewpoint.
type Camera struct {
	X         float64 // World X coordinate
	Y         float64 // World Y coordinate
	Z         float64 // World Z coordinate (eye level)
	Angle     float64 // View angle in degrees: 0=East (+X), 90=North (+Y), 180=West (-X), 270=South (-Y)
	EyeHeight float64 // Eye height offset above sector floor
}

type maskedSeg struct {
	seg        *wad.Seg
	ld         *wad.Linedef
	frontSide  *wad.Sidedef
	frontSec   *wad.Sector
	backSec    *wad.Sector
	midTex     *wad.Texture
	xStart     int
	xEnd       int
	tx1        float64
	tz1        float64
	dxCam      float64
	dzCam      float64
	uOffset1   float64
	du         float64
	clipOffset int
}

// Renderer performs perspective-correct front-to-back 2.5D software rendering of Doom maps.
type Renderer struct {
	viewWidth   int
	viewHeight  int
	bufHeight   int
	focalLength float64
	centerX     float64
	centerY     float64

	// Precomputed column angles, tangents (kx = (x - centerX)/focalLength), and cosines
	colAngles []float64
	colKx     []float64
	colCos    []float64

	// Per-column 1D occlusion clipping buffers
	ceilingClip []int
	floorClip   []int

	// 2D depth buffer for precise per-pixel occlusion between walls, floors, ceilings, and sprites
	depthBuffer []float64

	// Queued 2-sided masked segs (window grates, iron bars) for back-to-front rendering pass
	maskedSegs       []maskedSeg
	maskedClipTop    []int
	maskedClipBottom []int

	// Active item entities for Thing rendering
	items []*wad.ItemEntity

	// Pixel RGBA buffer for fast WritePixels
	pixels []byte
}

// NewRenderer creates and initializes a 2.5D software renderer.
func NewRenderer(viewWidth, viewHeight, bufHeight int) *Renderer {
	if viewWidth <= 0 {
		viewWidth = DefaultViewWidth
	}
	if viewHeight <= 0 {
		viewHeight = DefaultViewHeight
	}
	if bufHeight <= 0 {
		bufHeight = DefaultBufferHeight
	}

	halfW := float64(viewWidth) / 2.0
	// For 90 degree FOV: tan(45 deg) = 1.0 -> focalLength = halfW
	focalLength := halfW / math.Tan((DefaultFOV/2.0)*math.Pi/180.0)
	centerX := halfW
	centerY := float64(viewHeight) / 2.0

	colAngles := make([]float64, viewWidth)
	colKx := make([]float64, viewWidth)
	colCos := make([]float64, viewWidth)

	for x := 0; x < viewWidth; x++ {
		angle := math.Atan2(float64(x)-centerX, focalLength)
		colAngles[x] = angle
		colKx[x] = (float64(x) - centerX) / focalLength
		colCos[x] = math.Cos(angle)
	}

	depthBuf := make([]float64, viewWidth*bufHeight)
	for i := range depthBuf {
		depthBuf[i] = math.MaxFloat64
	}

	return &Renderer{
		viewWidth:   viewWidth,
		viewHeight:  viewHeight,
		bufHeight:   bufHeight,
		focalLength: focalLength,
		centerX:     centerX,
		centerY:     centerY,
		colAngles:   colAngles,
		colKx:       colKx,
		colCos:      colCos,
		ceilingClip: make([]int, viewWidth),
		floorClip:   make([]int, viewWidth),
		depthBuffer: depthBuf,
		pixels:      make([]byte, viewWidth*bufHeight*4),
	}
}

// SetDimensions updates the viewport dimensions and recalculates projection constants.
func (r *Renderer) SetDimensions(viewWidth, viewHeight, bufHeight int) {
	r.viewWidth = viewWidth
	r.viewHeight = viewHeight
	r.bufHeight = bufHeight

	halfW := float64(viewWidth) / 2.0
	r.focalLength = halfW / math.Tan((DefaultFOV/2.0)*math.Pi/180.0)
	r.centerX = halfW
	r.centerY = float64(viewHeight) / 2.0

	r.colAngles = make([]float64, viewWidth)
	r.colKx = make([]float64, viewWidth)
	r.colCos = make([]float64, viewWidth)
	for x := 0; x < viewWidth; x++ {
		angle := math.Atan2(float64(x)-r.centerX, r.focalLength)
		r.colAngles[x] = angle
		r.colKx[x] = (float64(x) - r.centerX) / r.focalLength
		r.colCos[x] = math.Cos(angle)
	}

	r.ceilingClip = make([]int, viewWidth)
	r.floorClip = make([]int, viewWidth)
	r.depthBuffer = make([]float64, viewWidth*bufHeight)
	for i := range r.depthBuffer {
		r.depthBuffer[i] = math.MaxFloat64
	}
	r.pixels = make([]byte, viewWidth*bufHeight*4)
}

// SetItems updates the active map item entities for Thing rendering.
func (r *Renderer) SetItems(items []*wad.ItemEntity) {
	r.items = items
}

// Items returns the active map item entities.
func (r *Renderer) Items() []*wad.ItemEntity {
	return r.items
}

// Clear fills the pixel buffer with a solid background color and resets the depth buffer.
func (r *Renderer) Clear(c color.RGBA) {
	for i := 0; i < len(r.pixels); i += 4 {
		r.pixels[i] = c.R
		r.pixels[i+1] = c.G
		r.pixels[i+2] = c.B
		r.pixels[i+3] = c.A
	}
	for i := range r.depthBuffer {
		r.depthBuffer[i] = math.MaxFloat64
	}
}

// setPixel sets a pixel at (x, y) with color c.
func (r *Renderer) setPixel(x, y int, c color.RGBA) {
	if x < 0 || x >= r.viewWidth || y < 0 || y >= r.viewHeight {
		return
	}
	idx := (y*r.viewWidth + x) * 4
	r.pixels[idx] = c.R
	r.pixels[idx+1] = c.G
	r.pixels[idx+2] = c.B
	r.pixels[idx+3] = c.A
}

// Render renders the map scene from the camera's perspective onto target.
func (r *Renderer) Render(target *ebiten.Image, mapData *wad.MapData, cam *Camera) {
	if target == nil || mapData == nil || cam == nil {
		return
	}

	// 1. Clear 1D column occlusion clipping buffer, depth buffer, and clear view area
	for x := 0; x < r.viewWidth; x++ {
		r.ceilingClip[x] = -1
		r.floorClip[x] = r.viewHeight
	}
	for i := 0; i < r.viewWidth*r.viewHeight; i++ {
		r.depthBuffer[i] = math.MaxFloat64
	}

	// Clear viewport pixels (black)
	for y := 0; y < r.viewHeight; y++ {
		for x := 0; x < r.viewWidth; x++ {
			idx := (y*r.viewWidth + x) * 4
			r.pixels[idx] = 0
			r.pixels[idx+1] = 0
			r.pixels[idx+2] = 0
			r.pixels[idx+3] = 255
		}
	}

	// Clear masked seg lists for this frame
	r.maskedSegs = r.maskedSegs[:0]
	r.maskedClipTop = r.maskedClipTop[:0]
	r.maskedClipBottom = r.maskedClipBottom[:0]

	// 2. Adjust camera Z to sector floor height if uninitialized
	if cam.Z == 0 {
		if sec, ok := mapData.SectorAt(cam.X, cam.Y); ok && sec != nil {
			eyeH := cam.EyeHeight
			if eyeH <= 0 {
				eyeH = DefaultPlayerEyeHeight
			}
			cam.Z = float64(sec.FloorHeight) + eyeH
		}
	}

	// 3. Traverse BSP front-to-back
	if len(mapData.Nodes) > 0 {
		r.renderNode(mapData, cam, len(mapData.Nodes)-1)
	} else if len(mapData.Subsectors) > 0 {
		r.renderSubsector(mapData, cam, 0)
	}

	// 4. Draw masked segs (2-sided middle textures like grates/bars) back-to-front
	r.drawMaskedSegs(mapData, cam)

	// 5. Draw things (sprites/billboards) back-to-front
	r.drawThings(mapData, cam)

	// 6. Upload rendered pixel buffer to target image
	target.WritePixels(r.pixels)
}

// isRangeOccluded checks if all columns in [x1, x2] are fully clipped.
func (r *Renderer) isRangeOccluded(x1, x2 int) bool {
	if x1 < 0 {
		x1 = 0
	}
	if x2 >= r.viewWidth {
		x2 = r.viewWidth - 1
	}
	if x1 > x2 {
		return true
	}
	for x := x1; x <= x2; x++ {
		if r.ceilingClip[x] < r.floorClip[x]-1 {
			return false
		}
	}
	return true
}

func (r *Renderer) renderNode(mapData *wad.MapData, cam *Camera, nodeIdx int) {
	if nodeIdx < 0 || nodeIdx >= len(mapData.Nodes) {
		return
	}

	node := &mapData.Nodes[nodeIdx]

	// Determine camera side relative to node partition line
	dx := cam.X - float64(node.PartitionX)
	dy := cam.Y - float64(node.PartitionY)
	left := float64(node.ChangeY) * dx
	right := dy * float64(node.ChangeX)

	var nearChild, farChild uint16
	var nearBBox, farBBox *[4]int16

	if right < left {
		// Front / Right side is nearer
		nearChild = node.RightChild
		nearBBox = &node.RightBoundingBox
		farChild = node.LeftChild
		farBBox = &node.LeftBoundingBox
	} else {
		// Back / Left side is nearer
		nearChild = node.LeftChild
		nearBBox = &node.LeftBoundingBox
		farChild = node.RightChild
		farBBox = &node.RightBoundingBox
	}

	// Render near subtree
	r.renderChild(mapData, cam, nearChild, nearBBox)

	// Render far subtree (if not occluded)
	r.renderChild(mapData, cam, farChild, farBBox)
}

func (r *Renderer) renderChild(mapData *wad.MapData, cam *Camera, child uint16, bbox *[4]int16) {
	// Check bounding box against view frustum and column occlusion
	if bbox != nil && !r.checkBoundingBox(cam, bbox) {
		return
	}

	if child&0x8000 != 0 {
		// Subsector leaf node
		subsectorIdx := int(child & 0x7FFF)
		r.renderSubsector(mapData, cam, subsectorIdx)
	} else {
		// Internal node
		r.renderNode(mapData, cam, int(child))
	}
}

// checkBoundingBox checks if a bounding box is in the view frustum and not fully occluded.
func (r *Renderer) checkBoundingBox(cam *Camera, bbox *[4]int16) bool {
	// bbox format: [Top/maxY, Bottom/minY, Left/minX, Right/maxX]
	maxY := float64(bbox[0])
	minY := float64(bbox[1])
	minX := float64(bbox[2])
	maxX := float64(bbox[3])

	// If camera is inside or near the bounding box (within player radius of 16 units),
	// never cull it as the geometry surrounds or is in immediate proximity to the player!
	const playerRadius = 16.0
	if cam.X >= minX-playerRadius && cam.X <= maxX+playerRadius &&
		cam.Y >= minY-playerRadius && cam.Y <= maxY+playerRadius {
		return true
	}

	// 4 box corners
	corners := [4][2]float64{
		{minX, minY},
		{maxX, minY},
		{minX, maxY},
		{maxX, maxY},
	}

	camRad := cam.Angle * math.Pi / 180.0
	cosA := math.Cos(camRad)
	sinA := math.Sin(camRad)

	minScreenX := float64(r.viewWidth)
	maxScreenX := -1.0
	hasFront := false
	hasBehind := false

	for _, c := range corners {
		rx := c[0] - cam.X
		ry := c[1] - cam.Y

		tz := rx*cosA + ry*sinA
		tx := rx*sinA - ry*cosA

		if tz > 1.0 {
			hasFront = true
			sx := r.centerX + tx*(r.focalLength/tz)
			if sx < minScreenX {
				minScreenX = sx
			}
			if sx > maxScreenX {
				maxScreenX = sx
			}
		} else {
			hasBehind = true
		}
	}

	// If all 4 corners are behind the camera near plane, the box is completely behind the player
	if !hasFront {
		return false
	}

	// If some corners are in front and some behind, the bounding box crosses the camera near plane.
	// In this case, the box extends across the view frustum, so we only check if the whole screen is occluded.
	if hasBehind {
		return !r.isRangeOccluded(0, r.viewWidth-1)
	}

	x1 := int(math.Floor(minScreenX))
	x2 := int(math.Ceil(maxScreenX))
	if x2 < 0 || x1 >= r.viewWidth {
		return false
	}

	return !r.isRangeOccluded(x1, x2)
}

func (r *Renderer) renderSubsector(mapData *wad.MapData, cam *Camera, ssIdx int) {
	if ssIdx < 0 || ssIdx >= len(mapData.Subsectors) {
		return
	}

	ss := &mapData.Subsectors[ssIdx]
	for i := 0; i < int(ss.NumSegs); i++ {
		segIdx := int(ss.FirstSeg) + i
		if segIdx < 0 || segIdx >= len(mapData.Segs) {
			continue
		}
		r.renderSeg(mapData, cam, &mapData.Segs[segIdx])
	}
}

func clampInt(val, min, max int) int {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}

func (r *Renderer) renderSeg(mapData *wad.MapData, cam *Camera, seg *wad.Seg) {
	if int(seg.V1) >= len(mapData.Vertexes) || int(seg.V2) >= len(mapData.Vertexes) {
		return
	}
	if int(seg.Linedef) >= len(mapData.Linedefs) {
		return
	}

	v1 := mapData.Vertexes[seg.V1]
	v2 := mapData.Vertexes[seg.V2]
	ld := &mapData.Linedefs[seg.Linedef]

	// Determine front and back sidedefs and sectors
	var frontSidedefIdx, backSidedefIdx uint16
	if seg.Direction == 0 {
		frontSidedefIdx = ld.RightSide
		backSidedefIdx = ld.LeftSide
	} else {
		frontSidedefIdx = ld.LeftSide
		backSidedefIdx = ld.RightSide
	}

	if frontSidedefIdx == 0xFFFF || int(frontSidedefIdx) >= len(mapData.Sidedefs) {
		return
	}
	frontSide := &mapData.Sidedefs[frontSidedefIdx]
	if int(frontSide.Sector) >= len(mapData.Sectors) {
		return
	}
	frontSec := &mapData.Sectors[frontSide.Sector]

	var backSide *wad.Sidedef
	var backSec *wad.Sector
	isTwoSided := backSidedefIdx != 0xFFFF && (ld.Flags&wad.LinedefTwoSided != 0)
	if isTwoSided && int(backSidedefIdx) < len(mapData.Sidedefs) {
		backSide = &mapData.Sidedefs[backSidedefIdx]
		if int(backSide.Sector) < len(mapData.Sectors) {
			backSec = &mapData.Sectors[backSide.Sector]
		}
	}

	// Transform seg vertices to camera coordinates
	camRad := cam.Angle * math.Pi / 180.0
	cosA := math.Cos(camRad)
	sinA := math.Sin(camRad)

	rx1 := float64(v1.X) - cam.X
	ry1 := float64(v1.Y) - cam.Y
	rx2 := float64(v2.X) - cam.X
	ry2 := float64(v2.Y) - cam.Y

	tz1 := rx1*cosA + ry1*sinA
	tz2 := rx2*cosA + ry2*sinA

	tx1 := rx1*sinA - ry1*cosA
	tx2 := rx2*sinA - ry2*cosA

	// Clip segment against near plane (tz >= 1.0)
	nearZ := 1.0

	// Both vertices behind near plane?
	if tz1 < nearZ && tz2 < nearZ {
		return
	}

	// Seg world length
	dx := float64(v2.X - v1.X)
	dy := float64(v2.Y - v1.Y)
	segLen := math.Sqrt(dx*dx + dy*dy)

	uOffset1 := 0.0
	uOffset2 := segLen

	if tz1 < nearZ {
		t := (nearZ - tz1) / (tz2 - tz1)
		tz1 = nearZ
		tx1 = tx1 + t*(tx2-tx1)
		uOffset1 = t * segLen
	} else if tz2 < nearZ {
		t := (nearZ - tz2) / (tz1 - tz2)
		tz2 = nearZ
		tx2 = tx2 + t*(tx1-tx2)
		uOffset2 = (1.0 - t) * segLen
	}

	// Project vertices to screen X
	sx1 := r.centerX + tx1*(r.focalLength/tz1)
	sx2 := r.centerX + tx2*(r.focalLength/tz2)

	// Back-facing seg check: front facing segs project with sx1 < sx2
	if sx1 >= sx2 {
		return
	}

	if sx2 < 0 || sx1 >= float64(r.viewWidth) {
		return
	}

	xStart := int(math.Max(0, math.Floor(sx1)))
	xEnd := int(math.Min(float64(r.viewWidth-1), math.Ceil(sx2-1)))
	if xStart > xEnd {
		return
	}

	// Texture, Flat, and Sky references
	texMgr := mapData.Textures
	var midTex, upperTex, lowerTex, skyTex *wad.Texture
	var floorFlat, ceilFlat *wad.Flat

	isFrontSky := strings.HasPrefix(frontSec.CeilingPic, "F_SKY")
	isBackSky := backSec != nil && strings.HasPrefix(backSec.CeilingPic, "F_SKY")

	if texMgr != nil {
		skyTex = texMgr.GetSkyTexture(mapData.Name)
		if frontSide.MiddleTexture != "" && frontSide.MiddleTexture != "-" {
			midTex, _ = texMgr.GetTexture(frontSide.MiddleTexture)
		}
		if frontSide.UpperTexture != "" && frontSide.UpperTexture != "-" {
			upperTex, _ = texMgr.GetTexture(frontSide.UpperTexture)
		}
		if frontSide.LowerTexture != "" && frontSide.LowerTexture != "-" {
			lowerTex, _ = texMgr.GetTexture(frontSide.LowerTexture)
		}
		if frontSec.FloorPic != "" && frontSec.FloorPic != "-" {
			floorFlat, _ = texMgr.GetFlat(frontSec.FloorPic)
		}
		if !isFrontSky && frontSec.CeilingPic != "" && frontSec.CeilingPic != "-" {
			ceilFlat, _ = texMgr.GetFlat(frontSec.CeilingPic)
		}
	}

	// 3D parametric vectors in camera space for exact perspective-correct wall mapping
	dxCam := tx2 - tx1
	dzCam := tz2 - tz1
	du := uOffset2 - uOffset1

	palette := [256]color.RGBA{}
	if texMgr != nil {
		palette = texMgr.PaletteRGBA()
	}

	for x := xStart; x <= xEnd; x++ {
		if r.ceilingClip[x] >= r.floorClip[x]-1 {
			continue
		}

		// Exact 3D ray-line parametric intersection along wall segment:
		// kx = (x - centerX) / focalLength = tx(s) / tz(s)
		// tx1 + s * dxCam = kx * (tz1 + s * dzCam)
		// s * (dxCam - kx * dzCam) = kx * tz1 - tx1
		kx := r.colKx[x]
		denom := dxCam - kx*dzCam
		if math.Abs(denom) < 1e-9 {
			continue
		}

		s := (kx*tz1 - tx1) / denom
		if s < 0.0 {
			s = 0.0
		} else if s > 1.0 {
			s = 1.0
		}

		tz := tz1 + s*dzCam
		if tz < 0.1 {
			tz = 0.1
		}
		scale := r.focalLength / tz

		// Exact 3D perspective-correct texture U
		uWorld := float64(seg.Offset) + uOffset1 + s*du + float64(frontSide.XOffset)
		texU := int(math.Floor(uWorld))

		// Screen Y projections for sector heights (clamped to sensible range)
		frontCeilY := int(math.Round(r.centerY - (float64(frontSec.CeilingHeight)-cam.Z)*scale))
		frontFloorY := int(math.Round(r.centerY - (float64(frontSec.FloorHeight)-cam.Z)*scale))

		frontCeilY = clampInt(frontCeilY, -1, r.viewHeight)
		frontFloorY = clampInt(frontFloorY, -1, r.viewHeight)

		if !isTwoSided || backSec == nil {
			// One-sided solid wall
			// 1. Ceiling / Sky span above frontCeilY
			ceilDrawTop := r.ceilingClip[x] + 1
			ceilDrawBottom := frontCeilY - 1
			if ceilDrawBottom > r.floorClip[x]-1 {
				ceilDrawBottom = r.floorClip[x] - 1
			}
			if ceilDrawBottom >= ceilDrawTop {
				if isFrontSky && skyTex != nil && texMgr != nil {
					r.drawSkySpan(cam, x, ceilDrawTop, ceilDrawBottom, skyTex, texMgr, palette)
				} else if ceilFlat != nil && texMgr != nil {
					r.drawCeilingSpan(cam, x, ceilDrawTop, ceilDrawBottom, float64(frontSec.CeilingHeight), int(frontSec.LightLevel), ceilFlat, texMgr, palette)
				}
			}

			// 2. Floor span below frontFloorY
			floorDrawTop := frontFloorY + 1
			floorDrawBottom := r.floorClip[x] - 1
			if floorDrawTop < r.ceilingClip[x]+1 {
				floorDrawTop = r.ceilingClip[x] + 1
			}
			if floorFlat != nil && texMgr != nil && floorDrawBottom >= floorDrawTop {
				r.drawFloorSpan(cam, x, floorDrawTop, floorDrawBottom, float64(frontSec.FloorHeight), int(frontSec.LightLevel), floorFlat, texMgr, palette)
			}

			// 3. Middle wall
			wallTop := frontCeilY
			if wallTop < r.ceilingClip[x]+1 {
				wallTop = r.ceilingClip[x] + 1
			}
			wallBottom := frontFloorY
			if wallBottom > r.floorClip[x]-1 {
				wallBottom = r.floorClip[x] - 1
			}

			wallTop = clampInt(wallTop, 0, r.viewHeight-1)
			wallBottom = clampInt(wallBottom, 0, r.viewHeight-1)

			if wallBottom >= wallTop {
				cmIdx := 0
				if texMgr != nil {
					cmIdx = texMgr.LightToColormap(int(frontSec.LightLevel), tz)
				}

				for y := wallTop; y <= wallBottom; y++ {
					var texV int
					if ld.Flags&wad.LinedefDontPegBottom != 0 {
						// Bottom unpegged: aligned to sector floor
						texV = int(math.Floor(float64(frontSec.FloorHeight) + float64(frontSide.YOffset) - (cam.Z - (float64(y)-r.centerY)/scale)))
					} else {
						// Top pegged: aligned to sector ceiling
						texV = int(math.Floor(float64(frontSec.CeilingHeight) + float64(frontSide.YOffset) - (cam.Z - (float64(y)-r.centerY)/scale)))
					}

					var clr color.RGBA
					if midTex != nil {
						palIdx, opaque := midTex.PixelAt(texU, texV)
						if opaque && texMgr != nil {
							clr = palette[texMgr.MapColor(cmIdx, palIdx)]
						}
					} else {
						clr = color.RGBA{R: 120, G: 120, B: 120, A: 255}
					}
					r.setPixel(x, y, clr)
					r.depthBuffer[y*r.viewWidth+x] = tz
				}
			}

			// Solid wall occludes column completely
			r.ceilingClip[x] = r.viewHeight
			r.floorClip[x] = -1
		} else {
			// Two-sided portal wall (step/window)
			backCeilY := int(math.Round(r.centerY - (float64(backSec.CeilingHeight)-cam.Z)*scale))
			backFloorY := int(math.Round(r.centerY - (float64(backSec.FloorHeight)-cam.Z)*scale))

			backCeilY = clampInt(backCeilY, -1, r.viewHeight)
			backFloorY = clampInt(backFloorY, -1, r.viewHeight)

			cmIdx := 0
			if texMgr != nil {
				cmIdx = texMgr.LightToColormap(int(frontSec.LightLevel), tz)
			}

			// 1. Ceiling & Upper wall
			if backSec.CeilingHeight < frontSec.CeilingHeight {
				// Upper wall hangs down from frontCeilY to backCeilY
				ceilDrawTop := r.ceilingClip[x] + 1
				ceilDrawBottom := frontCeilY - 1
				if ceilDrawBottom > r.floorClip[x]-1 {
					ceilDrawBottom = r.floorClip[x] - 1
				}
				if ceilDrawBottom >= ceilDrawTop {
					if isFrontSky && skyTex != nil && texMgr != nil {
						r.drawSkySpan(cam, x, ceilDrawTop, ceilDrawBottom, skyTex, texMgr, palette)
					} else if ceilFlat != nil && texMgr != nil {
						r.drawCeilingSpan(cam, x, ceilDrawTop, ceilDrawBottom, float64(frontSec.CeilingHeight), int(frontSec.LightLevel), ceilFlat, texMgr, palette)
					}
				}

				upperTop := frontCeilY
				if upperTop < r.ceilingClip[x]+1 {
					upperTop = r.ceilingClip[x] + 1
				}
				upperBottom := backCeilY
				if upperBottom > r.floorClip[x]-1 {
					upperBottom = r.floorClip[x] - 1
				}

				upperTop = clampInt(upperTop, 0, r.viewHeight-1)
				upperBottom = clampInt(upperBottom, 0, r.viewHeight-1)

				if upperBottom >= upperTop {
					if isFrontSky && isBackSky && skyTex != nil && texMgr != nil {
						// Looking through two sky ceilings: draw sky across opening
						r.drawSkySpan(cam, x, upperTop, upperBottom, skyTex, texMgr, palette)
					} else {
						for y := upperTop; y <= upperBottom; y++ {
							var texV int
							if ld.Flags&wad.LinedefDontPegTop != 0 {
								// Upper unpegged: aligned to front ceiling
								texV = int(math.Floor(float64(frontSec.CeilingHeight) + float64(frontSide.YOffset) - (cam.Z - (float64(y)-r.centerY)/scale)))
							} else {
								// Upper pegged: aligned to back ceiling
								texV = int(math.Floor(float64(backSec.CeilingHeight) + float64(frontSide.YOffset) - (cam.Z - (float64(y)-r.centerY)/scale)))
							}

							var clr color.RGBA
							if upperTex != nil {
								palIdx, opaque := upperTex.PixelAt(texU, texV)
								if opaque && texMgr != nil {
									clr = palette[texMgr.MapColor(cmIdx, palIdx)]
								}
							} else {
								clr = color.RGBA{R: 100, G: 100, B: 100, A: 255}
							}
							r.setPixel(x, y, clr)
							r.depthBuffer[y*r.viewWidth+x] = tz
						}
					}
				}

				if backCeilY > r.ceilingClip[x] {
					r.ceilingClip[x] = clampInt(backCeilY, -1, r.viewHeight)
				}
			} else {
				// Continuous ceiling across subsectors or ceiling step up
				ceilDrawTop := r.ceilingClip[x] + 1
				ceilDrawBottom := frontCeilY
				if ceilDrawBottom > r.floorClip[x]-1 {
					ceilDrawBottom = r.floorClip[x] - 1
				}
				if ceilDrawBottom >= ceilDrawTop {
					if isFrontSky && skyTex != nil && texMgr != nil {
						r.drawSkySpan(cam, x, ceilDrawTop, ceilDrawBottom, skyTex, texMgr, palette)
					} else if ceilFlat != nil && texMgr != nil {
						r.drawCeilingSpan(cam, x, ceilDrawTop, ceilDrawBottom, float64(frontSec.CeilingHeight), int(frontSec.LightLevel), ceilFlat, texMgr, palette)
					}
				}

				if frontCeilY > r.ceilingClip[x] {
					r.ceilingClip[x] = clampInt(frontCeilY, -1, r.viewHeight)
				}
			}

			// 2. Floor & Lower wall
			if backSec.FloorHeight > frontSec.FloorHeight {
				// Lower wall steps up from frontFloorY to backFloorY
				floorDrawTop := frontFloorY + 1
				floorDrawBottom := r.floorClip[x] - 1
				if floorDrawTop < r.ceilingClip[x]+1 {
					floorDrawTop = r.ceilingClip[x] + 1
				}
				if floorFlat != nil && texMgr != nil && floorDrawBottom >= floorDrawTop {
					r.drawFloorSpan(cam, x, floorDrawTop, floorDrawBottom, float64(frontSec.FloorHeight), int(frontSec.LightLevel), floorFlat, texMgr, palette)
				}

				lowerTop := backFloorY
				if lowerTop < r.ceilingClip[x]+1 {
					lowerTop = r.ceilingClip[x] + 1
				}
				lowerBottom := frontFloorY
				if lowerBottom > r.floorClip[x]-1 {
					lowerBottom = r.floorClip[x] - 1
				}

				lowerTop = clampInt(lowerTop, 0, r.viewHeight-1)
				lowerBottom = clampInt(lowerBottom, 0, r.viewHeight-1)

				if lowerBottom >= lowerTop {
					for y := lowerTop; y <= lowerBottom; y++ {
						var texV int
						if ld.Flags&wad.LinedefDontPegBottom != 0 {
							// Lower unpegged: aligned to front floor
							texV = int(math.Floor(float64(frontSec.FloorHeight) + float64(frontSide.YOffset) - (cam.Z - (float64(y)-r.centerY)/scale)))
						} else {
							// Lower pegged: aligned to back floor
							texV = int(math.Floor(float64(backSec.FloorHeight) + float64(frontSide.YOffset) - (cam.Z - (float64(y)-r.centerY)/scale)))
						}

						var clr color.RGBA
						if lowerTex != nil {
							palIdx, opaque := lowerTex.PixelAt(texU, texV)
							if opaque && texMgr != nil {
								clr = palette[texMgr.MapColor(cmIdx, palIdx)]
							}
						} else {
							clr = color.RGBA{R: 80, G: 80, B: 80, A: 255}
						}
						r.setPixel(x, y, clr)
						r.depthBuffer[y*r.viewWidth+x] = tz
					}
				}

				if backFloorY < r.floorClip[x] {
					r.floorClip[x] = clampInt(backFloorY, -1, r.viewHeight)
				}
			} else {
				// Continuous floor across subsectors or floor step down
				floorDrawTop := frontFloorY
				floorDrawBottom := r.floorClip[x] - 1
				if floorDrawTop < r.ceilingClip[x]+1 {
					floorDrawTop = r.ceilingClip[x] + 1
				}
				if floorFlat != nil && texMgr != nil && floorDrawBottom >= floorDrawTop {
					r.drawFloorSpan(cam, x, floorDrawTop, floorDrawBottom, float64(frontSec.FloorHeight), int(frontSec.LightLevel), floorFlat, texMgr, palette)
				}

				if frontFloorY < r.floorClip[x] {
					r.floorClip[x] = clampInt(frontFloorY, -1, r.viewHeight)
				}
			}
		}
	}

	// Record 2-sided seg with middle texture (window grates, iron bars) for back-to-front composite pass
	if isTwoSided && backSec != nil && midTex != nil {
		hasVisibleCol := false
		clipOffset := len(r.maskedClipTop)
		for x := xStart; x <= xEnd; x++ {
			cTop := r.ceilingClip[x]
			cBottom := r.floorClip[x]
			r.maskedClipTop = append(r.maskedClipTop, cTop)
			r.maskedClipBottom = append(r.maskedClipBottom, cBottom)
			if cTop < cBottom-1 {
				hasVisibleCol = true
			}
		}

		if hasVisibleCol {
			r.maskedSegs = append(r.maskedSegs, maskedSeg{
				seg:        seg,
				ld:         ld,
				frontSide:  frontSide,
				frontSec:   frontSec,
				backSec:    backSec,
				midTex:     midTex,
				xStart:     xStart,
				xEnd:       xEnd,
				tx1:        tx1,
				tz1:        tz1,
				dxCam:      dxCam,
				dzCam:      dzCam,
				uOffset1:   uOffset1,
				du:         du,
				clipOffset: clipOffset,
			})
		} else {
			r.maskedClipTop = r.maskedClipTop[:clipOffset]
			r.maskedClipBottom = r.maskedClipBottom[:clipOffset]
		}
	}
}

// drawMaskedSegs renders all queued 2-sided middle textures (window grates, iron bars, fences)
// in reverse order (back-to-front), respecting closer solid geometry occlusion clipping.
func (r *Renderer) drawMaskedSegs(mapData *wad.MapData, cam *Camera) {
	if len(r.maskedSegs) == 0 {
		return
	}

	texMgr := mapData.Textures
	palette := [256]color.RGBA{}
	if texMgr != nil {
		palette = texMgr.PaletteRGBA()
	}

	// Render in reverse order (back-to-front relative to front-to-back BSP traversal)
	for i := len(r.maskedSegs) - 1; i >= 0; i-- {
		ms := &r.maskedSegs[i]
		if ms.midTex == nil || ms.midTex.Height <= 0 || ms.midTex.Width <= 0 {
			continue
		}

		// Calculate world topZ and bottomZ based on standard Doom pegging rules
		var topZ, bottomZ float64
		if ms.ld.Flags&wad.LinedefDontPegBottom != 0 {
			// Lower unpegged: aligned to highest floor, extending upwards
			floorH := float64(ms.frontSec.FloorHeight)
			if ms.backSec != nil && float64(ms.backSec.FloorHeight) > floorH {
				floorH = float64(ms.backSec.FloorHeight)
			}
			bottomZ = floorH + float64(ms.frontSide.YOffset)
			topZ = bottomZ + float64(ms.midTex.Height)
		} else {
			// Top pegged: aligned to lowest ceiling, extending downwards
			ceilH := float64(ms.frontSec.CeilingHeight)
			if ms.backSec != nil && float64(ms.backSec.CeilingHeight) < ceilH {
				ceilH = float64(ms.backSec.CeilingHeight)
			}
			topZ = ceilH + float64(ms.frontSide.YOffset)
			bottomZ = topZ - float64(ms.midTex.Height)
		}

		for x := ms.xStart; x <= ms.xEnd; x++ {
			clipTop := r.maskedClipTop[ms.clipOffset+(x-ms.xStart)]
			clipBottom := r.maskedClipBottom[ms.clipOffset+(x-ms.xStart)]
			if clipTop >= clipBottom-1 {
				continue
			}

			kx := r.colKx[x]
			denom := ms.dxCam - kx*ms.dzCam
			if math.Abs(denom) < 1e-9 {
				continue
			}

			s := (kx*ms.tz1 - ms.tx1) / denom
			if s < 0.0 {
				s = 0.0
			} else if s > 1.0 {
				s = 1.0
			}

			tz := ms.tz1 + s*ms.dzCam
			if tz < 0.1 {
				tz = 0.1
			}
			scale := r.focalLength / tz

			uWorld := float64(ms.seg.Offset) + ms.uOffset1 + s*ms.du + float64(ms.frontSide.XOffset)
			texU := int(math.Floor(uWorld))

			midTopY := int(math.Round(r.centerY - (topZ-cam.Z)*scale))
			midBottomY := int(math.Round(r.centerY - (bottomZ-cam.Z)*scale))

			wallTop := midTopY
			if wallTop < clipTop+1 {
				wallTop = clipTop + 1
			}
			wallBottom := midBottomY
			if wallBottom > clipBottom-1 {
				wallBottom = clipBottom - 1
			}

			wallTop = clampInt(wallTop, 0, r.viewHeight-1)
			wallBottom = clampInt(wallBottom, 0, r.viewHeight-1)

			if wallBottom >= wallTop {
				cmIdx := 0
				if texMgr != nil {
					cmIdx = texMgr.LightToColormap(int(ms.frontSec.LightLevel), tz)
				}

				for y := wallTop; y <= wallBottom; y++ {
					idx := y*r.viewWidth + x
					if tz >= r.depthBuffer[idx] {
						continue
					}

					texV := int(math.Floor(topZ - (cam.Z - (float64(y)-r.centerY)/scale)))
					if texV < 0 || texV >= ms.midTex.Height {
						continue
					}

					palIdx, opaque := ms.midTex.PixelAt(texU, texV)
					if opaque {
						var clr color.RGBA
						if texMgr != nil {
							clr = palette[texMgr.MapColor(cmIdx, palIdx)]
						} else {
							clr = color.RGBA{R: 200, G: 200, B: 200, A: 255}
						}
						r.setPixel(x, y, clr)
						r.depthBuffer[idx] = tz
					}
				}
			}
		}
	}
}

func (r *Renderer) drawSkySpan(cam *Camera, x, yStart, yEnd int, skyTex *wad.Texture, texMgr *wad.TextureManager, palette [256]color.RGBA) {
	if skyTex == nil || yStart > yEnd {
		return
	}
	if yStart < 0 {
		yStart = 0
	}
	if yEnd >= r.viewHeight {
		yEnd = r.viewHeight - 1
	}
	if yStart > yEnd {
		return
	}

	colAngle := r.colAngles[x]
	// Screen right is clockwise (-colAngle), screen left is counter-clockwise (+colAngle)
	rayAngle := cam.Angle*math.Pi/180.0 - colAngle
	for rayAngle < 0 {
		rayAngle += 2.0 * math.Pi
	}
	for rayAngle >= 2.0*math.Pi {
		rayAngle -= 2.0 * math.Pi
	}

	// In Doom, 360 degrees covers 4 full repetitions (1024 units for a 256px wide texture)
	// 2*pi radians -> 4 * skyTex.Width
	skyU := int(math.Floor(rayAngle * (float64(skyTex.Width) * 4.0 / (2.0 * math.Pi))))

	for y := yStart; y <= yEnd; y++ {
		// Vertically aligned to screen row
		skyV := y
		palIdx, opaque := skyTex.PixelAt(skyU, skyV)
		if opaque {
			// Sky is always rendered full-bright (colormap 0)
			clr := palette[texMgr.MapColor(0, palIdx)]
			r.setPixel(x, y, clr)
		}
	}
}

func (r *Renderer) drawCeilingSpan(cam *Camera, x, yStart, yEnd int, ceilH float64, lightLevel int, flat *wad.Flat, texMgr *wad.TextureManager, palette [256]color.RGBA) {
	if yStart < 0 {
		yStart = 0
	}
	if yEnd >= r.viewHeight {
		yEnd = r.viewHeight - 1
	}
	if yStart > yEnd {
		return
	}

	hDiff := ceilH - cam.Z
	if hDiff <= 0 {
		return
	}

	colAngle := r.colAngles[x]
	cosCol := r.colCos[x]
	// Screen right is clockwise (-colAngle), screen left is counter-clockwise (+colAngle)
	rayAngle := cam.Angle*math.Pi/180.0 - colAngle
	rayCos := math.Cos(rayAngle)
	raySin := math.Sin(rayAngle)

	for y := yStart; y <= yEnd; y++ {
		dy := r.centerY - float64(y)
		if dy <= 0 {
			continue
		}
		dist := (hDiff * r.focalLength) / dy
		rDist := dist / cosCol

		wx := cam.X + rDist*rayCos
		wy := cam.Y + rDist*raySin

		u := int(math.Floor(wx)) & 63
		v := int(math.Floor(-wy)) & 63

		palIdx := flat.PixelAt(u, v)
		cmIdx := texMgr.LightToColormap(lightLevel, dist)
		clr := palette[texMgr.MapColor(cmIdx, palIdx)]
		r.setPixel(x, y, clr)
		r.depthBuffer[y*r.viewWidth+x] = dist
	}
}

func (r *Renderer) drawFloorSpan(cam *Camera, x, yStart, yEnd int, floorH float64, lightLevel int, flat *wad.Flat, texMgr *wad.TextureManager, palette [256]color.RGBA) {
	if yStart < 0 {
		yStart = 0
	}
	if yEnd >= r.viewHeight {
		yEnd = r.viewHeight - 1
	}
	if yStart > yEnd {
		return
	}

	hDiff := cam.Z - floorH
	if hDiff <= 0 {
		return
	}

	colAngle := r.colAngles[x]
	cosCol := r.colCos[x]
	// Screen right is clockwise (-colAngle), screen left is counter-clockwise (+colAngle)
	rayAngle := cam.Angle*math.Pi/180.0 - colAngle
	rayCos := math.Cos(rayAngle)
	raySin := math.Sin(rayAngle)

	for y := yStart; y <= yEnd; y++ {
		dy := float64(y) - r.centerY
		if dy <= 0 {
			continue
		}
		dist := (hDiff * r.focalLength) / dy
		rDist := dist / cosCol

		wx := cam.X + rDist*rayCos
		wy := cam.Y + rDist*raySin

		u := int(math.Floor(wx)) & 63
		v := int(math.Floor(-wy)) & 63

		palIdx := flat.PixelAt(u, v)
		cmIdx := texMgr.LightToColormap(lightLevel, dist)
		clr := palette[texMgr.MapColor(cmIdx, palIdx)]
		r.setPixel(x, y, clr)
		r.depthBuffer[y*r.viewWidth+x] = dist
	}
}

type spriteThing struct {
	x, y   float64
	floorZ float64
	radius float64
	tx, tz float64
	sprite string
}

// drawThings renders all visible map Things and items as camera-facing billboards,
// sorted back-to-front and clipped against solid wall geometry and colormap attenuation.
func (r *Renderer) drawThings(mapData *wad.MapData, cam *Camera) {
	if mapData == nil {
		return
	}

	texMgr := mapData.Textures
	palette := [256]color.RGBA{}
	if texMgr != nil {
		palette = texMgr.PaletteRGBA()
	}

	var thingsToRender []spriteThing

	rad := cam.Angle * math.Pi / 180.0
	cosA := math.Cos(rad)
	sinA := math.Sin(rad)

	// 1. Gather collectible items (if items tracked)
	if len(r.items) > 0 {
		for _, item := range r.items {
			if item == nil || item.Collected {
				continue
			}
			sprite := item.Def.Sprite
			if sprite == "" {
				continue
			}
			dx := item.X - cam.X
			dy := item.Y - cam.Y
			tz := dx*cosA + dy*sinA
			if tz < 1.0 {
				continue
			}
			tx := dx*sinA - dy*cosA
			radius := item.Radius
			if radius <= 0 {
				radius = 20.0
			}
			thingsToRender = append(thingsToRender, spriteThing{
				x:      item.X,
				y:      item.Y,
				floorZ: item.FloorZ,
				radius: radius,
				tx:     tx,
				tz:     tz,
				sprite: sprite,
			})
		}
	}

	// 2. Gather remaining map Things (e.g. barrels, props, torches, or items if r.items is empty)
	if len(mapData.Things) > 0 {
		for _, th := range mapData.Things {
			if len(r.items) > 0 && wad.IsItem(th.Type) {
				continue
			}
			sprite, ok := wad.LookupThingSprite(th.Type)
			if !ok || sprite == "" {
				continue
			}
			txWorld := float64(th.X)
			tyWorld := float64(th.Y)
			dx := txWorld - cam.X
			dy := tyWorld - cam.Y
			tz := dx*cosA + dy*sinA
			if tz < 1.0 {
				continue
			}
			tx := dx*sinA - dy*cosA

			floorZ := 0.0
			if sec, ok := mapData.SectorAt(txWorld, tyWorld); ok && sec != nil {
				floorZ = float64(sec.FloorHeight)
			}

			radius := 16.0
			if def, ok := wad.LookupItemDef(th.Type); ok && def.Radius > 0 {
				radius = def.Radius
			}

			thingsToRender = append(thingsToRender, spriteThing{
				x:      txWorld,
				y:      tyWorld,
				floorZ: floorZ,
				radius: radius,
				tx:     tx,
				tz:     tz,
				sprite: sprite,
			})
		}
	}

	if len(thingsToRender) == 0 {
		return
	}

	// Sort back-to-front (furthest depth tz drawn first)
	sort.Slice(thingsToRender, func(i, j int) bool {
		return thingsToRender[i].tz > thingsToRender[j].tz
	})

	for _, st := range thingsToRender {
		var patch *wad.Patch
		if texMgr != nil {
			p, err := texMgr.GetPatch(st.sprite)
			if err == nil {
				patch = p
			}
		}
		if patch == nil || patch.Width <= 0 || patch.Height <= 0 {
			continue
		}

		scale := r.focalLength / st.tz
		screenCenterX := r.centerX + (st.tx/st.tz)*r.focalLength

		x1 := int(math.Round(screenCenterX - float64(patch.LeftOffset)*scale))
		x2 := int(math.Round(screenCenterX + float64(patch.Width-patch.LeftOffset)*scale)) - 1
		if x2 < 0 || x1 >= r.viewWidth {
			continue
		}

		topZ := st.floorZ + float64(patch.TopOffset)
		bottomZ := topZ - float64(patch.Height)

		topY := int(math.Round(r.centerY - (topZ-cam.Z)*scale))
		bottomY := int(math.Round(r.centerY - (bottomZ-cam.Z)*scale))

		startX := clampInt(x1, 0, r.viewWidth-1)
		endX := clampInt(x2, 0, r.viewWidth-1)
		drawTop := clampInt(topY, 0, r.viewHeight-1)
		drawBottom := clampInt(bottomY, 0, r.viewHeight-1)
		if drawBottom < drawTop {
			continue
		}

		lightLevel := 160
		if sec, ok := mapData.SectorAt(st.x, st.y); ok && sec != nil {
			lightLevel = int(sec.LightLevel)
		}
		cmIdx := 0
		if texMgr != nil {
			cmIdx = texMgr.LightToColormap(lightLevel, st.tz)
		}

		patchLeftWorld := screenCenterX - float64(patch.LeftOffset)*scale
		patchTopWorld := r.centerY - (topZ-cam.Z)*scale
		depthThreshold := st.tz - st.radius

		for x := startX; x <= endX; x++ {
			texU := int(math.Floor((float64(x) - patchLeftWorld) / scale))
			if texU < 0 || texU >= patch.Width {
				continue
			}

			for y := drawTop; y <= drawBottom; y++ {
				idx := y*r.viewWidth + x
				if depthThreshold >= r.depthBuffer[idx] {
					continue
				}

				texV := int(math.Floor((float64(y) - patchTopWorld) / scale))
				if texV < 0 || texV >= patch.Height {
					continue
				}

				palIdx, opaque := patch.PixelAt(texU, texV)
				if !opaque {
					continue
				}

				var clr color.RGBA
				if texMgr != nil {
					clr = palette[texMgr.MapColor(cmIdx, palIdx)]
				} else {
					clr = color.RGBA{R: 200, G: 200, B: 200, A: 255}
				}
				r.setPixel(x, y, clr)
				r.depthBuffer[idx] = st.tz
			}
		}
	}
}
