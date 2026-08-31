package player

import (
	"testing"
)

func TestPlayerTryPickupKeys(t *testing.T) {
	ps := NewPlayerStats()

	// Blue card
	pickedUp, msg := ps.TryPickupItem(ThingKeyBlueCard)
	if !pickedUp || !ps.HasKey(KeyBlueCard) {
		t.Fatalf("expected blue card pickup, got %v (%s)", pickedUp, msg)
	}
	// Duplicate blue card
	dup, _ := ps.TryPickupItem(ThingKeyBlueCard)
	if dup {
		t.Errorf("expected duplicate blue card pickup to return false")
	}

	// Yellow card
	pickedUp, _ = ps.TryPickupItem(ThingKeyYellowCard)
	if !pickedUp || !ps.HasKey(KeyYellowCard) {
		t.Fatalf("expected yellow card pickup")
	}

	// Red card
	pickedUp, _ = ps.TryPickupItem(ThingKeyRedCard)
	if !pickedUp || !ps.HasKey(KeyRedCard) {
		t.Fatalf("expected red card pickup")
	}

	// Skull keys
	pickedUp, _ = ps.TryPickupItem(ThingKeyBlueSkull)
	if !pickedUp || !ps.HasKey(KeyBlueSkull) {
		t.Fatalf("expected blue skull pickup")
	}
	pickedUp, _ = ps.TryPickupItem(ThingKeyYellowSkull)
	if !pickedUp || !ps.HasKey(KeyYellowSkull) {
		t.Fatalf("expected yellow skull pickup")
	}
	pickedUp, _ = ps.TryPickupItem(ThingKeyRedSkull)
	if !pickedUp || !ps.HasKey(KeyRedSkull) {
		t.Fatalf("expected red skull pickup")
	}
}

func TestPlayerTryPickupArmor(t *testing.T) {
	ps := NewPlayerStats()
	ps.Armor = 0
	ps.ArmorType = 0

	// Armor bonus / shard: +1 armor, sets type to 1 if unarmored
	pickedUp, msg := ps.TryPickupItem(ThingArmorBonus)
	if !pickedUp || ps.Armor != 1 || ps.ArmorType != 1 {
		t.Fatalf("expected armor bonus pickup, got armor=%d, type=%d (%s)", ps.Armor, ps.ArmorType, msg)
	}

	// Green armor: sets armor to 100, type 1
	pickedUp, msg = ps.TryPickupItem(ThingGreenArmor)
	if !pickedUp || ps.Armor != 100 || ps.ArmorType != 1 {
		t.Fatalf("expected green armor pickup, got armor=%d, type=%d (%s)", ps.Armor, ps.ArmorType, msg)
	}

	// Duplicate green armor when at 100 should fail
	dup, _ := ps.TryPickupItem(ThingGreenArmor)
	if dup {
		t.Errorf("expected duplicate green armor at 100%% to fail")
	}

	// Blue armor / megaarmor: sets armor to 200, type 2
	pickedUp, msg = ps.TryPickupItem(ThingBlueArmor)
	if !pickedUp || ps.Armor != 200 || ps.ArmorType != 2 {
		t.Fatalf("expected blue armor pickup, got armor=%d, type=%d (%s)", ps.Armor, ps.ArmorType, msg)
	}

	// Armor bonus when at 200 should fail (super max reached)
	dup, _ = ps.TryPickupItem(ThingArmorBonus)
	if dup {
		t.Errorf("expected armor bonus at 200 armor to fail")
	}
}

func TestPlayerTryPickupHealth(t *testing.T) {
	ps := NewPlayerStats()
	ps.Health = 70

	// Stimpack: +10 up to 100
	pickedUp, _ := ps.TryPickupItem(ThingStimpack)
	if !pickedUp || ps.Health != 80 {
		t.Fatalf("expected stimpack to give +10, health is %d", ps.Health)
	}

	// Medikit: +25 up to 100
	pickedUp, _ = ps.TryPickupItem(ThingMedikit)
	if !pickedUp || ps.Health != 100 {
		t.Fatalf("expected medikit to top off at 100, health is %d", ps.Health)
	}

	// Stimpack at 100 should fail
	if ok, _ := ps.TryPickupItem(ThingStimpack); ok {
		t.Errorf("expected stimpack at 100 health to fail")
	}

	// Health bonus (vial): can exceed 100 up to 200
	pickedUp, _ = ps.TryPickupItem(ThingHealthBonus)
	if !pickedUp || ps.Health != 101 {
		t.Fatalf("expected health bonus to give +1 past 100, health is %d", ps.Health)
	}

	// Soulsphere / Supercharge: +100 up to 200
	pickedUp, _ = ps.TryPickupItem(ThingSoulsphere)
	if !pickedUp || ps.Health != 200 {
		t.Fatalf("expected soulsphere to give 200, health is %d", ps.Health)
	}

	// Health bonus at 200 should fail
	if ok, _ := ps.TryPickupItem(ThingHealthBonus); ok {
		t.Errorf("expected health bonus at 200 health to fail")
	}

	// Megasphere: restores 200 health & 200 blue armor
	ps.Health = 50
	ps.Armor = 20
	ps.ArmorType = 1
	pickedUp, _ = ps.TryPickupItem(ThingMegasphere)
	if !pickedUp || ps.Health != 200 || ps.Armor != 200 || ps.ArmorType != 2 {
		t.Fatalf("expected megasphere to grant 200 health and 200 blue armor, got health=%d, armor=%d, type=%d", ps.Health, ps.Armor, ps.ArmorType)
	}
}

