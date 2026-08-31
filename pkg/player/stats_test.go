package player

import (
	"testing"
)

func TestPlayerStatsInitialValues(t *testing.T) {
	ps := NewPlayerStats()
	if ps.Health != 100 {
		t.Errorf("expected initial health 100, got %d", ps.Health)
	}
	if ps.Armor != 0 {
		t.Errorf("expected initial armor 0, got %d", ps.Armor)
	}
	if !ps.HasWeapon(WeaponFist) || !ps.HasWeapon(WeaponPistol) {
		t.Error("expected player to start with Fist and Pistol")
	}
	if ps.HasWeapon(WeaponShotgun) {
		t.Error("expected player to not have Shotgun initially")
	}
	if ps.ReadyWeapon != WeaponPistol {
		t.Errorf("expected ReadyWeapon Pistol, got %v", ps.ReadyWeapon)
	}
	if ps.Ammo[AmmoBullets] != 50 {
		t.Errorf("expected 50 bullets, got %d", ps.Ammo[AmmoBullets])
	}
	if ps.Ammo[AmmoShells] != 0 {
		t.Errorf("expected 0 shells, got %d", ps.Ammo[AmmoShells])
	}

	hasAmmo, cur, max := ps.GetReadyWeaponAmmo()
	if !hasAmmo || cur != 50 || max != 200 {
		t.Errorf("expected Pistol ammo (true, 50, 200), got (%v, %d, %d)", hasAmmo, cur, max)
	}

	if ps.GetFaceFrame() != "STFST00" && ps.GetFaceFrame() != "STFST01" && ps.GetFaceFrame() != "STFST02" {
		t.Errorf("expected Tier 0 glance face frame, got %s", ps.GetFaceFrame())
	}
}

func TestPlayerStatsDamageAndArmor(t *testing.T) {
	ps := NewPlayerStats()

	// Green armor (absorbs 1/3)
	ps.GiveArmor(100, 1)
	if ps.Armor != 100 || ps.ArmorType != 1 {
		t.Errorf("expected 100 green armor, got %d type %d", ps.Armor, ps.ArmorType)
	}

	// 30 damage: green armor absorbs 10, health loses 20
	ps.Damage(30)
	if ps.Armor != 90 {
		t.Errorf("expected armor 90, got %d", ps.Armor)
	}
	if ps.Health != 80 {
		t.Errorf("expected health 80, got %d", ps.Health)
	}
	if ps.DamageTimer <= 0 {
		t.Error("expected damage timer to be active after taking damage")
	}
	if ps.GetFaceFrame() != "STFOUCH0" {
		t.Errorf("expected STFOUCH0 face frame during damage, got %s", ps.GetFaceFrame())
	}

	// Blue armor upgrade
	ps.GiveArmor(100, 2)
	if ps.ArmorType != 2 {
		t.Errorf("expected blue armor type 2, got %d", ps.ArmorType)
	}

	// 40 damage: blue armor absorbs 20, health loses 20
	ps.Damage(40)
	if ps.Health != 60 {
		t.Errorf("expected health 60, got %d", ps.Health)
	}
	if ps.HealthTier() != 1 {
		t.Errorf("expected health tier 1 for 60 health, got %d", ps.HealthTier())
	}

	// Fatal damage
	ps.Damage(200)
	if ps.Health != 0 {
		t.Errorf("expected health 0 after lethal damage, got %d", ps.Health)
	}
	if ps.GetFaceFrame() != "STFDEAD0" {
		t.Errorf("expected STFDEAD0 on death, got %s", ps.GetFaceFrame())
	}
}

func TestPlayerStatsWeaponsAndBackpack(t *testing.T) {
	ps := NewPlayerStats()

	// Give Shotgun
	ps.GiveWeapon(WeaponShotgun)
	if !ps.HasWeapon(WeaponShotgun) {
		t.Error("expected player to have Shotgun")
	}
	if ps.ReadyWeapon != WeaponShotgun {
		t.Errorf("expected ReadyWeapon Shotgun, got %v", ps.ReadyWeapon)
	}
	if ps.Ammo[AmmoShells] != 8 {
		t.Errorf("expected 8 shells granted with Shotgun, got %d", ps.Ammo[AmmoShells])
	}
	if ps.EvilGrinTimer <= 0 {
		t.Error("expected EvilGrinTimer after picking up new weapon")
	}
	if ps.GetFaceFrame() != "STFEVL0" {
		t.Errorf("expected STFEVL0 face frame, got %s", ps.GetFaceFrame())
	}

	// Select weapon slots
	if !ps.SelectSlot(2) || ps.ReadyWeapon != WeaponPistol {
		t.Errorf("expected Slot 2 selection to switch to Pistol, got %v", ps.ReadyWeapon)
	}
	if !ps.SelectSlot(3) || ps.ReadyWeapon != WeaponShotgun {
		t.Errorf("expected Slot 3 selection to switch to Shotgun, got %v", ps.ReadyWeapon)
	}

	// Give Backpack
	ps.GiveBackpack()
	if !ps.HasBackpack {
		t.Error("expected HasBackpack true")
	}
	if ps.MaxAmmo[AmmoBullets] != BackpackBulletsMax || ps.MaxAmmo[AmmoShells] != BackpackShellsMax {
		t.Errorf("expected doubled max ammo: bullets %d, shells %d", ps.MaxAmmo[AmmoBullets], ps.MaxAmmo[AmmoShells])
	}

	// Give All (IDKFA)
	ps.GiveAll()
	if ps.Health != 200 || ps.Armor != 200 || ps.ArmorType != 2 {
		t.Errorf("expected 200/200 health/armor from GiveAll, got %d/%d", ps.Health, ps.Armor)
	}
	for w := 0; w < NumWeapons; w++ {
		if !ps.HasWeapon(WeaponType(w)) {
			t.Errorf("expected player to own weapon %d after GiveAll", w)
		}
	}
	for k := 0; k < NumKeys; k++ {
		if !ps.HasKey(KeyType(k)) {
			t.Errorf("expected player to own key %d after GiveAll", k)
		}
	}
}

func TestPlayerStatsFaceGlanceAndGodMode(t *testing.T) {
	ps := NewPlayerStats()
	ps.SetGodMode(true)
	if !ps.GodMode {
		t.Error("expected GodMode true")
	}
	if ps.GetFaceFrame() != "STFGOD0" {
		t.Errorf("expected STFGOD0 face in god mode, got %s", ps.GetFaceFrame())
	}

	// God mode prevents damage
	ps.Damage(50)
	if ps.Health != 100 {
		t.Errorf("expected health 100 under GodMode, got %d", ps.Health)
	}

	ps.SetGodMode(false)
	// Update ticks
	for i := 0; i < 300; i++ {
		ps.Update()
	}
	frame := ps.GetFaceFrame()
	if frame != "STFST00" && frame != "STFST01" && frame != "STFST02" {
		t.Errorf("expected valid glance frame, got %s", frame)
	}
}
