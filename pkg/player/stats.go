package player

import (
	"fmt"
	"math/rand"
)

// AmmoType represents the 4 classic Doom ammo pools.
type AmmoType int

const (
	AmmoBullets AmmoType = 0
	AmmoShells  AmmoType = 1
	AmmoRockets AmmoType = 2
	AmmoCells   AmmoType = 3
	NumAmmoTypes         = 4
)

// WeaponType represents classic Doom weapons.
type WeaponType int

const (
	WeaponFist           WeaponType = 0
	WeaponPistol         WeaponType = 1
	WeaponShotgun        WeaponType = 2
	WeaponChaingun       WeaponType = 3
	WeaponRocketLauncher WeaponType = 4
	WeaponPlasma         WeaponType = 5
	WeaponBFG            WeaponType = 6
	WeaponChainsaw       WeaponType = 7
	WeaponSuperShotgun   WeaponType = 8
	NumWeapons                      = 9
)

// KeyType represents Doom keycards and skull keys.
type KeyType int

const (
	KeyBlueCard    KeyType = 0
	KeyYellowCard  KeyType = 1
	KeyRedCard     KeyType = 2
	KeyBlueSkull   KeyType = 3
	KeyYellowSkull KeyType = 4
	KeyRedSkull    KeyType = 5
	NumKeys                = 6
)

const (
	MaxHealthStandard = 100
	MaxHealthSuper    = 200
	MaxArmorStandard  = 100
	MaxArmorSuper     = 200

	DefaultBulletsMax = 200
	DefaultShellsMax  = 50
	DefaultRocketsMax = 50
	DefaultCellsMax   = 300

	BackpackBulletsMax = 400
	BackpackShellsMax  = 100
	BackpackRocketsMax = 100
	BackpackCellsMax   = 600

	DefaultInitialBullets = 50
	DefaultInitialHealth  = 100
)

// WeaponSlot returns the Doom status bar HUD slot number (1-7) for a given weapon.
func WeaponSlot(w WeaponType) int {
	switch w {
	case WeaponFist, WeaponChainsaw:
		return 1
	case WeaponPistol:
		return 2
	case WeaponShotgun, WeaponSuperShotgun:
		return 3
	case WeaponChaingun:
		return 4
	case WeaponRocketLauncher:
		return 5
	case WeaponPlasma:
		return 6
	case WeaponBFG:
		return 7
	default:
		return 1
	}
}

// WeaponAmmoType returns the ammo pool used by a given weapon, or -1 if none.
func WeaponAmmoType(w WeaponType) AmmoType {
	switch w {
	case WeaponPistol, WeaponChaingun:
		return AmmoBullets
	case WeaponShotgun, WeaponSuperShotgun:
		return AmmoShells
	case WeaponRocketLauncher:
		return AmmoRockets
	case WeaponPlasma, WeaponBFG:
		return AmmoCells
	default:
		return -1
	}
}

// PlayerStats holds all player gameplay statistics, inventory, ammo, and status bar state.
type PlayerStats struct {
	Health      int
	Armor       int
	ArmorType   int // 0: None, 1: Green Armor (1/3 absorption), 2: Blue Armor (1/2 absorption)
	Weapons     [NumWeapons]bool
	ReadyWeapon WeaponType
	Ammo        [NumAmmoTypes]int
	MaxAmmo     [NumAmmoTypes]int
	HasBackpack bool
	Keys        [NumKeys]bool
	GodMode     bool

	// Mugshot Face Animation state
	GlanceDir     int // 0: Straight, 1: Look Right, 2: Look Left
	GlanceTimer   int // Ticks remaining for current glance direction
	DamageTimer   int // Ticks remaining for ouch/pain face
	EvilGrinTimer int // Ticks remaining for evil grin
	RampageTimer  int // Ticks remaining for firing rampage face
	RampageSide   int // 0: Left, 1: Right
	TotalTics     int
}

// NewPlayerStats creates a new PlayerStats initialized with starting Doom values.
func NewPlayerStats() *PlayerStats {
	ps := &PlayerStats{}
	ps.Reset()
	return ps
}

