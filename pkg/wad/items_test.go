package wad

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestItemDefLookup(t *testing.T) {
	// Test recognized items
	tests := []struct {
		thingType int16
		expectedName string
		expectedCat  ItemCategory
	}{
		{ThingKeyBlueCard, "Blue Keycard", ItemCategoryKey},
		{ThingKeyYellowCard, "Yellow Keycard", ItemCategoryKey},
		{ThingKeyRedCard, "Red Keycard", ItemCategoryKey},
		{ThingKeyBlueSkull, "Blue Skull Key", ItemCategoryKey},
		{ThingKeyYellowSkull, "Yellow Skull Key", ItemCategoryKey},
		{ThingKeyRedSkull, "Red Skull Key", ItemCategoryKey},
		{ThingStimpack, "Stimpack", ItemCategoryHealth},
		{ThingMedikit, "Medikit", ItemCategoryHealth},
		{ThingHealthBonus, "Health Bonus", ItemCategoryHealth},
		{ThingSoulsphere, "Supercharge", ItemCategoryHealth},
		{ThingMegasphere, "MegaSphere", ItemCategoryHealth},
		{ThingBerserk, "Berserk", ItemCategoryHealth},
		{ThingArmorBonus, "Armor Bonus", ItemCategoryArmor},
		{ThingGreenArmor, "Green Armor", ItemCategoryArmor},
		{ThingBlueArmor, "MegaArmor", ItemCategoryArmor},
		{ThingAmmoClip, "Ammo Clip", ItemCategoryAmmo},
		{ThingBoxBullets, "Box of Bullets", ItemCategoryAmmo},
		{ThingShells, "4 Shotgun Shells", ItemCategoryAmmo},
		{ThingBoxShells, "Box of Shells", ItemCategoryAmmo},
		{ThingRocket, "Rocket", ItemCategoryAmmo},
		{ThingBoxRockets, "Box of Rockets", ItemCategoryAmmo},
		{ThingCell, "Energy Cell", ItemCategoryAmmo},
		{ThingCellPack, "Energy Cell Pack", ItemCategoryAmmo},
		{ThingBackpack, "Backpack", ItemCategoryAmmo},
		{ThingWeaponShotgun, "Shotgun", ItemCategoryWeapon},
		{ThingWeaponSuperShotgun, "Super Shotgun", ItemCategoryWeapon},
		{ThingWeaponChaingun, "Chaingun", ItemCategoryWeapon},
		{ThingWeaponRocketLauncher, "Rocket Launcher", ItemCategoryWeapon},
		{ThingWeaponPlasma, "Plasma Gun", ItemCategoryWeapon},
		{ThingWeaponBFG, "BFG 9000", ItemCategoryWeapon},
		{ThingWeaponChainsaw, "Chainsaw", ItemCategoryWeapon},
		{ThingInvulnerability, "Invulnerability", ItemCategoryPowerup},
		{ThingRadiationSuit, "Radiation Shielding Suit", ItemCategoryPowerup},
		{ThingComputerMap, "Computer Area Map", ItemCategoryPowerup},
		{ThingLiteAmp, "Light Amplification Visor", ItemCategoryPowerup},
		{ThingInvisibility, "Invisibility", ItemCategoryPowerup},
	}

	for _, tt := range tests {
		def, ok := LookupItemDef(tt.thingType)
		if !ok {
			t.Errorf("LookupItemDef(%d) returned not found", tt.thingType)
			continue
		}
		if def.Name != tt.expectedName {
			t.Errorf("LookupItemDef(%d).Name = %q, expected %q", tt.thingType, def.Name, tt.expectedName)
		}
		if def.Category != tt.expectedCat {
			t.Errorf("LookupItemDef(%d).Category = %v, expected %v", tt.thingType, def.Category, tt.expectedCat)
		}
		if !IsItem(tt.thingType) {
			t.Errorf("IsItem(%d) returned false, expected true", tt.thingType)
		}
	}

	// Test non-items (e.g. Player 1 start)
	if IsItem(ThingPlayer1Start) {
		t.Errorf("IsItem(ThingPlayer1Start) returned true, expected false")
	}
	if _, ok := LookupItemDef(ThingPlayer1Start); ok {
		t.Errorf("LookupItemDef(ThingPlayer1Start) returned ok, expected false")
	}
}

