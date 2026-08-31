package script

import (
	"fmt"
	"math"

	"github.com/d5/tengo/v2"

	"github.com/qbradq/redoomed/pkg/wad"
)

// createMapModule creates the Tengo "map" module providing ID-based access to the loaded Doom map.
func createMapModule(getMapData func() *wad.MapData) map[string]tengo.Object {
	getMap := func() (*wad.MapData, error) {
		if getMapData == nil {
			return nil, fmt.Errorf("no map data provider configured")
		}
		m := getMapData()
		if m == nil {
			return nil, fmt.Errorf("no map currently loaded")
		}
		return m, nil
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

	toBool := func(b bool) tengo.Object {
		if b {
			return tengo.TrueValue
		}
		return tengo.FalseValue
	}

	getNameFunc := &tengo.UserFunction{
		Name: "get_name",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			m, err := getMap()
			if err != nil {
				return &tengo.String{Value: ""}, nil
			}
			return &tengo.String{Value: m.Name}, nil
		},
	}

	numLinesFunc := &tengo.UserFunction{
		Name: "num_lines",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			m, err := getMap()
			if err != nil {
				return &tengo.Int{Value: 0}, nil
			}
			return &tengo.Int{Value: int64(len(m.Linedefs))}, nil
		},
	}

	numSectorsFunc := &tengo.UserFunction{
		Name: "num_sectors",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			m, err := getMap()
			if err != nil {
				return &tengo.Int{Value: 0}, nil
			}
			return &tengo.Int{Value: int64(len(m.Sectors))}, nil
		},
	}

	numThingsFunc := &tengo.UserFunction{
		Name: "num_things",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			m, err := getMap()
			if err != nil {
				return &tengo.Int{Value: 0}, nil
			}
			return &tengo.Int{Value: int64(len(m.Things))}, nil
		},
	}

	numVertexesFunc := &tengo.UserFunction{
		Name: "num_vertexes",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			m, err := getMap()
			if err != nil {
				return &tengo.Int{Value: 0}, nil
			}
			return &tengo.Int{Value: int64(len(m.Vertexes))}, nil
		},
	}

	sectorAtFunc := &tengo.UserFunction{
		Name: "sector_at",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			m, err := getMap()
			if err != nil {
				return tengo.UndefinedValue, nil
			}
			if len(args) < 2 {
				return nil, fmt.Errorf("sector_at: missing x, y coordinates")
			}
			var x, y float64
			if fx, ok := args[0].(*tengo.Float); ok {
				x = fx.Value
			} else if ix, ok := args[0].(*tengo.Int); ok {
				x = float64(ix.Value)
			}
			if fy, ok := args[1].(*tengo.Float); ok {
				y = fy.Value
			} else if iy, ok := args[1].(*tengo.Int); ok {
				y = float64(iy.Value)
			}
			if sec, ok := m.SectorAt(x, y); ok && sec != nil {
				for i := range m.Sectors {
					if &m.Sectors[i] == sec {
						return &tengo.Int{Value: int64(i)}, nil
					}
				}
			}
			return &tengo.Int{Value: -1}, nil
		},
	}

	getLineFunc := &tengo.UserFunction{
		Name: "get_line",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			m, err := getMap()
			if err != nil {
				return tengo.UndefinedValue, nil
			}
			if len(args) == 0 {
				return nil, fmt.Errorf("get_line: missing line_id")
			}
			lineID, err := toInt(args[0], "line_id")
			if err != nil {
				return nil, err
			}
			if lineID < 0 || lineID >= len(m.Linedefs) {
				return tengo.UndefinedValue, nil
			}

			ld := &m.Linedefs[lineID]
			frontSide := int64(-1)
			frontSec := int64(-1)
			if ld.RightSide != 0xFFFF && int(ld.RightSide) < len(m.Sidedefs) {
				frontSide = int64(ld.RightSide)
				frontSec = int64(m.Sidedefs[ld.RightSide].Sector)
			}

			backSide := int64(-1)
			backSec := int64(-1)
			if ld.LeftSide != 0xFFFF && int(ld.LeftSide) < len(m.Sidedefs) {
				backSide = int64(ld.LeftSide)
				backSec = int64(m.Sidedefs[ld.LeftSide].Sector)
			}

			res := &tengo.Map{
				Value: map[string]tengo.Object{
					"id":             &tengo.Int{Value: int64(lineID)},
					"v1":             &tengo.Int{Value: int64(ld.V1)},
					"v2":             &tengo.Int{Value: int64(ld.V2)},
					"flags":          &tengo.Int{Value: int64(ld.Flags)},
					"special":        &tengo.Int{Value: int64(ld.Special)},
					"tag":            &tengo.Int{Value: int64(ld.Tag)},
					"front_sidedef":  &tengo.Int{Value: frontSide},
					"back_sidedef":   &tengo.Int{Value: backSide},
					"right_sidedef":  &tengo.Int{Value: frontSide},
					"left_sidedef":   &tengo.Int{Value: backSide},
					"front_sector":   &tengo.Int{Value: frontSec},
					"back_sector":    &tengo.Int{Value: backSec},
				},
			}
			return res, nil
		},
	}

	setLineSpecialFunc := &tengo.UserFunction{
		Name: "set_line_special",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			m, err := getMap()
			if err != nil {
				return tengo.UndefinedValue, nil
			}
			if len(args) < 2 {
				return nil, fmt.Errorf("set_line_special: expected line_id and special")
			}
			lineID, err := toInt(args[0], "line_id")
			if err != nil {
				return nil, err
			}
			special, err := toInt(args[1], "special")
			if err != nil {
				return nil, err
			}
			if lineID >= 0 && lineID < len(m.Linedefs) {
				m.Linedefs[lineID].Special = int16(special)
			}
			return tengo.UndefinedValue, nil
		},
	}

	setLineFlagsFunc := &tengo.UserFunction{
		Name: "set_line_flags",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			m, err := getMap()
			if err != nil {
				return tengo.UndefinedValue, nil
			}
			if len(args) < 2 {
				return nil, fmt.Errorf("set_line_flags: expected line_id and flags")
			}
			lineID, err := toInt(args[0], "line_id")
			if err != nil {
				return nil, err
			}
			flags, err := toInt(args[1], "flags")
			if err != nil {
				return nil, err
			}
			if lineID >= 0 && lineID < len(m.Linedefs) {
				m.Linedefs[lineID].Flags = uint16(flags)
			}
			return tengo.UndefinedValue, nil
		},
	}

	setLineTagFunc := &tengo.UserFunction{
		Name: "set_line_tag",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			m, err := getMap()
			if err != nil {
				return tengo.UndefinedValue, nil
			}
			if len(args) < 2 {
				return nil, fmt.Errorf("set_line_tag: expected line_id and tag")
			}
			lineID, err := toInt(args[0], "line_id")
			if err != nil {
				return nil, err
			}
			tag, err := toInt(args[1], "tag")
			if err != nil {
				return nil, err
			}
			if lineID >= 0 && lineID < len(m.Linedefs) {
				m.Linedefs[lineID].Tag = int16(tag)
			}
			return tengo.UndefinedValue, nil
		},
	}

	findLinesByTagFunc := &tengo.UserFunction{
		Name: "find_lines_by_tag",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			m, err := getMap()
			if err != nil {
				return &tengo.Array{}, nil
			}
			if len(args) == 0 {
				return nil, fmt.Errorf("find_lines_by_tag: missing tag")
			}
			tag, err := toInt(args[0], "tag")
			if err != nil {
				return nil, err
			}
			var matches []tengo.Object
			for i, ld := range m.Linedefs {
				if int(ld.Tag) == tag {
					matches = append(matches, &tengo.Int{Value: int64(i)})
				}
			}
			return &tengo.Array{Value: matches}, nil
		},
	}

	getSectorFunc := &tengo.UserFunction{
		Name: "get_sector",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			m, err := getMap()
			if err != nil {
				return tengo.UndefinedValue, nil
			}
			if len(args) == 0 {
				return nil, fmt.Errorf("get_sector: missing sector_id")
			}
			secID, err := toInt(args[0], "sector_id")
			if err != nil {
				return nil, err
			}
			if secID < 0 || secID >= len(m.Sectors) {
				return tengo.UndefinedValue, nil
			}

			sec := &m.Sectors[secID]
			res := &tengo.Map{
				Value: map[string]tengo.Object{
					"id":             &tengo.Int{Value: int64(secID)},
					"floor_height":   &tengo.Int{Value: int64(sec.FloorHeight)},
					"ceiling_height": &tengo.Int{Value: int64(sec.CeilingHeight)},
					"floor_pic":      &tengo.String{Value: sec.FloorPic},
					"ceiling_pic":    &tengo.String{Value: sec.CeilingPic},
					"light_level":    &tengo.Int{Value: int64(sec.LightLevel)},
					"special":        &tengo.Int{Value: int64(sec.Special)},
					"tag":            &tengo.Int{Value: int64(sec.Tag)},
				},
			}
			return res, nil
		},
	}

	setSectorFloorHeightFunc := &tengo.UserFunction{
		Name: "set_sector_floor_height",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			m, err := getMap()
			if err != nil {
				return tengo.UndefinedValue, nil
			}
			if len(args) < 2 {
				return nil, fmt.Errorf("set_sector_floor_height: expected sector_id and height")
			}
			secID, err := toInt(args[0], "sector_id")
			if err != nil {
				return nil, err
			}
			h, err := toInt(args[1], "height")
			if err != nil {
				return nil, err
			}
			if secID >= 0 && secID < len(m.Sectors) {
				m.Sectors[secID].FloorHeight = int16(h)
			}
			return tengo.UndefinedValue, nil
		},
	}

	setSectorCeilingHeightFunc := &tengo.UserFunction{
		Name: "set_sector_ceiling_height",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			m, err := getMap()
			if err != nil {
				return tengo.UndefinedValue, nil
			}
			if len(args) < 2 {
				return nil, fmt.Errorf("set_sector_ceiling_height: expected sector_id and height")
			}
			secID, err := toInt(args[0], "sector_id")
			if err != nil {
				return nil, err
			}
			h, err := toInt(args[1], "height")
			if err != nil {
				return nil, err
			}
			if secID >= 0 && secID < len(m.Sectors) {
				m.Sectors[secID].CeilingHeight = int16(h)
			}
			return tengo.UndefinedValue, nil
		},
	}

	setSectorLightLevelFunc := &tengo.UserFunction{
		Name: "set_sector_light_level",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			m, err := getMap()
			if err != nil {
				return tengo.UndefinedValue, nil
			}
			if len(args) < 2 {
				return nil, fmt.Errorf("set_sector_light_level: expected sector_id and light")
			}
			secID, err := toInt(args[0], "sector_id")
			if err != nil {
				return nil, err
			}
			light, err := toInt(args[1], "light")
			if err != nil {
				return nil, err
			}
			if secID >= 0 && secID < len(m.Sectors) {
				m.Sectors[secID].LightLevel = int16(light)
			}
			return tengo.UndefinedValue, nil
		},
	}

	setSectorSpecialFunc := &tengo.UserFunction{
		Name: "set_sector_special",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			m, err := getMap()
			if err != nil {
				return tengo.UndefinedValue, nil
			}
			if len(args) < 2 {
				return nil, fmt.Errorf("set_sector_special: expected sector_id and special")
			}
			secID, err := toInt(args[0], "sector_id")
			if err != nil {
				return nil, err
			}
			special, err := toInt(args[1], "special")
			if err != nil {
				return nil, err
			}
			if secID >= 0 && secID < len(m.Sectors) {
				m.Sectors[secID].Special = int16(special)
			}
			return tengo.UndefinedValue, nil
		},
	}

	setSectorTagFunc := &tengo.UserFunction{
		Name: "set_sector_tag",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			m, err := getMap()
			if err != nil {
				return tengo.UndefinedValue, nil
			}
			if len(args) < 2 {
				return nil, fmt.Errorf("set_sector_tag: expected sector_id and tag")
			}
			secID, err := toInt(args[0], "sector_id")
			if err != nil {
				return nil, err
			}
			tag, err := toInt(args[1], "tag")
			if err != nil {
				return nil, err
			}
			if secID >= 0 && secID < len(m.Sectors) {
				m.Sectors[secID].Tag = int16(tag)
			}
			return tengo.UndefinedValue, nil
		},
	}

	setSectorFloorPicFunc := &tengo.UserFunction{
		Name: "set_sector_floor_pic",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			m, err := getMap()
			if err != nil {
				return tengo.UndefinedValue, nil
			}
			if len(args) < 2 {
				return nil, fmt.Errorf("set_sector_floor_pic: expected sector_id and pic name")
			}
			secID, err := toInt(args[0], "sector_id")
			if err != nil {
				return nil, err
			}
			picStr, ok := args[1].(*tengo.String)
			if !ok {
				return nil, fmt.Errorf("set_sector_floor_pic: expected string for pic, found %s", args[1].TypeName())
			}
			if secID >= 0 && secID < len(m.Sectors) {
				m.Sectors[secID].FloorPic = picStr.Value
			}
			return tengo.UndefinedValue, nil
		},
	}

	setSectorCeilingPicFunc := &tengo.UserFunction{
		Name: "set_sector_ceiling_pic",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			m, err := getMap()
			if err != nil {
				return tengo.UndefinedValue, nil
			}
			if len(args) < 2 {
				return nil, fmt.Errorf("set_sector_ceiling_pic: expected sector_id and pic name")
			}
			secID, err := toInt(args[0], "sector_id")
			if err != nil {
				return nil, err
			}
			picStr, ok := args[1].(*tengo.String)
			if !ok {
				return nil, fmt.Errorf("set_sector_ceiling_pic: expected string for pic, found %s", args[1].TypeName())
			}
			if secID >= 0 && secID < len(m.Sectors) {
				m.Sectors[secID].CeilingPic = picStr.Value
			}
			return tengo.UndefinedValue, nil
		},
	}

	findSectorsByTagFunc := &tengo.UserFunction{
		Name: "find_sectors_by_tag",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			m, err := getMap()
			if err != nil {
				return &tengo.Array{}, nil
			}
			if len(args) == 0 {
				return nil, fmt.Errorf("find_sectors_by_tag: missing tag")
			}
			tag, err := toInt(args[0], "tag")
			if err != nil {
				return nil, err
			}
			var matches []tengo.Object
			for i, sec := range m.Sectors {
				if int(sec.Tag) == tag {
					matches = append(matches, &tengo.Int{Value: int64(i)})
				}
			}
			return &tengo.Array{Value: matches}, nil
		},
	}

	getAdjacentSectorsFunc := &tengo.UserFunction{
		Name: "get_adjacent_sectors",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			m, err := getMap()
			if err != nil {
				return &tengo.Array{}, nil
			}
			if len(args) == 0 {
				return nil, fmt.Errorf("get_adjacent_sectors: missing sector_id")
			}
			secID, err := toInt(args[0], "sector_id")
			if err != nil {
				return nil, err
			}
			adj := findAdjacentSectors(m, secID)
			var res []tengo.Object
			for _, s := range adj {
				res = append(res, &tengo.Int{Value: int64(s)})
			}
			return &tengo.Array{Value: res}, nil
		},
	}

	getLowestAdjacentCeilingFunc := &tengo.UserFunction{
		Name: "get_lowest_adjacent_ceiling",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			m, err := getMap()
			if err != nil {
				return &tengo.Int{Value: 0}, nil
			}
			if len(args) == 0 {
				return nil, fmt.Errorf("get_lowest_adjacent_ceiling: missing sector_id")
			}
			secID, err := toInt(args[0], "sector_id")
			if err != nil {
				return nil, err
			}
			val := getLowestAdjacentCeiling(m, secID)
			return &tengo.Int{Value: int64(val)}, nil
		},
	}

	getHighestAdjacentCeilingFunc := &tengo.UserFunction{
		Name: "get_highest_adjacent_ceiling",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			m, err := getMap()
			if err != nil {
				return &tengo.Int{Value: 0}, nil
			}
			if len(args) == 0 {
				return nil, fmt.Errorf("get_highest_adjacent_ceiling: missing sector_id")
			}
			secID, err := toInt(args[0], "sector_id")
			if err != nil {
				return nil, err
			}
			val := getHighestAdjacentCeiling(m, secID)
			return &tengo.Int{Value: int64(val)}, nil
		},
	}

	getLowestAdjacentFloorFunc := &tengo.UserFunction{
		Name: "get_lowest_adjacent_floor",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			m, err := getMap()
			if err != nil {
				return &tengo.Int{Value: 0}, nil
			}
			if len(args) == 0 {
				return nil, fmt.Errorf("get_lowest_adjacent_floor: missing sector_id")
			}
			secID, err := toInt(args[0], "sector_id")
			if err != nil {
				return nil, err
			}
			val := getLowestAdjacentFloor(m, secID)
			return &tengo.Int{Value: int64(val)}, nil
		},
	}

	getHighestAdjacentFloorFunc := &tengo.UserFunction{
		Name: "get_highest_adjacent_floor",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			m, err := getMap()
			if err != nil {
				return &tengo.Int{Value: 0}, nil
			}
			if len(args) == 0 {
				return nil, fmt.Errorf("get_highest_adjacent_floor: missing sector_id")
			}
			secID, err := toInt(args[0], "sector_id")
			if err != nil {
				return nil, err
			}
			val := getHighestAdjacentFloor(m, secID)
			return &tengo.Int{Value: int64(val)}, nil
		},
	}

	getThingFunc := &tengo.UserFunction{
		Name: "get_thing",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			m, err := getMap()
			if err != nil {
				return tengo.UndefinedValue, nil
			}
			if len(args) == 0 {
				return nil, fmt.Errorf("get_thing: missing thing_id")
			}
			thingID, err := toInt(args[0], "thing_id")
			if err != nil {
				return nil, err
			}
			if thingID < 0 || thingID >= len(m.Things) {
				return tengo.UndefinedValue, nil
			}

			th := &m.Things[thingID]
			res := &tengo.Map{
				Value: map[string]tengo.Object{
					"id":      &tengo.Int{Value: int64(thingID)},
					"x":       &tengo.Int{Value: int64(th.X)},
					"y":       &tengo.Int{Value: int64(th.Y)},
					"angle":   &tengo.Int{Value: int64(th.Angle)},
					"type":    &tengo.Int{Value: int64(th.Type)},
					"flags":   &tengo.Int{Value: int64(th.Flags)},
					"is_item": toBool(wad.IsItem(th.Type)),
				},
			}
			return res, nil
		},
	}

	numItemsFunc := &tengo.UserFunction{
		Name: "num_items",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			m, err := getMap()
			if err != nil {
				return &tengo.Int{Value: 0}, nil
			}
			items := wad.ParseMapItems(m)
			return &tengo.Int{Value: int64(len(items))}, nil
		},
	}

	isItemFunc := &tengo.UserFunction{
		Name: "is_item",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) == 0 {
				return nil, fmt.Errorf("is_item: missing thing_type")
			}
			tType, err := toInt(args[0], "thing_type")
			if err != nil {
				return nil, err
			}
			if wad.IsItem(int16(tType)) {
				return tengo.TrueValue, nil
			}
			return tengo.FalseValue, nil
		},
	}

	getItemFunc := &tengo.UserFunction{
		Name: "get_item",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			m, err := getMap()
			if err != nil {
				return tengo.UndefinedValue, nil
			}
			if len(args) == 0 {
				return nil, fmt.Errorf("get_item: missing item_id")
			}
			itemID, err := toInt(args[0], "item_id")
			if err != nil {
				return nil, err
			}
			items := wad.ParseMapItems(m)
			if itemID < 0 || itemID >= len(items) {
				return tengo.UndefinedValue, nil
			}
			it := items[itemID]
			return &tengo.Map{
				Value: map[string]tengo.Object{
					"id":        &tengo.Int{Value: int64(it.ID)},
					"x":         &tengo.Int{Value: int64(it.X)},
					"y":         &tengo.Int{Value: int64(it.Y)},
					"floor_z":   &tengo.Float{Value: it.FloorZ},
					"type":      &tengo.Int{Value: int64(it.Def.Type)},
					"name":      &tengo.String{Value: it.Def.Name},
					"category":  &tengo.String{Value: it.Def.Category.String()},
					"sprite":    &tengo.String{Value: it.Def.Sprite},
					"collected": toBool(it.Collected),
				},
			}, nil
		},
	}

	getItemsFunc := &tengo.UserFunction{
		Name: "get_items",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			m, err := getMap()
			if err != nil {
				return &tengo.Array{}, nil
			}
			items := wad.ParseMapItems(m)
			var res []tengo.Object
			for _, it := range items {
				res = append(res, &tengo.Map{
					Value: map[string]tengo.Object{
						"id":        &tengo.Int{Value: int64(it.ID)},
						"x":         &tengo.Int{Value: int64(it.X)},
						"y":         &tengo.Int{Value: int64(it.Y)},
						"floor_z":   &tengo.Float{Value: it.FloorZ},
						"type":      &tengo.Int{Value: int64(it.Def.Type)},
						"name":      &tengo.String{Value: it.Def.Name},
						"category":  &tengo.String{Value: it.Def.Category.String()},
						"sprite":    &tengo.String{Value: it.Def.Sprite},
						"collected": toBool(it.Collected),
					},
				})
			}
			return &tengo.Array{Value: res}, nil
		},
	}

	getPlayerFunc := &tengo.UserFunction{
		Name: "get_player",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			m, err := getMap()
			if err != nil {
				return tengo.UndefinedValue, nil
			}
			targetType := wad.ThingPlayer1Start
			if len(args) > 0 {
				pNum, err := toInt(args[0], "player_num")
				if err == nil {
					switch pNum {
					case 1:
						targetType = wad.ThingPlayer1Start
					case 2:
						targetType = wad.ThingPlayer2Start
					case 3:
						targetType = wad.ThingPlayer3Start
					case 4:
						targetType = wad.ThingPlayer4Start
					}
				}
			}
			for i, th := range m.Things {
				if th.Type == targetType {
					return &tengo.Map{
						Value: map[string]tengo.Object{
							"id":    &tengo.Int{Value: int64(i)},
							"x":     &tengo.Int{Value: int64(th.X)},
							"y":     &tengo.Int{Value: int64(th.Y)},
							"angle": &tengo.Int{Value: int64(th.Angle)},
							"type":  &tengo.Int{Value: int64(th.Type)},
							"flags": &tengo.Int{Value: int64(th.Flags)},
						},
					}, nil
				}
			}
			return tengo.UndefinedValue, nil
		},
	}

	getVertexFunc := &tengo.UserFunction{
		Name: "get_vertex",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			m, err := getMap()
			if err != nil {
				return tengo.UndefinedValue, nil
			}
			if len(args) == 0 {
				return nil, fmt.Errorf("get_vertex: missing vertex_id")
			}
			vID, err := toInt(args[0], "vertex_id")
			if err != nil {
				return nil, err
			}
			if vID < 0 || vID >= len(m.Vertexes) {
				return tengo.UndefinedValue, nil
			}
			v := &m.Vertexes[vID]
			return &tengo.Map{
				Value: map[string]tengo.Object{
					"id": &tengo.Int{Value: int64(vID)},
					"x":  &tengo.Int{Value: int64(v.X)},
					"y":  &tengo.Int{Value: int64(v.Y)},
				},
			}, nil
		},
	}

	numSidedefsFunc := &tengo.UserFunction{
		Name: "num_sidedefs",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			m, err := getMap()
			if err != nil {
				return &tengo.Int{Value: 0}, nil
			}
			return &tengo.Int{Value: int64(len(m.Sidedefs))}, nil
		},
	}

	getSidedefFunc := &tengo.UserFunction{
		Name: "get_sidedef",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			m, err := getMap()
			if err != nil {
				return tengo.UndefinedValue, nil
			}
			if len(args) == 0 {
				return nil, fmt.Errorf("get_sidedef: missing sidedef_id")
			}
			sideID, err := toInt(args[0], "sidedef_id")
			if err != nil {
				return nil, err
			}
			if sideID < 0 || sideID >= len(m.Sidedefs) {
				return tengo.UndefinedValue, nil
			}
			s := &m.Sidedefs[sideID]
			return &tengo.Map{
				Value: map[string]tengo.Object{
					"id":             &tengo.Int{Value: int64(sideID)},
					"x_offset":       &tengo.Int{Value: int64(s.XOffset)},
					"y_offset":       &tengo.Int{Value: int64(s.YOffset)},
					"upper_texture":  &tengo.String{Value: s.UpperTexture},
					"lower_texture":  &tengo.String{Value: s.LowerTexture},
					"middle_texture": &tengo.String{Value: s.MiddleTexture},
					"sector":         &tengo.Int{Value: int64(s.Sector)},
				},
			}, nil
		},
	}

	setSidedefXOffsetFunc := &tengo.UserFunction{
		Name: "set_sidedef_x_offset",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			m, err := getMap()
			if err != nil {
				return tengo.UndefinedValue, nil
			}
			if len(args) < 2 {
				return nil, fmt.Errorf("set_sidedef_x_offset: expected sidedef_id and offset")
			}
			sideID, err := toInt(args[0], "sidedef_id")
			if err != nil {
				return nil, err
			}
			offset, err := toInt(args[1], "offset")
			if err != nil {
				return nil, err
			}
			if sideID >= 0 && sideID < len(m.Sidedefs) {
				m.Sidedefs[sideID].XOffset = int16(offset)
			}
			return tengo.UndefinedValue, nil
		},
	}

	setSidedefYOffsetFunc := &tengo.UserFunction{
		Name: "set_sidedef_y_offset",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			m, err := getMap()
			if err != nil {
				return tengo.UndefinedValue, nil
			}
			if len(args) < 2 {
				return nil, fmt.Errorf("set_sidedef_y_offset: expected sidedef_id and offset")
			}
			sideID, err := toInt(args[0], "sidedef_id")
			if err != nil {
				return nil, err
			}
			offset, err := toInt(args[1], "offset")
			if err != nil {
				return nil, err
			}
			if sideID >= 0 && sideID < len(m.Sidedefs) {
				m.Sidedefs[sideID].YOffset = int16(offset)
			}
			return tengo.UndefinedValue, nil
		},
	}

	setSidedefUpperTextureFunc := &tengo.UserFunction{
		Name: "set_sidedef_upper_texture",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			m, err := getMap()
			if err != nil {
				return tengo.UndefinedValue, nil
			}
			if len(args) < 2 {
				return nil, fmt.Errorf("set_sidedef_upper_texture: expected sidedef_id and texture name")
			}
			sideID, err := toInt(args[0], "sidedef_id")
			if err != nil {
				return nil, err
			}
			texStr, ok := args[1].(*tengo.String)
			if !ok {
				return nil, fmt.Errorf("set_sidedef_upper_texture: expected string for texture name, found %s", args[1].TypeName())
			}
			if sideID >= 0 && sideID < len(m.Sidedefs) {
				m.Sidedefs[sideID].UpperTexture = texStr.Value
			}
			return tengo.UndefinedValue, nil
		},
	}

	setSidedefLowerTextureFunc := &tengo.UserFunction{
		Name: "set_sidedef_lower_texture",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			m, err := getMap()
			if err != nil {
				return tengo.UndefinedValue, nil
			}
			if len(args) < 2 {
				return nil, fmt.Errorf("set_sidedef_lower_texture: expected sidedef_id and texture name")
			}
			sideID, err := toInt(args[0], "sidedef_id")
			if err != nil {
				return nil, err
			}
			texStr, ok := args[1].(*tengo.String)
			if !ok {
				return nil, fmt.Errorf("set_sidedef_lower_texture: expected string for texture name, found %s", args[1].TypeName())
			}
			if sideID >= 0 && sideID < len(m.Sidedefs) {
				m.Sidedefs[sideID].LowerTexture = texStr.Value
			}
			return tengo.UndefinedValue, nil
		},
	}

	setSidedefMiddleTextureFunc := &tengo.UserFunction{
		Name: "set_sidedef_middle_texture",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			m, err := getMap()
			if err != nil {
				return tengo.UndefinedValue, nil
			}
			if len(args) < 2 {
				return nil, fmt.Errorf("set_sidedef_middle_texture: expected sidedef_id and texture name")
			}
			sideID, err := toInt(args[0], "sidedef_id")
			if err != nil {
				return nil, err
			}
			texStr, ok := args[1].(*tengo.String)
			if !ok {
				return nil, fmt.Errorf("set_sidedef_middle_texture: expected string for texture name, found %s", args[1].TypeName())
			}
			if sideID >= 0 && sideID < len(m.Sidedefs) {
				m.Sidedefs[sideID].MiddleTexture = texStr.Value
			}
			return tengo.UndefinedValue, nil
		},
	}

	return map[string]tengo.Object{

		// Info & Counts
		"name":                         getNameFunc,
		"get_name":                     getNameFunc,
		"GetName":                      getNameFunc,
		"num_lines":                    numLinesFunc,
		"NumLines":                     numLinesFunc,
		"line_count":                   numLinesFunc,
		"num_sectors":                  numSectorsFunc,
		"NumSectors":                   numSectorsFunc,
		"sector_count":                 numSectorsFunc,
		"num_things":                   numThingsFunc,
		"NumThings":                    numThingsFunc,
		"thing_count":                  numThingsFunc,
		"num_vertexes":                 numVertexesFunc,
		"NumVertexes":                  numVertexesFunc,
		"vertex_count":                 numVertexesFunc,
		"sector_at":                    sectorAtFunc,
		"SectorAt":                     sectorAtFunc,

		// Lines
		"get_line":                     getLineFunc,
		"GetLine":                      getLineFunc,
		"set_line_special":             setLineSpecialFunc,
		"SetLineSpecial":               setLineSpecialFunc,
		"set_line_flags":               setLineFlagsFunc,
		"SetLineFlags":                 setLineFlagsFunc,
		"set_line_tag":                 setLineTagFunc,
		"SetLineTag":                   setLineTagFunc,
		"find_lines_by_tag":            findLinesByTagFunc,
		"FindLinesByTag":               findLinesByTagFunc,
		"get_lines_by_tag":             findLinesByTagFunc,

		// Sectors
		"get_sector":                   getSectorFunc,
		"GetSector":                    getSectorFunc,
		"set_sector_floor_height":      setSectorFloorHeightFunc,
		"SetSectorFloorHeight":         setSectorFloorHeightFunc,
		"set_floor_height":             setSectorFloorHeightFunc,
		"set_sector_ceiling_height":    setSectorCeilingHeightFunc,
		"SetSectorCeilingHeight":       setSectorCeilingHeightFunc,
		"set_ceiling_height":           setSectorCeilingHeightFunc,
		"set_sector_light_level":       setSectorLightLevelFunc,
		"SetSectorLightLevel":          setSectorLightLevelFunc,
		"set_light_level":              setSectorLightLevelFunc,
		"set_sector_special":           setSectorSpecialFunc,
		"SetSectorSpecial":             setSectorSpecialFunc,
		"set_sector_tag":               setSectorTagFunc,
		"SetSectorTag":                 setSectorTagFunc,
		"set_sector_floor_pic":         setSectorFloorPicFunc,
		"SetSectorFloorPic":            setSectorFloorPicFunc,
		"set_sector_ceiling_pic":       setSectorCeilingPicFunc,
		"SetSectorCeilingPic":          setSectorCeilingPicFunc,
		"find_sectors_by_tag":          findSectorsByTagFunc,
		"FindSectorsByTag":             findSectorsByTagFunc,
		"get_sectors_by_tag":           findSectorsByTagFunc,

		// Adjacent Calculations
		"get_adjacent_sectors":         getAdjacentSectorsFunc,
		"GetAdjacentSectors":           getAdjacentSectorsFunc,
		"get_lowest_adjacent_ceiling":  getLowestAdjacentCeilingFunc,
		"GetLowestAdjacentCeiling":     getLowestAdjacentCeilingFunc,
		"get_highest_adjacent_ceiling": getHighestAdjacentCeilingFunc,
		"GetHighestAdjacentCeiling":    getHighestAdjacentCeilingFunc,
		"get_lowest_adjacent_floor":    getLowestAdjacentFloorFunc,
		"GetLowestAdjacentFloor":       getLowestAdjacentFloorFunc,
		"get_highest_adjacent_floor":   getHighestAdjacentFloorFunc,
		"GetHighestAdjacentFloor":      getHighestAdjacentFloorFunc,

		// Sidedefs
		"get_sidedef":                  getSidedefFunc,
		"GetSidedef":                  getSidedefFunc,
		"set_sidedef_x_offset":         setSidedefXOffsetFunc,
		"SetSidedefXOffset":            setSidedefXOffsetFunc,
		"set_sidedef_y_offset":         setSidedefYOffsetFunc,
		"SetSidedefYOffset":            setSidedefYOffsetFunc,
		"set_sidedef_upper_texture":    setSidedefUpperTextureFunc,
		"SetSidedefUpperTexture":       setSidedefUpperTextureFunc,
		"set_sidedef_lower_texture":    setSidedefLowerTextureFunc,
		"SetSidedefLowerTexture":       setSidedefLowerTextureFunc,
		"set_sidedef_middle_texture":   setSidedefMiddleTextureFunc,
		"SetSidedefMiddleTexture":      setSidedefMiddleTextureFunc,
		"num_sidedefs":                 numSidedefsFunc,
		"NumSidedefs":                  numSidedefsFunc,
		"sidedef_count":                numSidedefsFunc,

		// Things & Vertexes
		"get_thing":                    getThingFunc,
		"GetThing":                     getThingFunc,
		"get_player":                   getPlayerFunc,
		"GetPlayer":                    getPlayerFunc,
		"get_vertex":                   getVertexFunc,
		"GetVertex":                    getVertexFunc,

		// Items
		"num_items":                    numItemsFunc,
		"NumItems":                     numItemsFunc,
		"item_count":                   numItemsFunc,
		"get_item":                     getItemFunc,
		"GetItem":                      getItemFunc,
		"get_items":                    getItemsFunc,
		"GetItems":                     getItemsFunc,
		"is_item":                      isItemFunc,
		"IsItem":                       isItemFunc,
	}
}