// Reset resets player stats to the beginning of a game/level.
func (ps *PlayerStats) Reset() {
	ps.Health = DefaultInitialHealth
	ps.Armor = 0
	ps.ArmorType = 0
	for i := 0; i < NumWeapons; i++ {
		ps.Weapons[i] = false
	}
	ps.Weapons[WeaponFist] = true
	ps.Weapons[WeaponPistol] = true
	ps.ReadyWeapon = WeaponPistol

	ps.MaxAmmo[AmmoBullets] = DefaultBulletsMax
	ps.MaxAmmo[AmmoShells] = DefaultShellsMax
	ps.MaxAmmo[AmmoRockets] = DefaultRocketsMax
	ps.MaxAmmo[AmmoCells] = DefaultCellsMax

	for i := 0; i < NumAmmoTypes; i++ {
		ps.Ammo[i] = 0
	}
	ps.Ammo[AmmoBullets] = DefaultInitialBullets

	ps.HasBackpack = false
	for i := 0; i < NumKeys; i++ {
		ps.Keys[i] = false
	}
	ps.GodMode = false

	ps.GlanceDir = 0
	ps.GlanceTimer = 90 + rand.Intn(60)
	ps.DamageTimer = 0
	ps.EvilGrinTimer = 0
	ps.RampageTimer = 0
	ps.RampageSide = 0
	ps.TotalTics = 0
}

// GiveHealth adds health up to maxHealth (or standard 100/200). Returns true if health was increased.
func (ps *PlayerStats) GiveHealth(amount, maxHealth int) bool {
	if amount <= 0 {
		return false
	}
	if maxHealth <= 0 {
		maxHealth = MaxHealthStandard
	}
	if ps.Health >= maxHealth {
		return false
	}
	ps.Health += amount
	if ps.Health > maxHealth {
		ps.Health = maxHealth
	}
	return true
}

// Damage applies damage to the player, reducing armor and health according to armor absorption.
func (ps *PlayerStats) Damage(amount int) {
	if ps.GodMode || ps.Health <= 0 || amount <= 0 {
		return
	}

	// Trigger pain/ouch mugshot face (approx 35 tics / ~0.6 sec)
	ps.DamageTimer = 35

	var absorbed int
	if ps.Armor > 0 && ps.ArmorType > 0 {
		if ps.ArmorType == 1 {
			// Green armor absorbs 1/3 of damage
			absorbed = amount / 3
		} else {
			// Blue armor absorbs 1/2 of damage
			absorbed = amount / 2
		}
		if absorbed > ps.Armor {
			absorbed = ps.Armor
		}
		ps.Armor -= absorbed
		if ps.Armor == 0 {
			ps.ArmorType = 0
		}
	}

	damageToHealth := amount - absorbed
	ps.Health -= damageToHealth
	if ps.Health < 0 {
		ps.Health = 0
	}
}

// GiveArmor gives armor points and sets armor type (1: Green, 2: Blue). Returns true if armor was given/upgraded.
func (ps *PlayerStats) GiveArmor(amount, armorType int) bool {
	if amount <= 0 {
		return false
	}
	maxArmor := MaxArmorStandard
	if armorType >= 2 {
		maxArmor = MaxArmorSuper
	}

	if ps.Armor >= maxArmor && ps.ArmorType >= armorType {
		return false
	}

	ps.Armor += amount
	if ps.Armor > maxArmor {
		ps.Armor = maxArmor
	}
	if armorType > ps.ArmorType || ps.ArmorType == 0 {
		ps.ArmorType = armorType
	}
	return true
}

// GiveWeapon grants the specified weapon and standard initial ammo, triggering an evil grin.
func (ps *PlayerStats) GiveWeapon(w WeaponType) bool {
	if w < 0 || int(w) >= NumWeapons {
		return false
	}
	wasOwned := ps.Weapons[w]
	ps.Weapons[w] = true

	// Grant standard weapon ammo on first acquisition
	if !wasOwned {
		ps.EvilGrinTimer = 45 // ~0.75s grin
		switch w {
		case WeaponShotgun:
			ps.GiveAmmo(AmmoShells, 8)
		case WeaponSuperShotgun:
			ps.GiveAmmo(AmmoShells, 8)
		case WeaponChaingun:
			ps.GiveAmmo(AmmoBullets, 20)
		case WeaponRocketLauncher:
			ps.GiveAmmo(AmmoRockets, 2)
		case WeaponPlasma:
			ps.GiveAmmo(AmmoCells, 40)
		case WeaponBFG:
			ps.GiveAmmo(AmmoCells, 40)
		}
		// Auto-switch to newly acquired weapon
		ps.ReadyWeapon = w
	}
	return true
}

// HasWeapon reports if the weapon is owned.
func (ps *PlayerStats) HasWeapon(w WeaponType) bool {
	if w < 0 || int(w) >= NumWeapons {
		return false
	}
	return ps.Weapons[w]
}

