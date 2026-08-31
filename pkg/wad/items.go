package wad

// ItemCategory defines the category of an item.
type ItemCategory int

const (
	ItemCategoryOther ItemCategory = iota
	ItemCategoryKey
	ItemCategoryHealth
	ItemCategoryArmor
	ItemCategoryAmmo
	ItemCategoryWeapon
	ItemCategoryPowerup
)

// String returns a readable name for the item category.
func (c ItemCategory) String() string {
	switch c {
	case ItemCategoryKey:
		return "Key"
	case ItemCategoryHealth:
		return "Health"
	case ItemCategoryArmor:
		return "Armor"
	case ItemCategoryAmmo:
		return "Ammo"
	case ItemCategoryWeapon:
		return "Weapon"
	case ItemCategoryPowerup:
		return "Powerup"
	default:
		return "Other"
	}
}

// Thing type constants for items.
const (
	// Keys
	ThingKeyBlueCard    int16 = 5
	ThingKeyYellowCard  int16 = 6
	ThingKeyRedCard     int16 = 13
	ThingKeyRedSkull    int16 = 38
	ThingKeyYellowSkull int16 = 39
	ThingKeyBlueSkull   int16 = 40

	// Health
	ThingStimpack    int16 = 2011
	ThingMedikit     int16 = 2012
	ThingHealthBonus int16 = 2014
	ThingSoulsphere  int16 = 2024
	ThingMegasphere  int16 = 2045
	ThingBerserk     int16 = 2023

	// Armor
	ThingArmorBonus int16 = 2015
	ThingGreenArmor int16 = 2018
	ThingBlueArmor  int16 = 2019

	// Ammo
	ThingAmmoClip   int16 = 2007
	ThingBoxBullets int16 = 2048
	ThingShells     int16 = 2008
	ThingBoxShells  int16 = 2049
	ThingRocket     int16 = 2010
	ThingBoxRockets int16 = 2046
	ThingCell       int16 = 2047
	ThingCellPack   int16 = 2044
	ThingBackpack   int16 = 8

	// Weapons
	ThingWeaponShotgun        int16 = 2001
	ThingWeaponChaingun       int16 = 2002
	ThingWeaponRocketLauncher int16 = 2003
	ThingWeaponPlasma         int16 = 2004
	ThingWeaponChainsaw       int16 = 2005
	ThingWeaponBFG            int16 = 2006
	ThingWeaponSuperShotgun   int16 = 82

	// Powerups & Artifacts
	ThingInvulnerability int16 = 2022
	ThingRadiationSuit    int16 = 2025
	ThingComputerMap      int16 = 2026
	ThingLiteAmp          int16 = 2027
	ThingInvisibility     int16 = 2028
)

// ItemDef describes static attributes and metadata of an item Thing.
type ItemDef struct {
	Type      int16
	Name      string
	Category  ItemCategory
	Sprite    string
	Radius    float64
	Height    float64
	PickupMsg string
}

// Default item collision radius and height.
const (
	DefaultItemRadius = 20.0
	DefaultItemHeight = 16.0
)

