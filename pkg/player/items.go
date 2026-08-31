package player

// Item Thing type constants matching wad.Thing constants for player item pickup dispatching.
const (
	ThingKeyBlueCard    int16 = 5
	ThingKeyYellowCard  int16 = 6
	ThingKeyRedCard     int16 = 13
	ThingKeyRedSkull    int16 = 38
	ThingKeyYellowSkull int16 = 39
	ThingKeyBlueSkull   int16 = 40

	ThingStimpack    int16 = 2011
	ThingMedikit     int16 = 2012
	ThingHealthBonus int16 = 2014
	ThingSoulsphere  int16 = 2024
	ThingMegasphere  int16 = 2045
	ThingBerserk     int16 = 2023

	ThingArmorBonus int16 = 2015
	ThingGreenArmor int16 = 2018
	ThingBlueArmor  int16 = 2019

	ThingAmmoClip   int16 = 2007
	ThingBoxBullets int16 = 2048
	ThingShells     int16 = 2008
	ThingBoxShells  int16 = 2049
	ThingRocket     int16 = 2010
	ThingBoxRockets int16 = 2046
	ThingCell       int16 = 2047
	ThingCellPack   int16 = 2044
	ThingBackpack   int16 = 8

	ThingWeaponShotgun        int16 = 2001
	ThingWeaponChaingun       int16 = 2002
	ThingWeaponRocketLauncher int16 = 2003
	ThingWeaponPlasma         int16 = 2004
	ThingWeaponChainsaw       int16 = 2005
	ThingWeaponBFG            int16 = 2006
	ThingWeaponSuperShotgun   int16 = 82

	ThingInvulnerability int16 = 2022
	ThingRadiationSuit    int16 = 2025
	ThingComputerMap      int16 = 2026
	ThingLiteAmp          int16 = 2027
	ThingInvisibility     int16 = 2028
)