// findAdjacentSectors returns the list of unique sector IDs sharing a 2-sided linedef with secID.
func findAdjacentSectors(mapData *wad.MapData, secID int) []int {
	if mapData == nil || secID < 0 || secID >= len(mapData.Sectors) {
		return nil
	}
	adjMap := make(map[int]bool)
	for i := range mapData.Linedefs {
		ld := &mapData.Linedefs[i]
		if ld.RightSide == 0xFFFF || ld.LeftSide == 0xFFFF {
			continue
		}
		if int(ld.RightSide) >= len(mapData.Sidedefs) || int(ld.LeftSide) >= len(mapData.Sidedefs) {
			continue
		}
		sec1 := int(mapData.Sidedefs[ld.RightSide].Sector)
		sec2 := int(mapData.Sidedefs[ld.LeftSide].Sector)
		if sec1 == secID && sec2 != secID && sec2 >= 0 && sec2 < len(mapData.Sectors) {
			adjMap[sec2] = true
		} else if sec2 == secID && sec1 != secID && sec1 >= 0 && sec1 < len(mapData.Sectors) {
			adjMap[sec1] = true
		}
	}
	res := make([]int, 0, len(adjMap))
	for s := range adjMap {
		res = append(res, s)
	}
	return res
}

// getLowestAdjacentCeiling computes the lowest ceiling height of adjacent sectors.
func getLowestAdjacentCeiling(mapData *wad.MapData, secID int) int {
	if mapData == nil || secID < 0 || secID >= len(mapData.Sectors) {
		return 0
	}
	adj := findAdjacentSectors(mapData, secID)
	if len(adj) == 0 {
		return int(mapData.Sectors[secID].CeilingHeight)
	}
	minCeil := int(mapData.Sectors[adj[0]].CeilingHeight)
	for _, s := range adj[1:] {
		c := int(mapData.Sectors[s].CeilingHeight)
		if c < minCeil {
			minCeil = c
		}
	}
	return minCeil
}