var itemDefs = map[int16]ItemDef{
	// Keys
	ThingKeyBlueCard: {
		Type:      ThingKeyBlueCard,
		Name:      "Blue Keycard",
		Category:  ItemCategoryKey,
		Sprite:    "BKEYA0",
		Radius:    DefaultItemRadius,
		Height:    DefaultItemHeight,
		PickupMsg: "Picked up a blue keycard.",
	},
	ThingKeyYellowCard: {
		Type:      ThingKeyYellowCard,
		Name:      "Yellow Keycard",
		Category:  ItemCategoryKey,
		Sprite:    "YKEYA0",
		Radius:    DefaultItemRadius,
		Height:    DefaultItemHeight,
		PickupMsg: "Picked up a yellow keycard.",
	},
	ThingKeyRedCard: {
		Type:      ThingKeyRedCard,
		Name:      "Red Keycard",
		Category:  ItemCategoryKey,
		Sprite:    "RKEYA0",
		Radius:    DefaultItemRadius,
		Height:    DefaultItemHeight,
		PickupMsg: "Picked up a red keycard.",
	},
	ThingKeyBlueSkull: {
		Type:      ThingKeyBlueSkull,
		Name:      "Blue Skull Key",
		Category:  ItemCategoryKey,
		Sprite:    "BSKUA0",
		Radius:    DefaultItemRadius,
		Height:    DefaultItemHeight,
		PickupMsg: "Picked up a blue skull key.",
	},
	ThingKeyYellowSkull: {
		Type:      ThingKeyYellowSkull,
		Name:      "Yellow Skull Key",
		Category:  ItemCategoryKey,
		Sprite:    "YSKUA0",
		Radius:    DefaultItemRadius,
		Height:    DefaultItemHeight,
		PickupMsg: "Picked up a yellow skull key.",
	},
	ThingKeyRedSkull: {
		Type:      ThingKeyRedSkull,
		Name:      "Red Skull Key",
		Category:  ItemCategoryKey,
		Sprite:    "RSKUA0",
		Radius:    DefaultItemRadius,
		Height:    DefaultItemHeight,
		PickupMsg: "Picked up a red skull key.",
	},

	// Health
	ThingStimpack: {
		Type:      ThingStimpack,
		Name:      "Stimpack",
		Category:  ItemCategoryHealth,
		Sprite:    "STIMA0",
		Radius:    DefaultItemRadius,
		Height:    DefaultItemHeight,
		PickupMsg: "Picked up a stimpack.",
	},
	ThingMedikit: {
		Type:      ThingMedikit,
		Name:      "Medikit",
		Category:  ItemCategoryHealth,
		Sprite:    "MEDIA0",
		Radius:    DefaultItemRadius,
		Height:    DefaultItemHeight,
		PickupMsg: "Picked up a medikit.",
	},
	ThingHealthBonus: {
		Type:      ThingHealthBonus,
		Name:      "Health Bonus",
		Category:  ItemCategoryHealth,
		Sprite:    "BON1A0",
		Radius:    DefaultItemRadius,
		Height:    DefaultItemHeight,
		PickupMsg: "Picked up a health bonus.",
	},
	ThingSoulsphere: {
		Type:      ThingSoulsphere,
		Name:      "Supercharge",
		Category:  ItemCategoryHealth,
		Sprite:    "SOURA0",
		Radius:    DefaultItemRadius,
		Height:    DefaultItemHeight,
		PickupMsg: "Supercharge!",
	},
	ThingMegasphere: {
		Type:      ThingMegasphere,
		Name:      "MegaSphere",
		Category:  ItemCategoryHealth,
		Sprite:    "MEGAA0",
		Radius:    DefaultItemRadius,
		Height:    DefaultItemHeight,
		PickupMsg: "MegaSphere!",
	},
	ThingBerserk: {
		Type:      ThingBerserk,
		Name:      "Berserk",
		Category:  ItemCategoryHealth,
		Sprite:    "PSTRA0",
		Radius:    DefaultItemRadius,
		Height:    DefaultItemHeight,
		PickupMsg: "Berserk!",
	},

	// Armor
	ThingArmorBonus: {
		Type:      ThingArmorBonus,
		Name:      "Armor Bonus",
		Category:  ItemCategoryArmor,
		Sprite:    "BON2A0",
		Radius:    DefaultItemRadius,
		Height:    DefaultItemHeight,
		PickupMsg: "Picked up an armor bonus.",
	},
	ThingGreenArmor: {
		Type:      ThingGreenArmor,
		Name:      "Green Armor",
		Category:  ItemCategoryArmor,
		Sprite:    "ARM1A0",
		Radius:    DefaultItemRadius,
		Height:    DefaultItemHeight,
		PickupMsg: "Picked up the armor.",
	},
	ThingBlueArmor: {
		Type:      ThingBlueArmor,
		Name:      "MegaArmor",
		Category:  ItemCategoryArmor,
		Sprite:    "ARM2A0",
		Radius:    DefaultItemRadius,
		Height:    DefaultItemHeight,
		PickupMsg: "Picked up the MegaArmor!",
	},

	// Ammo
	ThingAmmoClip: {
		Type:      ThingAmmoClip,
		Name:      "Ammo Clip",
		Category:  ItemCategoryAmmo,
		Sprite:    "CLIPA0",
		Radius:    DefaultItemRadius,
		Height:    DefaultItemHeight,
		PickupMsg: "Picked up an ammo clip.",
	},
	ThingBoxBullets: {
		Type:      ThingBoxBullets,
		Name:      "Box of Bullets",
		Category:  ItemCategoryAmmo,
		Sprite:    "AMMOA0",
		Radius:    DefaultItemRadius,
		Height:    DefaultItemHeight,
		PickupMsg: "Picked up a box of bullets.",
	},
	ThingShells: {
		Type:      ThingShells,
		Name:      "4 Shotgun Shells",
		Category:  ItemCategoryAmmo,
		Sprite:    "SHELA0",
		Radius:    DefaultItemRadius,
		Height:    DefaultItemHeight,
		PickupMsg: "Picked up 4 shotgun shells.",
	},
	ThingBoxShells: {
		Type:      ThingBoxShells,
		Name:      "Box of Shells",
		Category:  ItemCategoryAmmo,
		Sprite:    "SBOXA0",
		Radius:    DefaultItemRadius,
		Height:    DefaultItemHeight,
		PickupMsg: "Picked up a box of shotgun shells.",
	},
	ThingRocket: {
		Type:      ThingRocket,
		Name:      "Rocket",
		Category:  ItemCategoryAmmo,
		Sprite:    "ROCKA0",
		Radius:    DefaultItemRadius,
		Height:    DefaultItemHeight,
		PickupMsg: "Picked up a rocket.",
	},
	ThingBoxRockets: {
		Type:      ThingBoxRockets,
		Name:      "Box of Rockets",
		Category:  ItemCategoryAmmo,
		Sprite:    "BROKA0",
		Radius:    DefaultItemRadius,
		Height:    DefaultItemHeight,
		PickupMsg: "Picked up a box of rockets.",
	},
	ThingCell: {
		Type:      ThingCell,
		Name:      "Energy Cell",
		Category:  ItemCategoryAmmo,
		Sprite:    "CELLA0",
		Radius:    DefaultItemRadius,
		Height:    DefaultItemHeight,
		PickupMsg: "Picked up an energy cell.",
	},
	ThingCellPack: {
		Type:      ThingCellPack,
		Name:      "Energy Cell Pack",
		Category:  ItemCategoryAmmo,
		Sprite:    "CELPA0",
		Radius:    DefaultItemRadius,
		Height:    DefaultItemHeight,
		PickupMsg: "Picked up an energy pack.",
	},
	ThingBackpack: {
		Type:      ThingBackpack,
		Name:      "Backpack",
		Category:  ItemCategoryAmmo,
		Sprite:    "BPAKA0",
		Radius:    DefaultItemRadius,
		Height:    DefaultItemHeight,
		PickupMsg: "Picked up a backpack full of ammo!",
	},

	// Weapons
	ThingWeaponShotgun: {
		Type:      ThingWeaponShotgun,
		Name:      "Shotgun",
		Category:  ItemCategoryWeapon,
		Sprite:    "SHOTA0",
		Radius:    DefaultItemRadius,
		Height:    DefaultItemHeight,
		PickupMsg: "You got the shotgun!",
	},
	ThingWeaponSuperShotgun: {
		Type:      ThingWeaponSuperShotgun,
		Name:      "Super Shotgun",
		Category:  ItemCategoryWeapon,
		Sprite:    "SGN2A0",
		Radius:    DefaultItemRadius,
		Height:    DefaultItemHeight,
		PickupMsg: "You got the super shotgun!",
	},
	ThingWeaponChaingun: {
		Type:      ThingWeaponChaingun,
		Name:      "Chaingun",
		Category:  ItemCategoryWeapon,
		Sprite:    "MGUNA0",
		Radius:    DefaultItemRadius,
		Height:    DefaultItemHeight,
		PickupMsg: "You got the chaingun!",
	},
	ThingWeaponRocketLauncher: {
		Type:      ThingWeaponRocketLauncher,
		Name:      "Rocket Launcher",
		Category:  ItemCategoryWeapon,
		Sprite:    "LAUNA0",
		Radius:    DefaultItemRadius,
		Height:    DefaultItemHeight,
		PickupMsg: "You got the rocket launcher!",
	},
	ThingWeaponPlasma: {
		Type:      ThingWeaponPlasma,
		Name:      "Plasma Gun",
		Category:  ItemCategoryWeapon,
		Sprite:    "PLASA0",
		Radius:    DefaultItemRadius,
		Height:    DefaultItemHeight,
		PickupMsg: "You got the plasma gun!",
	},
	ThingWeaponBFG: {
		Type:      ThingWeaponBFG,
		Name:      "BFG 9000",
		Category:  ItemCategoryWeapon,
		Sprite:    "BFUGA0",
		Radius:    DefaultItemRadius,
		Height:    DefaultItemHeight,
		PickupMsg: "You got the BFG9000! Oh, yes.",
	},
	ThingWeaponChainsaw: {
		Type:      ThingWeaponChainsaw,
		Name:      "Chainsaw",
		Category:  ItemCategoryWeapon,
		Sprite:    "CSAWA0",
		Radius:    DefaultItemRadius,
		Height:    DefaultItemHeight,
		PickupMsg: "A chainsaw! Find some meat!",
	},

	// Powerups & Artifacts
	ThingInvulnerability: {
		Type:      ThingInvulnerability,
		Name:      "Invulnerability",
		Category:  ItemCategoryPowerup,
		Sprite:    "PINVA0",
		Radius:    DefaultItemRadius,
		Height:    DefaultItemHeight,
		PickupMsg: "Invulnerability!",
	},
	ThingRadiationSuit: {
		Type:      ThingRadiationSuit,
		Name:      "Radiation Shielding Suit",
		Category:  ItemCategoryPowerup,
		Sprite:    "SUITA0",
		Radius:    DefaultItemRadius,
		Height:    DefaultItemHeight,
		PickupMsg: "Radiation Shielding Suit",
	},
	ThingComputerMap: {
		Type:      ThingComputerMap,
		Name:      "Computer Area Map",
		Category:  ItemCategoryPowerup,
		Sprite:    "PMAPA0",
		Radius:    DefaultItemRadius,
		Height:    DefaultItemHeight,
		PickupMsg: "Computer Area Map",
	},
	ThingLiteAmp: {
		Type:      ThingLiteAmp,
		Name:      "Light Amplification Visor",
		Category:  ItemCategoryPowerup,
		Sprite:    "PVISA0",
		Radius:    DefaultItemRadius,
		Height:    DefaultItemHeight,
		PickupMsg: "Light Amplification Visor",
	},
	ThingInvisibility: {
		Type:      ThingInvisibility,
		Name:      "Invisibility",
		Category:  ItemCategoryPowerup,
		Sprite:    "PINSA0",
		Radius:    DefaultItemRadius,
		Height:    DefaultItemHeight,
		PickupMsg: "Partial Invisibility",
	},
}