// HasWeaponInSlot reports if the player owns any weapon in the specified status bar slot (1-7).
func (ps *PlayerStats) HasWeaponInSlot(slot int) bool {
	for w := 0; w < NumWeapons; w++ {
		if WeaponSlot(WeaponType(w)) == slot && ps.Weapons[w] {
			return true
		}
	}
	return false
}

// SelectWeapon changes the currently ready weapon if the player owns it.
func (ps *PlayerStats) SelectWeapon(w WeaponType) bool {
	if !ps.HasWeapon(w) {
		return false
	}
	ps.ReadyWeapon = w
	return true
}

// SelectSlot selects the best weapon corresponding to status bar slot 1-7.
func (ps *PlayerStats) SelectSlot(slot int) bool {
	switch slot {
	case 1:
		if ps.Weapons[WeaponChainsaw] && ps.ReadyWeapon != WeaponChainsaw {
			ps.ReadyWeapon = WeaponChainsaw
			return true
		}
		if ps.Weapons[WeaponFist] {
			ps.ReadyWeapon = WeaponFist
			return true
		}
	case 2:
		if ps.Weapons[WeaponPistol] {
			ps.ReadyWeapon = WeaponPistol
			return true
		}
	case 3:
		if ps.Weapons[WeaponSuperShotgun] && ps.ReadyWeapon != WeaponSuperShotgun {
			ps.ReadyWeapon = WeaponSuperShotgun
			return true
		}
		if ps.Weapons[WeaponShotgun] {
			ps.ReadyWeapon = WeaponShotgun
			return true
		}
	case 4:
		if ps.Weapons[WeaponChaingun] {
			ps.ReadyWeapon = WeaponChaingun
			return true
		}
	case 5:
		if ps.Weapons[WeaponRocketLauncher] {
			ps.ReadyWeapon = WeaponRocketLauncher
			return true
		}
	case 6:
		if ps.Weapons[WeaponPlasma] {
			ps.ReadyWeapon = WeaponPlasma
			return true
		}
	case 7:
		if ps.Weapons[WeaponBFG] {
			ps.ReadyWeapon = WeaponBFG
			return true
		}
	}
	return false
}

// GiveAmmo adds ammo of the given type up to maximum capacity.
func (ps *PlayerStats) GiveAmmo(a AmmoType, amount int) bool {
	if a < 0 || int(a) >= NumAmmoTypes || amount <= 0 {
		return false
	}
	if ps.Ammo[a] >= ps.MaxAmmo[a] {
		return false
	}
	ps.Ammo[a] += amount
	if ps.Ammo[a] > ps.MaxAmmo[a] {
		ps.Ammo[a] = ps.MaxAmmo[a]
	}
	return true
}

// UseAmmo deducts ammo. Returns true if sufficient ammo was available.
func (ps *PlayerStats) UseAmmo(a AmmoType, amount int) bool {
	if a < 0 || int(a) >= NumAmmoTypes || amount <= 0 {
		return false
	}
	if ps.Ammo[a] < amount {
		return false
	}
	ps.Ammo[a] -= amount
	return true
}

// GiveBackpack doubles maximum ammo capacity and gives ammo.
func (ps *PlayerStats) GiveBackpack() {
	if !ps.HasBackpack {
		ps.HasBackpack = true
		ps.MaxAmmo[AmmoBullets] = BackpackBulletsMax
		ps.MaxAmmo[AmmoShells] = BackpackShellsMax
		ps.MaxAmmo[AmmoRockets] = BackpackRocketsMax
		ps.MaxAmmo[AmmoCells] = BackpackCellsMax
	}
	ps.GiveAmmo(AmmoBullets, 10)
	ps.GiveAmmo(AmmoShells, 4)
	ps.GiveAmmo(AmmoRockets, 1)
	ps.GiveAmmo(AmmoCells, 20)
}

// GiveKey grants the specified keycard or skull key.
func (ps *PlayerStats) GiveKey(k KeyType) {
	if k >= 0 && int(k) < NumKeys {
		ps.Keys[k] = true
	}
}

// HasKey reports if the player possesses the given key.
func (ps *PlayerStats) HasKey(k KeyType) bool {
	if k >= 0 && int(k) < NumKeys {
		return ps.Keys[k]
	}
	return false
}