// getHighestAdjacentCeiling computes the highest ceiling height of adjacent sectors.
func getHighestAdjacentCeiling(mapData *wad.MapData, secID int) int {
	if mapData == nil || secID < 0 || secID >= len(mapData.Sectors) {
		return 0
	}
	adj := findAdjacentSectors(mapData, secID)
	if len(adj) == 0 {
		return int(mapData.Sectors[secID].CeilingHeight)
	}
	maxCeil := int(mapData.Sectors[adj[0]].CeilingHeight)
	for _, s := range adj[1:] {
		c := int(mapData.Sectors[s].CeilingHeight)
		if c > maxCeil {
			maxCeil = c
		}
	}
	return maxCeil
}

// getLowestAdjacentFloor computes the lowest floor height of adjacent sectors.
func getLowestAdjacentFloor(mapData *wad.MapData, secID int) int {
	if mapData == nil || secID < 0 || secID >= len(mapData.Sectors) {
		return 0
	}
	adj := findAdjacentSectors(mapData, secID)
	if len(adj) == 0 {
		return int(mapData.Sectors[secID].FloorHeight)
	}
	minFloor := math.MaxInt32
	for _, s := range adj {
		f := int(mapData.Sectors[s].FloorHeight)
		if f < minFloor {
			minFloor = f
		}
	}
	return minFloor
}

// getHighestAdjacentFloor computes the highest floor height of adjacent sectors.
func getHighestAdjacentFloor(mapData *wad.MapData, secID int) int {
	if mapData == nil || secID < 0 || secID >= len(mapData.Sectors) {
		return 0
	}
	adj := findAdjacentSectors(mapData, secID)
	if len(adj) == 0 {
		return int(mapData.Sectors[secID].FloorHeight)
	}
	maxFloor := math.MinInt32
	for _, s := range adj {
		f := int(mapData.Sectors[s].FloorHeight)
		if f > maxFloor {
			maxFloor = f
		}
	}
	return maxFloor
}
