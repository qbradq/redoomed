package mode

import (
	"fmt"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/qbradq/redoomed/pkg/player"
	"github.com/qbradq/redoomed/pkg/wad"
)

// HUDAssets contains preloaded and cached Ebitengine textures for status bar rendering.
type HUDAssets struct {
	STBAR       *ebiten.Image
	STARMS      *ebiten.Image
	TallNums    [10]*ebiten.Image // STTNUM0 - STTNUM9
	TallMinus   *ebiten.Image     // STTMINUS
	TallPercent *ebiten.Image     // STTPRCNT
	SmallNums   [10]*ebiten.Image // STYSNUM0 - STYSNUM9 (yellow)
	GrayNums    [10]*ebiten.Image // STGNUM0 - STGNUM9 (gray)
	Keys        [9]*ebiten.Image  // STKEYS0 - STKEYS8
	Faces       map[string]*ebiten.Image
}

// NewHUDAssets loads status bar patches from the given WAD file.
func NewHUDAssets(w *wad.WAD) *HUDAssets {
	assets := &HUDAssets{
		Faces: make(map[string]*ebiten.Image),
	}
	if w == nil {
		return assets
	}

	// 1. Backgrounds
	if img, err := w.GetPatchImage("STBAR"); err == nil {
		assets.STBAR = img
	}
	if img, err := w.GetPatchImage("STARMS"); err == nil {
		assets.STARMS = img
	}

	// 2. Tall Numbers (STTNUM0 - STTNUM9)
	for i := 0; i <= 9; i++ {
		lump := fmt.Sprintf("STTNUM%d", i)
		if img, err := w.GetPatchImage(lump); err == nil {
			assets.TallNums[i] = img
		}
	}
	if img, err := w.GetPatchImage("STTMINUS"); err == nil {
		assets.TallMinus = img
	}
	if img, err := w.GetPatchImage("STTPRCNT"); err == nil {
		assets.TallPercent = img
	}

	// 3. Small Numbers (STYSNUM0 - STYSNUM9, STGNUM0 - STGNUM9)
	for i := 0; i <= 9; i++ {
		lumpY := fmt.Sprintf("STYSNUM%d", i)
		if img, err := w.GetPatchImage(lumpY); err == nil {
			assets.SmallNums[i] = img
		}
		lumpG := fmt.Sprintf("STGNUM%d", i)
		if img, err := w.GetPatchImage(lumpG); err == nil {
			assets.GrayNums[i] = img
		}
	}

	// 4. Keys (STKEYS0 - STKEYS8)
	for i := 0; i <= 8; i++ {
		lump := fmt.Sprintf("STKEYS%d", i)
		if img, err := w.GetPatchImage(lump); err == nil {
			assets.Keys[i] = img
		}
	}

	// 5. Mugshot Faces
	faceLumps := []string{
		"STFGOD0", "STFDEAD0",
	}
	for tier := 0; tier <= 4; tier++ {
		for dir := 0; dir <= 2; dir++ {
			faceLumps = append(faceLumps, fmt.Sprintf("STFST%d%d", tier, dir))
		}
		faceLumps = append(faceLumps, fmt.Sprintf("STFOUCH%d", tier))
		faceLumps = append(faceLumps, fmt.Sprintf("STFEVL%d", tier))
		faceLumps = append(faceLumps, fmt.Sprintf("STFKILL%d", tier))
		faceLumps = append(faceLumps, fmt.Sprintf("STFTL%d0", tier))
		faceLumps = append(faceLumps, fmt.Sprintf("STFTR%d0", tier))
	}

	for _, lump := range faceLumps {
		if img, err := w.GetPatchImage(lump); err == nil {
			assets.Faces[lump] = img
		}
	}

	return assets
}

// DrawTallNumber renders a right-aligned integer using tall yellow numbers at rightX.
func (a *HUDAssets) DrawTallNumber(target *ebiten.Image, val int, rightX, y int) {
	if val < 0 {
		val = 0
	}
	str := strconv.Itoa(val)
	x := rightX
	for i := len(str) - 1; i >= 0; i-- {
		digit := str[i] - '0'
		if int(digit) < len(a.TallNums) && a.TallNums[digit] != nil {
			img := a.TallNums[digit]
			w := img.Bounds().Dx()
			x -= w
			op := &ebiten.DrawImageOptions{}
			op.Filter = ebiten.FilterNearest
			op.GeoM.Translate(float64(x), float64(y))
			target.DrawImage(img, op)
		}
	}
}