// HasKeySlot returns (hasCard, hasSkull) for color slot (0: Blue, 1: Yellow, 2: Red).
func (ps *PlayerStats) HasKeySlot(slot int) (hasCard bool, hasSkull bool) {
	switch slot {
	case 0: // Blue
		return ps.Keys[KeyBlueCard], ps.Keys[KeyBlueSkull]
	case 1: // Yellow
		return ps.Keys[KeyYellowCard], ps.Keys[KeyYellowSkull]
	case 2: // Red
		return ps.Keys[KeyRedCard], ps.Keys[KeyRedSkull]
	default:
		return false, false
	}
}

// GiveAll grants all weapons, all keys, full backpack and ammo, 200% health and blue armor (IDKFA cheat).
func (ps *PlayerStats) GiveAll() {
	ps.Health = MaxHealthSuper
	ps.Armor = MaxArmorSuper
	ps.ArmorType = 2
	ps.GiveBackpack()

	for i := 0; i < NumWeapons; i++ {
		ps.Weapons[i] = true
	}
	for i := 0; i < NumAmmoTypes; i++ {
		ps.Ammo[i] = ps.MaxAmmo[i]
	}
	for i := 0; i < NumKeys; i++ {
		ps.Keys[i] = true
	}
	ps.EvilGrinTimer = 60
}

// SetGodMode toggles invulnerability (IDDQD cheat).
func (ps *PlayerStats) SetGodMode(god bool) {
	ps.GodMode = god
	if god {
		ps.Health = MaxHealthStandard
	}
}

// GetReadyWeaponAmmo returns whether the active weapon consumes ammo, and its current and max counts.
func (ps *PlayerStats) GetReadyWeaponAmmo() (hasAmmo bool, current int, max int) {
	ammoType := WeaponAmmoType(ps.ReadyWeapon)
	if ammoType < 0 {
		return false, 0, 0
	}
	return true, ps.Ammo[ammoType], ps.MaxAmmo[ammoType]
}

// Update advances face animation timers and glances. Should be called every simulation tic (e.g. 60Hz).
func (ps *PlayerStats) Update() {
	ps.TotalTics++

	if ps.DamageTimer > 0 {
		ps.DamageTimer--
	}
	if ps.EvilGrinTimer > 0 {
		ps.EvilGrinTimer--
	}
	if ps.RampageTimer > 0 {
		ps.RampageTimer--
	}

	// Update idle glance direction
	ps.GlanceTimer--
	if ps.GlanceTimer <= 0 {
		if ps.GlanceDir == 0 {
			// Look sideways for 20-35 tics
			if rand.Float64() < 0.5 {
				ps.GlanceDir = 1 // Right
			} else {
				ps.GlanceDir = 2 // Left
			}
			ps.GlanceTimer = 20 + rand.Intn(16)
		} else {
			// Return to straight for 70-130 tics
			ps.GlanceDir = 0
			ps.GlanceTimer = 70 + rand.Intn(60)
		}
	}
}

// TriggerRampage sets the firing rampage face for the specified duration.
func (ps *PlayerStats) TriggerRampage(ticks int) {
	ps.RampageTimer = ticks
	if ps.RampageSide == 0 {
		ps.RampageSide = 1
	} else {
		ps.RampageSide = 0
	}
}

// HealthTier returns the Doom status bar health tier (0 to 4).
func (ps *PlayerStats) HealthTier() int {
	switch {
	case ps.Health >= 80:
		return 0
	case ps.Health >= 60:
		return 1
	case ps.Health >= 40:
		return 2
	case ps.Health >= 20:
		return 3
	default:
		return 4
	}
}

// GetFaceFrame computes the appropriate Doom status bar mugshot patch lump name.
func (ps *PlayerStats) GetFaceFrame() string {
	if ps.Health <= 0 {
		return "STFDEAD0"
	}
	if ps.GodMode {
		return "STFGOD0"
	}

	tier := ps.HealthTier()

	if ps.DamageTimer > 0 {
		return fmt.Sprintf("STFOUCH%d", tier)
	}

	if ps.EvilGrinTimer > 0 {
		return fmt.Sprintf("STFEVL%d", tier)
	}

	if ps.RampageTimer > 0 {
		if ps.RampageSide == 0 {
			return fmt.Sprintf("STFTL%d0", tier)
		}
		return fmt.Sprintf("STFTR%d0", tier)
	}

	// Normal glance frame (0: straight, 1: right, 2: left)
	return fmt.Sprintf("STFST%d%d", tier, ps.GlanceDir)
}