func TestPlayerTryPickupAmmoAndBackpack(t *testing.T) {
	ps := NewPlayerStats()
	// Initial bullets is 50, max 200
	ps.Ammo[AmmoBullets] = 50

	// Clip (+10 bullets)
	pickedUp, _ := ps.TryPickupItem(ThingAmmoClip)
	if !pickedUp || ps.Ammo[AmmoBullets] != 60 {
		t.Fatalf("expected clip to give 10 bullets, got %d", ps.Ammo[AmmoBullets])
	}

	// Box of bullets (+50 bullets)
	pickedUp, _ = ps.TryPickupItem(ThingBoxBullets)
	if !pickedUp || ps.Ammo[AmmoBullets] != 110 {
		t.Fatalf("expected box of bullets to give 50 bullets, got %d", ps.Ammo[AmmoBullets])
	}

	// Shells (+4 shells)
	pickedUp, _ = ps.TryPickupItem(ThingShells)
	if !pickedUp || ps.Ammo[AmmoShells] != 4 {
		t.Fatalf("expected shells to give 4, got %d", ps.Ammo[AmmoShells])
	}

	// Box of shells (+20 shells)
	pickedUp, _ = ps.TryPickupItem(ThingBoxShells)
	if !pickedUp || ps.Ammo[AmmoShells] != 24 {
		t.Fatalf("expected box of shells to give 20, got %d", ps.Ammo[AmmoShells])
	}

	// Rocket (+1 rocket)
	pickedUp, _ = ps.TryPickupItem(ThingRocket)
	if !pickedUp || ps.Ammo[AmmoRockets] != 1 {
		t.Fatalf("expected rocket to give 1, got %d", ps.Ammo[AmmoRockets])
	}

	// Box of rockets (+5 rockets)
	pickedUp, _ = ps.TryPickupItem(ThingBoxRockets)
	if !pickedUp || ps.Ammo[AmmoRockets] != 6 {
		t.Fatalf("expected box of rockets to give 5, got %d", ps.Ammo[AmmoRockets])
	}

	// Cell (+20 cells)
	pickedUp, _ = ps.TryPickupItem(ThingCell)
	if !pickedUp || ps.Ammo[AmmoCells] != 20 {
		t.Fatalf("expected cell to give 20, got %d", ps.Ammo[AmmoCells])
	}

	// Cell pack (+100 cells)
	pickedUp, _ = ps.TryPickupItem(ThingCellPack)
	if !pickedUp || ps.Ammo[AmmoCells] != 120 {
		t.Fatalf("expected cell pack to give 100, got %d", ps.Ammo[AmmoCells])
	}

	// Backpack: doubles capacity and adds ammo
	pickedUp, _ = ps.TryPickupItem(ThingBackpack)
	if !pickedUp || !ps.HasBackpack || ps.MaxAmmo[AmmoBullets] != 400 {
		t.Fatalf("expected backpack pickup, max bullets=%d, hasBackpack=%v", ps.MaxAmmo[AmmoBullets], ps.HasBackpack)
	}
}

func TestPlayerTryPickupWeaponsAndPowerups(t *testing.T) {
	ps := NewPlayerStats()

	// Shotgun
	pickedUp, msg := ps.TryPickupItem(ThingWeaponShotgun)
	if !pickedUp || !ps.HasWeapon(WeaponShotgun) || ps.ReadyWeapon != WeaponShotgun {
		t.Fatalf("expected shotgun pickup and auto-switch, got %v (%s)", pickedUp, msg)
	}
	if ps.Ammo[AmmoShells] != 8 {
		t.Fatalf("expected initial 8 shells on shotgun pickup, got %d", ps.Ammo[AmmoShells])
	}

	// Chaingun
	pickedUp, _ = ps.TryPickupItem(ThingWeaponChaingun)
	if !pickedUp || !ps.HasWeapon(WeaponChaingun) || ps.ReadyWeapon != WeaponChaingun {
		t.Fatalf("expected chaingun pickup")
	}

	// Plasma
	pickedUp, _ = ps.TryPickupItem(ThingWeaponPlasma)
	if !pickedUp || !ps.HasWeapon(WeaponPlasma) || ps.ReadyWeapon != WeaponPlasma {
		t.Fatalf("expected plasma pickup")
	}

	// BFG
	pickedUp, _ = ps.TryPickupItem(ThingWeaponBFG)
	if !pickedUp || !ps.HasWeapon(WeaponBFG) || ps.ReadyWeapon != WeaponBFG {
		t.Fatalf("expected bfg pickup")
	}

	// Invulnerability
	pickedUp, _ = ps.TryPickupItem(ThingInvulnerability)
	if !pickedUp || !ps.GodMode {
		t.Fatalf("expected invulnerability to set god mode")
	}
}