// LookupItemDef returns the ItemDef definition for a given Thing type if it is a recognized item.
func LookupItemDef(thingType int16) (ItemDef, bool) {
	def, ok := itemDefs[thingType]
	return def, ok
}

// IsItem reports whether the given thing type represents a collectible item.
func IsItem(thingType int16) bool {
	_, ok := itemDefs[thingType]
	return ok
}

var propSprites = map[int16]string{
	2035: "BAR1A0", // Explosive Barrel
	70:   "FCANA0", // Burning barrel
	34:   "CANDA0", // Candle
	35:   "CBRAA0", // Candelabra
	44:   "TBULA0", // Tall blue torch
	45:   "TGREA0", // Tall green torch
	46:   "TREDA0", // Tall red torch
	55:   "SMBTA0", // Short blue torch
	56:   "SMGTA0", // Short green torch
	57:   "SMRTA0", // Short red torch
	47:   "SMITA0", // Brown stub / tree
	43:   "TRE1A0", // Burnt tree
	54:   "TRE2A0", // Big tree
	48:   "ELECA0", // Tech column
	30:   "COL1A0", // Tall green pillar
	32:   "COL3A0", // Tall red pillar
	31:   "COL2A0", // Short green pillar
	33:   "COL4A0", // Short red pillar
	36:   "COL5A0", // Short green pillar with beating heart
	37:   "COL6A0", // Short red pillar with skull
	41:   "CEYEA0", // Evil eye
	42:   "FSKUA0", // Floating skull
	85:   "TLMPA0", // Tall techno floor lamp
	86:   "TLMPB0", // Short techno floor lamp
	49:   "GOR1A0", // Hanging swaying body
	50:   "GOR2A0", // Hanging body arms out
	51:   "GOR3A0", // Hanging body one leg
	52:   "GOR4A0", // Hanging leg
	53:   "GOR5A0", // Hanging torso
	59:   "GOR2A0", // Hanging swaying body 2
	60:   "GOR3A0", // Hanging body 3
	61:   "GOR4A0", // Hanging body 4
	62:   "GOR5A0", // Hanging body 5
	63:   "GOR1A0", // Hanging torso 2
	73:   "HDB1A0", // Hanging pair of legs
	74:   "HDB2A0", // Hanging pair of legs (no torso)
	75:   "HDB3A0", // Hanging leg
	76:   "HDB4A0", // Hanging pair of legs (open)
	77:   "HDB5A0", // Hanging leg (no feet)
	78:   "HDB6A0", // Hanging torso
	24:   "POL5A0", // Pool of blood and flesh
	79:   "POB1A0", // Pool of blood
	80:   "POB2A0", // Pool of blood 2
	81:   "BRS1A0", // Pool of brains
	25:   "POL1A0", // Impaled human
	26:   "POL6A0", // Skull on a pole
	27:   "POL4A0", // Heads on a stick
	28:   "POL2A0", // 5 heads on a pole
	29:   "POL3A0", // Pile of skulls and candles
	10:   "POL5A0", // Bloody mess
	12:   "POB2A0", // Bloody mess 2
	15:   "PLAYN0", // Dead player
	18:   "POSSL0", // Dead former human
	19:   "SPOSL0", // Dead former sergeant
	20:   "TROOM0", // Dead imp
	21:   "SARGN0", // Dead demon
	22:   "HEADL0", // Dead caco
	23:   "SKULA0", // Dead lost soul
}