func TestParseMapItemsSynthetic(t *testing.T) {
	md := &MapData{
		Things: []Thing{
			{X: 100, Y: 200, Angle: 0, Type: ThingPlayer1Start},
			{X: 150, Y: 250, Angle: 0, Type: ThingKeyBlueCard},
			{X: 200, Y: 300, Angle: 0, Type: ThingArmorBonus},
			{X: 250, Y: 350, Angle: 0, Type: ThingStimpack},
		},
	}

	items := ParseMapItems(md)
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}

	if items[0].Def.Type != ThingKeyBlueCard || items[0].X != 150 || items[0].Y != 250 {
		t.Errorf("item 0 mismatch: %+v", items[0])
	}
	if items[1].Def.Type != ThingArmorBonus || items[1].X != 200 || items[1].Y != 300 {
		t.Errorf("item 1 mismatch: %+v", items[1])
	}
	if items[2].Def.Type != ThingStimpack || items[2].X != 250 || items[2].Y != 350 {
		t.Errorf("item 2 mismatch: %+v", items[2])
	}
}

func TestParseMapItemsFromFreedoom2(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Skip("unable to get caller path")
	}
	wadPath := filepath.Join(filepath.Dir(filename), "..", "..", "freedoom2.wad")

	w, err := Open(wadPath)
	if err != nil {
		t.Skipf("skipping freedoom2.wad test: %v", err)
	}
	defer w.Close()

	md, err := w.LoadMap("MAP01")
	if err != nil {
		t.Fatalf("LoadMap(MAP01) failed: %v", err)
	}

	items := ParseMapItems(md)
	if len(items) == 0 {
		t.Fatal("expected items in MAP01, found none")
	}

	var foundArmorBonus, foundShells, foundHealthBonus bool
	for _, item := range items {
		switch item.Def.Type {
		case ThingArmorBonus:
			foundArmorBonus = true
		case ThingShells, ThingBoxShells:
			foundShells = true
		case ThingHealthBonus:
			foundHealthBonus = true
		}
	}

	if !foundArmorBonus && !foundShells && !foundHealthBonus {
		t.Errorf("expected common items (armor bonus, shells, health bonus) in MAP01")
	}
}

func TestLookupThingSprite(t *testing.T) {
	// Item sprite lookup
	sprite, ok := LookupThingSprite(ThingKeyBlueCard)
	if !ok || sprite != "BKEYA0" {
		t.Errorf("expected BKEYA0 for ThingKeyBlueCard, got %q (ok=%v)", sprite, ok)
	}

	// Prop sprite lookup (e.g. explosive barrel)
	sprite, ok = LookupThingSprite(2035)
	if !ok || sprite != "BAR1A0" {
		t.Errorf("expected BAR1A0 for barrel (2035), got %q (ok=%v)", sprite, ok)
	}

	// Torch lookup
	sprite, ok = LookupThingSprite(44)
	if !ok || sprite != "TBULA0" {
		t.Errorf("expected TBULA0 for tall blue torch (44), got %q (ok=%v)", sprite, ok)
	}

	// Unknown thing lookup
	_, ok = LookupThingSprite(9999)
	if ok {
		t.Errorf("expected false for unknown thing type 9999")
	}
}

func TestTextureManagerGetPatch(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	wadPath := filepath.Join(filepath.Dir(filename), "../../freedoom2.wad")
	w, err := Open(wadPath)
	if err != nil {
		t.Skipf("skipping Freedoom2 test: %v", err)
	}
	defer w.Close()

	tm, err := NewTextureManager(w)
	if err != nil {
		t.Fatalf("NewTextureManager failed: %v", err)
	}

	// Fetch sprite patch
	patch, err := tm.GetPatch("BON2A0")
	if err != nil {
		t.Fatalf("GetPatch(BON2A0) failed: %v", err)
	}
	t.Logf("BON2A0: width=%d, height=%d, left=%d, top=%d", patch.Width, patch.Height, patch.LeftOffset, patch.TopOffset)

	for _, name := range []string{"STIMA0", "MEDIA0", "BKEYA0", "BAR1A0", "BON1A0"} {
		if p, err := tm.GetPatch(name); err == nil {
			t.Logf("%s: width=%d, height=%d, left=%d, top=%d", name, p.Width, p.Height, p.LeftOffset, p.TopOffset)
		}
	}

	// Fetch again to test cache hit
	patch2, err := tm.GetPatch("BON2A0")
	if err != nil || patch2 != patch {
		t.Errorf("expected cached patch instance, got err=%v, match=%v", err, patch2 == patch)
	}
}