// TryPickupItem tests whether the player can collect the given item type and applies its effects to player stats.
// Returns (true, message) if collected, or (false, "") if the player cannot benefit from the item.
func (ps *PlayerStats) TryPickupItem(itemType int16) (bool, string) {
	if ps == nil {
		return false, ""
	}

	switch itemType {
	// Keys
	case ThingKeyBlueCard:
		if !ps.Keys[KeyBlueCard] {
			ps.GiveKey(KeyBlueCard)
			return true, "Picked up a blue keycard."
		}
		return false, ""
	case ThingKeyYellowCard:
		if !ps.Keys[KeyYellowCard] {
			ps.GiveKey(KeyYellowCard)
			return true, "Picked up a yellow keycard."
		}
		return false, ""
	case ThingKeyRedCard:
		if !ps.Keys[KeyRedCard] {
			ps.GiveKey(KeyRedCard)
			return true, "Picked up a red keycard."
		}
		return false, ""
	case ThingKeyBlueSkull:
		if !ps.Keys[KeyBlueSkull] {
			ps.GiveKey(KeyBlueSkull)
			return true, "Picked up a blue skull key."
		}
		return false, ""
	case ThingKeyYellowSkull:
		if !ps.Keys[KeyYellowSkull] {
			ps.GiveKey(KeyYellowSkull)
			return true, "Picked up a yellow skull key."
		}
		return false, ""
	case ThingKeyRedSkull:
		if !ps.Keys[KeyRedSkull] {
			ps.GiveKey(KeyRedSkull)
			return true, "Picked up a red skull key."
		}
		return false, ""

	// Health
	case ThingStimpack:
		if ps.GiveHealth(10, MaxHealthStandard) {
			return true, "Picked up a stimpack."
		}
		return false, ""
	case ThingMedikit:
		if ps.GiveHealth(25, MaxHealthStandard) {
			return true, "Picked up a medikit."
		}
		return false, ""
	case ThingHealthBonus:
		if ps.GiveHealth(1, MaxHealthSuper) {
			return true, "Picked up a health bonus."
		}
		return false, ""
	case ThingSoulsphere:
		if ps.GiveHealth(100, MaxHealthSuper) {
			return true, "Supercharge!"
		}
		return false, ""
	case ThingMegasphere:
		if ps.Health < MaxHealthSuper || ps.Armor < MaxArmorSuper || ps.ArmorType < 2 {
			ps.Health = MaxHealthSuper
			ps.Armor = MaxArmorSuper
			ps.ArmorType = 2
			return true, "MegaSphere!"
		}
		return false, ""
	case ThingBerserk:
		ps.GiveHealth(MaxHealthStandard, MaxHealthStandard)
		ps.SelectWeapon(WeaponFist)
		return true, "Berserk!"

	// Armor
	case ThingArmorBonus:
		if ps.Armor < MaxArmorSuper {
			ps.Armor++
			if ps.ArmorType == 0 {
				ps.ArmorType = 1
			}
			return true, "Picked up an armor bonus."
		}
		return false, ""
	case ThingGreenArmor:
		if ps.Armor < MaxArmorStandard || ps.ArmorType < 1 {
			ps.Armor = MaxArmorStandard
			ps.ArmorType = 1
			return true, "Picked up the armor."
		}
		return false, ""
	case ThingBlueArmor:
		if ps.Armor < MaxArmorSuper || ps.ArmorType < 2 {
			ps.Armor = MaxArmorSuper
			ps.ArmorType = 2
			return true, "Picked up the MegaArmor!"
		}
		return false, ""

	// Ammo
	case ThingAmmoClip:
		if ps.GiveAmmo(AmmoBullets, 10) {
			return true, "Picked up an ammo clip."
		}
		return false, ""
	case ThingBoxBullets:
		if ps.GiveAmmo(AmmoBullets, 50) {
			return true, "Picked up a box of bullets."
		}
		return false, ""
	case ThingShells:
		if ps.GiveAmmo(AmmoShells, 4) {
			return true, "Picked up 4 shotgun shells."
		}
		return false, ""
	case ThingBoxShells:
		if ps.GiveAmmo(AmmoShells, 20) {
			return true, "Picked up a box of shotgun shells."
		}
		return false, ""
	case ThingRocket:
		if ps.GiveAmmo(AmmoRockets, 1) {
			return true, "Picked up a rocket."
		}
		return false, ""
	case ThingBoxRockets:
		if ps.GiveAmmo(AmmoRockets, 5) {
			return true, "Picked up a box of rockets."
		}
		return false, ""
	case ThingCell:
		if ps.GiveAmmo(AmmoCells, 20) {
			return true, "Picked up an energy cell."
		}
		return false, ""
	case ThingCellPack:
		if ps.GiveAmmo(AmmoCells, 100) {
			return true, "Picked up an energy pack."
		}
		return false, ""
	case ThingBackpack:
		hadBackpack := ps.HasBackpack
		g1 := ps.Ammo[AmmoBullets] < BackpackBulletsMax
		g2 := ps.Ammo[AmmoShells] < BackpackShellsMax
		g3 := ps.Ammo[AmmoRockets] < BackpackRocketsMax
		g4 := ps.Ammo[AmmoCells] < BackpackCellsMax
		if !hadBackpack || g1 || g2 || g3 || g4 {
			ps.GiveBackpack()
			return true, "Picked up a backpack full of ammo!"
		}
		return false, ""

	// Weapons
	case ThingWeaponShotgun:
		if !ps.HasWeapon(WeaponShotgun) {
			ps.GiveWeapon(WeaponShotgun)
			return true, "You got the shotgun!"
		}
		if ps.GiveAmmo(AmmoShells, 8) {
			return true, "You got the shotgun!"
		}
		return false, ""
	case ThingWeaponSuperShotgun:
		if !ps.HasWeapon(WeaponSuperShotgun) {
			ps.GiveWeapon(WeaponSuperShotgun)
			return true, "You got the super shotgun!"
		}
		if ps.GiveAmmo(AmmoShells, 8) {
			return true, "You got the super shotgun!"
		}
		return false, ""
	case ThingWeaponChaingun:
		if !ps.HasWeapon(WeaponChaingun) {
			ps.GiveWeapon(WeaponChaingun)
			return true, "You got the chaingun!"
		}
		if ps.GiveAmmo(AmmoBullets, 20) {
			return true, "You got the chaingun!"
		}
		return false, ""
	case ThingWeaponRocketLauncher:
		if !ps.HasWeapon(WeaponRocketLauncher) {
			ps.GiveWeapon(WeaponRocketLauncher)
			return true, "You got the rocket launcher!"
		}
		if ps.GiveAmmo(AmmoRockets, 2) {
			return true, "You got the rocket launcher!"
		}
		return false, ""
	case ThingWeaponPlasma:
		if !ps.HasWeapon(WeaponPlasma) {
			ps.GiveWeapon(WeaponPlasma)
			return true, "You got the plasma gun!"
		}
		if ps.GiveAmmo(AmmoCells, 40) {
			return true, "You got the plasma gun!"
		}
		return false, ""
	case ThingWeaponBFG:
		if !ps.HasWeapon(WeaponBFG) {
			ps.GiveWeapon(WeaponBFG)
			return true, "You got the BFG9000! Oh, yes."
		}
		if ps.GiveAmmo(AmmoCells, 40) {
			return true, "You got the BFG9000! Oh, yes."
		}
		return false, ""
	case ThingWeaponChainsaw:
		if !ps.HasWeapon(WeaponChainsaw) {
			ps.GiveWeapon(WeaponChainsaw)
			return true, "A chainsaw! Find some meat!"
		}
		return false, ""

	// Powerups
	case ThingInvulnerability:
		ps.SetGodMode(true)
		return true, "Invulnerability!"
	case ThingRadiationSuit:
		return true, "Radiation Shielding Suit"
	case ThingComputerMap:
		return true, "Computer Area Map"
	case ThingLiteAmp:
		return true, "Light Amplification Visor"
	case ThingInvisibility:
		return true, "Partial Invisibility"
	}

	return false, ""
}