// LookupThingSprite returns the sprite patch lump name for any known thing type (item or prop).
func LookupThingSprite(thingType int16) (string, bool) {
	if def, ok := itemDefs[thingType]; ok && def.Sprite != "" {
		return def.Sprite, true
	}
	if sprite, ok := propSprites[thingType]; ok {
		return sprite, true
	}
	return "", false
}

// ItemEntity represents a placed item instance in a map.
type ItemEntity struct {
	ID        int
	Thing     Thing
	Def       ItemDef
	X         float64
	Y         float64
	FloorZ    float64
	CeilingZ  float64
	Radius    float64
	Height    float64
	Sector    int
	Collected bool
}

// ParseMapItems extracts and instantiates all collectible item Things from a MapData.
func ParseMapItems(mapData *MapData) []*ItemEntity {
	if mapData == nil || len(mapData.Things) == 0 {
		return nil
	}

	var items []*ItemEntity
	itemCount := 0

	for _, th := range mapData.Things {
		def, isItem := LookupItemDef(th.Type)
		if !isItem {
			continue
		}

		x := float64(th.X)
		y := float64(th.Y)
		floorZ := 0.0
		ceilingZ := 128.0
		secIdx := -1

		// Query sector floor height for item placement
		if len(mapData.Subsectors) > 0 {
			ssIdx := mapData.FindSubsector(x, y)
			if ssIdx >= 0 && ssIdx < len(mapData.Subsectors) {
				ss := &mapData.Subsectors[ssIdx]
				if ss.NumSegs > 0 && int(ss.FirstSeg) < len(mapData.Segs) {
					seg := &mapData.Segs[ss.FirstSeg]
					if int(seg.Linedef) < len(mapData.Linedefs) {
						ld := &mapData.Linedefs[seg.Linedef]
						sidedefIdx := ld.RightSide
						if seg.Direction != 0 {
							sidedefIdx = ld.LeftSide
						}
						if sidedefIdx != 0xFFFF && int(sidedefIdx) < len(mapData.Sidedefs) {
							s := int(mapData.Sidedefs[sidedefIdx].Sector)
							if s >= 0 && s < len(mapData.Sectors) {
								secIdx = s
								floorZ = float64(mapData.Sectors[s].FloorHeight)
								ceilingZ = float64(mapData.Sectors[s].CeilingHeight)
							}
						}
					}
				}
			}
		}

		radius := def.Radius
		if radius <= 0 {
			radius = DefaultItemRadius
		}
		height := def.Height
		if height <= 0 {
			height = DefaultItemHeight
		}

		item := &ItemEntity{
			ID:        itemCount,
			Thing:     th,
			Def:       def,
			X:         x,
			Y:         y,
			FloorZ:    floorZ,
			CeilingZ:  ceilingZ,
			Radius:    radius,
			Height:    height,
			Sector:    secIdx,
			Collected: false,
		}
		items = append(items, item)
		itemCount++
	}

	return items
}