// DrawSmallNumber renders a right-aligned integer using small yellow numbers at rightX.
func (a *HUDAssets) DrawSmallNumber(target *ebiten.Image, val int, rightX, y int) {
	if val < 0 {
		val = 0
	}
	str := strconv.Itoa(val)
	x := rightX
	for i := len(str) - 1; i >= 0; i-- {
		digit := str[i] - '0'
		if int(digit) < len(a.SmallNums) && a.SmallNums[digit] != nil {
			img := a.SmallNums[digit]
			w := img.Bounds().Dx()
			x -= w
			op := &ebiten.DrawImageOptions{}
			op.Filter = ebiten.FilterNearest
			op.GeoM.Translate(float64(x), float64(y))
			target.DrawImage(img, op)
		}
	}
}

// DrawPercent draws the `%` sign at (x, y).
func (a *HUDAssets) DrawPercent(target *ebiten.Image, x, y int) {
	if a.TallPercent != nil {
		op := &ebiten.DrawImageOptions{}
		op.Filter = ebiten.FilterNearest
		op.GeoM.Translate(float64(x), float64(y))
		target.DrawImage(a.TallPercent, op)
	}
}

// DrawKey draws the key card or skull key for slot (0: Blue, 1: Yellow, 2: Red).
func (a *HUDAssets) DrawKey(target *ebiten.Image, slot int, hasCard, hasSkull bool, x, y int) {
	if !hasCard && !hasSkull {
		return
	}

	var keyImg *ebiten.Image

	if hasCard && hasSkull {
		// Combined card + skull lump index (STKEYS6, STKEYS7, STKEYS8)
		combIdx := 6 + slot
		if combIdx < len(a.Keys) && a.Keys[combIdx] != nil {
			keyImg = a.Keys[combIdx]
		}
	}

	if keyImg == nil && hasCard {
		// Card lump index (STKEYS0, STKEYS1, STKEYS2)
		cardIdx := slot
		if cardIdx < len(a.Keys) && a.Keys[cardIdx] != nil {
			keyImg = a.Keys[cardIdx]
		}
	}

	if keyImg == nil && hasSkull {
		// Skull lump index (STKEYS3, STKEYS4, STKEYS5)
		skullIdx := 3 + slot
		if skullIdx < len(a.Keys) && a.Keys[skullIdx] != nil {
			keyImg = a.Keys[skullIdx]
		}
	}

	if keyImg != nil {
		op := &ebiten.DrawImageOptions{}
		op.Filter = ebiten.FilterNearest
		op.GeoM.Translate(float64(x), float64(y))
		target.DrawImage(keyImg, op)
	}
}

// DrawArms renders the arms background and slots 2-7 weapon ownership indicators.
func (a *HUDAssets) DrawArms(target *ebiten.Image, ps *player.PlayerStats, x, y int) {
	if a.STARMS != nil {
		op := &ebiten.DrawImageOptions{}
		op.Filter = ebiten.FilterNearest
		op.GeoM.Translate(float64(x), float64(y))
		target.DrawImage(a.STARMS, op)
	}

	if ps == nil {
		return
	}

	// Slot coordinates relative to status bar (x=104, y=168):
	// Slot 2: (111, 172) -> dx = 7, dy = 4
	// Slot 3: (123, 172) -> dx = 19, dy = 4
	// Slot 4: (135, 172) -> dx = 31, dy = 4
	// Slot 5: (111, 182) -> dx = 7, dy = 14
	// Slot 6: (123, 182) -> dx = 19, dy = 14
	// Slot 7: (135, 182) -> dx = 31, dy = 14
	slotCoords := []struct {
		slot int
		dx   int
		dy   int
	}{
		{2, 7, 4},
		{3, 19, 4},
		{4, 31, 4},
		{5, 7, 14},
		{6, 19, 14},
		{7, 31, 14},
	}

	for _, sc := range slotCoords {
		hasW := ps.HasWeaponInSlot(sc.slot)
		var img *ebiten.Image
		if hasW && sc.slot < len(a.SmallNums) {
			img = a.SmallNums[sc.slot]
		} else if !hasW && sc.slot < len(a.GrayNums) && a.STARMS == nil {
			// If STARMS background is missing, draw gray number as fallback
			img = a.GrayNums[sc.slot]
		}
		if img != nil {
			op := &ebiten.DrawImageOptions{}
			op.Filter = ebiten.FilterNearest
			op.GeoM.Translate(float64(x+sc.dx), float64(y+sc.dy))
			target.DrawImage(img, op)
		}
	}
}

// DrawFace renders the current mugshot face patch frame at (x, y).
func (a *HUDAssets) DrawFace(target *ebiten.Image, frameName string, x, y int) {
	img, ok := a.Faces[frameName]
	if !ok || img == nil {
		// Fallback to basic straight face if specific animation is missing
		img = a.Faces["STFST00"]
	}
	if img != nil {
		op := &ebiten.DrawImageOptions{}
		op.Filter = ebiten.FilterNearest
		op.GeoM.Translate(float64(x), float64(y))
		target.DrawImage(img, op)
	}
}
