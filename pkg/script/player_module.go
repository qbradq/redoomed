package script

import (
	"fmt"
	"strings"

	"github.com/d5/tengo/v2"

	"github.com/qbradq/redoomed/pkg/player"
)

// createPlayerModule creates the Tengo "player" module providing script access to player statistics.
func createPlayerModule(getPlayerStats func() *player.PlayerStats) map[string]tengo.Object {
	getStats := func() (*player.PlayerStats, error) {
		if getPlayerStats == nil {
			return nil, fmt.Errorf("no player stats provider configured")
		}
		ps := getPlayerStats()
		if ps == nil {
			return nil, fmt.Errorf("no player active")
		}
		return ps, nil
	}

	toInt := func(obj tengo.Object, name string) (int, error) {
		switch v := obj.(type) {
		case *tengo.Int:
			return int(v.Value), nil
		case *tengo.Float:
			return int(v.Value), nil
		default:
			return 0, fmt.Errorf("expected integer for %s, found %s", name, obj.TypeName())
		}
	}

	parseAmmoType := func(obj tengo.Object) (player.AmmoType, error) {
		switch v := obj.(type) {
		case *tengo.Int:
			if v.Value < 0 || v.Value >= player.NumAmmoTypes {
				return -1, fmt.Errorf("invalid ammo type index %d", v.Value)
			}
			return player.AmmoType(v.Value), nil
		case *tengo.String:
			s := strings.ToLower(strings.TrimSpace(v.Value))
			switch s {
			case "bullet", "bullets", "bull":
				return player.AmmoBullets, nil
			case "shell", "shells", "shel":
				return player.AmmoShells, nil
			case "rocket", "rockets", "rckt":
				return player.AmmoRockets, nil
			case "cell", "cells":
				return player.AmmoCells, nil
			default:
				return -1, fmt.Errorf("unknown ammo type %q", v.Value)
			}
		default:
			return -1, fmt.Errorf("expected int or string for ammo type, found %s", obj.TypeName())
		}
	}

	parseWeaponType := func(obj tengo.Object) (player.WeaponType, error) {
		switch v := obj.(type) {
		case *tengo.Int:
			if v.Value < 0 || v.Value >= player.NumWeapons {
				return -1, fmt.Errorf("invalid weapon index %d", v.Value)
			}
			return player.WeaponType(v.Value), nil
		case *tengo.String:
			s := strings.ToLower(strings.TrimSpace(v.Value))
			switch s {
			case "fist":
				return player.WeaponFist, nil
			case "pistol":
				return player.WeaponPistol, nil
			case "shotgun":
				return player.WeaponShotgun, nil
			case "chaingun":
				return player.WeaponChaingun, nil
			case "rocket", "rocket_launcher", "missile":
				return player.WeaponRocketLauncher, nil
			case "plasma", "plasma_rifle":
				return player.WeaponPlasma, nil
			case "bfg", "bfg9000":
				return player.WeaponBFG, nil
			case "chainsaw":
				return player.WeaponChainsaw, nil
			case "ssg", "supershotgun", "super_shotgun":
				return player.WeaponSuperShotgun, nil
			default:
				return -1, fmt.Errorf("unknown weapon %q", v.Value)
			}
		default:
			return -1, fmt.Errorf("expected int or string for weapon, found %s", obj.TypeName())
		}
	}

	parseKeyType := func(obj tengo.Object) (player.KeyType, error) {
		switch v := obj.(type) {
		case *tengo.Int:
			if v.Value < 0 || v.Value >= player.NumKeys {
				return -1, fmt.Errorf("invalid key index %d", v.Value)
			}
			return player.KeyType(v.Value), nil
		case *tengo.String:
			s := strings.ToLower(strings.TrimSpace(v.Value))
			switch s {
			case "blue_card", "bluecard", "blue":
				return player.KeyBlueCard, nil
			case "yellow_card", "yellowcard", "yellow":
				return player.KeyYellowCard, nil
			case "red_card", "redcard", "red":
				return player.KeyRedCard, nil
			case "blue_skull", "blueskull":
				return player.KeyBlueSkull, nil
			case "yellow_skull", "yellowskull":
				return player.KeyYellowSkull, nil
			case "red_skull", "redskull":
				return player.KeyRedSkull, nil
			default:
				return -1, fmt.Errorf("unknown key %q", v.Value)
			}
		default:
			return -1, fmt.Errorf("expected int or string for key, found %s", obj.TypeName())
		}
	}

	getHealthFunc := &tengo.UserFunction{
		Name: "get_health",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			ps, err := getStats()
			if err != nil {
				return nil, err
			}
			return &tengo.Int{Value: int64(ps.Health)}, nil
		},
	}

	getArmorFunc := &tengo.UserFunction{
		Name: "get_armor",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			ps, err := getStats()
			if err != nil {
				return nil, err
			}
			return &tengo.Int{Value: int64(ps.Armor)}, nil
		},
	}

	getArmorTypeFunc := &tengo.UserFunction{
		Name: "get_armor_type",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			ps, err := getStats()
			if err != nil {
				return nil, err
			}
			return &tengo.Int{Value: int64(ps.ArmorType)}, nil
		},
	}

	giveHealthFunc := &tengo.UserFunction{
		Name: "give_health",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) == 0 {
				return nil, fmt.Errorf("give_health: expected amount argument")
			}
			amount, err := toInt(args[0], "amount")
			if err != nil {
				return nil, err
			}
			maxHealth := player.MaxHealthStandard
			if len(args) > 1 {
				m, err := toInt(args[1], "max_health")
				if err != nil {
					return nil, err
				}
				maxHealth = m
			}
			ps, err := getStats()
			if err != nil {
				return nil, err
			}
			res := ps.GiveHealth(amount, maxHealth)
			if res {
				return tengo.TrueValue, nil
			}
			return tengo.FalseValue, nil
		},
	}

	damageFunc := &tengo.UserFunction{
		Name: "damage",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) == 0 {
				return nil, fmt.Errorf("damage: expected amount argument")
			}
			amount, err := toInt(args[0], "amount")
			if err != nil {
				return nil, err
			}
			ps, err := getStats()
			if err != nil {
				return nil, err
			}
			ps.Damage(amount)
			return tengo.UndefinedValue, nil
		},
	}

	giveArmorFunc := &tengo.UserFunction{
		Name: "give_armor",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) < 2 {
				return nil, fmt.Errorf("give_armor: expected amount and armor_type arguments")
			}
			amount, err := toInt(args[0], "amount")
			if err != nil {
				return nil, err
			}
			armorType, err := toInt(args[1], "armor_type")
			if err != nil {
				return nil, err
			}
			ps, err := getStats()
			if err != nil {
				return nil, err
			}
			res := ps.GiveArmor(amount, armorType)
			if res {
				return tengo.TrueValue, nil
			}
			return tengo.FalseValue, nil
		},
	}

	giveWeaponFunc := &tengo.UserFunction{
		Name: "give_weapon",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) == 0 {
				return nil, fmt.Errorf("give_weapon: expected weapon argument")
			}
			w, err := parseWeaponType(args[0])
			if err != nil {
				return nil, err
			}
			ps, err := getStats()
			if err != nil {
				return nil, err
			}
			res := ps.GiveWeapon(w)
			if res {
				return tengo.TrueValue, nil
			}
			return tengo.FalseValue, nil
		},
	}

	hasWeaponFunc := &tengo.UserFunction{
		Name: "has_weapon",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) == 0 {
				return nil, fmt.Errorf("has_weapon: expected weapon argument")
			}
			w, err := parseWeaponType(args[0])
			if err != nil {
				return nil, err
			}
			ps, err := getStats()
			if err != nil {
				return nil, err
			}
			if ps.HasWeapon(w) {
				return tengo.TrueValue, nil
			}
			return tengo.FalseValue, nil
		},
	}

	selectWeaponFunc := &tengo.UserFunction{
		Name: "select_weapon",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) == 0 {
				return nil, fmt.Errorf("select_weapon: expected weapon or slot argument")
			}
			ps, err := getStats()
			if err != nil {
				return nil, err
			}
			if iVal, ok := args[0].(*tengo.Int); ok && iVal.Value >= 1 && iVal.Value <= 7 {
				res := ps.SelectSlot(int(iVal.Value))
				if res {
					return tengo.TrueValue, nil
				}
				return tengo.FalseValue, nil
			}
			w, err := parseWeaponType(args[0])
			if err != nil {
				return nil, err
			}
			res := ps.SelectWeapon(w)
			if res {
				return tengo.TrueValue, nil
			}
			return tengo.FalseValue, nil
		},
	}

	getReadyWeaponFunc := &tengo.UserFunction{
		Name: "get_ready_weapon",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			ps, err := getStats()
			if err != nil {
				return nil, err
			}
			return &tengo.Int{Value: int64(ps.ReadyWeapon)}, nil
		},
	}

	getAmmoFunc := &tengo.UserFunction{
		Name: "get_ammo",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) == 0 {
				return nil, fmt.Errorf("get_ammo: expected ammo type argument")
			}
			a, err := parseAmmoType(args[0])
			if err != nil {
				return nil, err
			}
			ps, err := getStats()
			if err != nil {
				return nil, err
			}
			return &tengo.Int{Value: int64(ps.Ammo[a])}, nil
		},
	}

	getMaxAmmoFunc := &tengo.UserFunction{
		Name: "get_max_ammo",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) == 0 {
				return nil, fmt.Errorf("get_max_ammo: expected ammo type argument")
			}
			a, err := parseAmmoType(args[0])
			if err != nil {
				return nil, err
			}
			ps, err := getStats()
			if err != nil {
				return nil, err
			}
			return &tengo.Int{Value: int64(ps.MaxAmmo[a])}, nil
		},
	}

	giveAmmoFunc := &tengo.UserFunction{
		Name: "give_ammo",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) < 2 {
				return nil, fmt.Errorf("give_ammo: expected ammo type and amount arguments")
			}
			a, err := parseAmmoType(args[0])
			if err != nil {
				return nil, err
			}
			amount, err := toInt(args[1], "amount")
			if err != nil {
				return nil, err
			}
			ps, err := getStats()
			if err != nil {
				return nil, err
			}
			res := ps.GiveAmmo(a, amount)
			if res {
				return tengo.TrueValue, nil
			}
			return tengo.FalseValue, nil
		},
	}

	giveBackpackFunc := &tengo.UserFunction{
		Name: "give_backpack",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			ps, err := getStats()
			if err != nil {
				return nil, err
			}
			ps.GiveBackpack()
			return tengo.UndefinedValue, nil
		},
	}

	giveKeyFunc := &tengo.UserFunction{
		Name: "give_key",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) == 0 {
				return nil, fmt.Errorf("give_key: expected key argument")
			}
			k, err := parseKeyType(args[0])
			if err != nil {
				return nil, err
			}
			ps, err := getStats()
			if err != nil {
				return nil, err
			}
			ps.GiveKey(k)
			return tengo.UndefinedValue, nil
		},
	}

	hasKeyFunc := &tengo.UserFunction{
		Name: "has_key",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) == 0 {
				return nil, fmt.Errorf("has_key: expected key argument")
			}
			ps, err := getStats()
			if err != nil {
				return nil, err
			}
			if str, ok := args[0].(*tengo.String); ok {
				s := strings.ToLower(strings.TrimSpace(str.Value))
				switch s {
				case "blue":
					if ps.HasKey(player.KeyBlueCard) || ps.HasKey(player.KeyBlueSkull) {
						return tengo.TrueValue, nil
					}
					return tengo.FalseValue, nil
				case "yellow":
					if ps.HasKey(player.KeyYellowCard) || ps.HasKey(player.KeyYellowSkull) {
						return tengo.TrueValue, nil
					}
					return tengo.FalseValue, nil
				case "red":
					if ps.HasKey(player.KeyRedCard) || ps.HasKey(player.KeyRedSkull) {
						return tengo.TrueValue, nil
					}
					return tengo.FalseValue, nil
				}
			}
			k, err := parseKeyType(args[0])
			if err != nil {
				return nil, err
			}
			if ps.HasKey(k) {
				return tengo.TrueValue, nil
			}
			return tengo.FalseValue, nil
		},
	}

	giveAllFunc := &tengo.UserFunction{
		Name: "give_all",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			ps, err := getStats()
			if err != nil {
				return nil, err
			}
			ps.GiveAll()
			return tengo.UndefinedValue, nil
		},
	}

	setGodModeFunc := &tengo.UserFunction{
		Name: "god_mode",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			ps, err := getStats()
			if err != nil {
				return nil, err
			}
			god := true
			if len(args) > 0 {
				if bVal, ok := args[0].(*tengo.Bool); ok {
					god = !bVal.IsFalsy()
				}
			}
			ps.SetGodMode(god)
			return tengo.UndefinedValue, nil
		},
	}

	isGodModeFunc := &tengo.UserFunction{
		Name: "is_god_mode",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			ps, err := getStats()
			if err != nil {
				return nil, err
			}
			if ps.GodMode {
				return tengo.TrueValue, nil
			}
			return tengo.FalseValue, nil
		},
	}

	resetFunc := &tengo.UserFunction{
		Name: "reset",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			ps, err := getStats()
			if err != nil {
				return nil, err
			}
			ps.Reset()
			return tengo.UndefinedValue, nil
		},
	}

	return map[string]tengo.Object{
		"get_health":       getHealthFunc,
		"GetHealth":        getHealthFunc,
		"get_armor":        getArmorFunc,
		"GetArmor":         getArmorFunc,
		"get_armor_type":   getArmorTypeFunc,
		"GetArmorType":     getArmorTypeFunc,
		"give_health":      giveHealthFunc,
		"GiveHealth":       giveHealthFunc,
		"damage":           damageFunc,
		"Damage":           damageFunc,
		"give_armor":       giveArmorFunc,
		"GiveArmor":        giveArmorFunc,
		"give_weapon":      giveWeaponFunc,
		"GiveWeapon":       giveWeaponFunc,
		"has_weapon":       hasWeaponFunc,
		"HasWeapon":        hasWeaponFunc,
		"select_weapon":    selectWeaponFunc,
		"SelectWeapon":     selectWeaponFunc,
		"get_ready_weapon": getReadyWeaponFunc,
		"GetReadyWeapon":   getReadyWeaponFunc,
		"get_ammo":         getAmmoFunc,
		"GetAmmo":          getAmmoFunc,
		"get_max_ammo":     getMaxAmmoFunc,
		"GetMaxAmmo":       getMaxAmmoFunc,
		"give_ammo":        giveAmmoFunc,
		"GiveAmmo":         giveAmmoFunc,
		"give_backpack":    giveBackpackFunc,
		"GiveBackpack":     giveBackpackFunc,
		"give_key":         giveKeyFunc,
		"GiveKey":          giveKeyFunc,
		"has_key":          hasKeyFunc,
		"HasKey":           hasKeyFunc,
		"give_all":         giveAllFunc,
		"GiveAll":          giveAllFunc,
		"god_mode":         setGodModeFunc,
		"GodMode":          setGodModeFunc,
		"set_god_mode":     setGodModeFunc,
		"SetGodMode":       setGodModeFunc,
		"is_god_mode":      isGodModeFunc,
		"IsGodMode":        isGodModeFunc,
		"reset":            resetFunc,
		"Reset":            resetFunc,
	}
}
