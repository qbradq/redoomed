package mode

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/qbradq/redoomed/pkg/wad"
)

func TestGameModeItemPickup(t *testing.T) {
	// Create mock map data with player start and an armor bonus nearby
	mapData := &wad.MapData{
		Name: "TEST_ITEMS",
		Vertexes: []wad.Vertex{
			{X: 0, Y: 0},
			{X: 200, Y: 0},
			{X: 200, Y: 200},
			{X: 0, Y: 200},
		},
		Linedefs: []wad.Linedef{
			{V1: 0, V2: 1, Flags: 0, RightSide: 0, LeftSide: 0xFFFF},
			{V1: 1, V2: 2, Flags: 0, RightSide: 0, LeftSide: 0xFFFF},
			{V1: 2, V2: 3, Flags: 0, RightSide: 0, LeftSide: 0xFFFF},
			{V1: 3, V2: 0, Flags: 0, RightSide: 0, LeftSide: 0xFFFF},
		},
		Sidedefs: []wad.Sidedef{
			{Sector: 0, MiddleTexture: "STARTAN2"},
		},
		Sectors: []wad.Sector{
			{FloorHeight: 0, CeilingHeight: 128, FloorPic: "FLOOR0_1", CeilingPic: "CEIL1_1", LightLevel: 192},
		},
		Things: []wad.Thing{
			{X: 50, Y: 50, Angle: 0, Type: wad.ThingPlayer1Start},
			{X: 70, Y: 50, Angle: 0, Type: wad.ThingArmorBonus},
			{X: 150, Y: 150, Angle: 0, Type: wad.ThingKeyBlueCard},
		},
	}

	gm := NewGameMode("", nil, nil)
	gm.levelViewLayer.SetMapData(mapData)
	gm.miniMapLayer.SetMapData(mapData)
	gm.miniMapLayer.SetItems(gm.levelViewLayer.Items())
	gm.levelViewLayer.SetVisible(true)

	if len(gm.Items()) != 2 {
		t.Fatalf("expected 2 items, got %d", len(gm.Items()))
	}

	var logged []string
	gm.SetOnLog(func(msg string) {
		logged = append(logged, msg)
	})

	var pickedUpItems []*wad.ItemEntity
	gm.SetOnItemPickup(func(item *wad.ItemEntity, msg string) {
		pickedUpItems = append(pickedUpItems, item)
	})

	// Initial player stats
	if gm.PlayerStats().Armor != 0 {
		t.Errorf("expected initial armor 0, got %d", gm.PlayerStats().Armor)
	}

	// Move player forward (toward X=70, where armor bonus is placed)
	gm.levelViewLayer.MovePlayer(15.0, 0, 0)
	gm.CheckItemPickups()

	// Verify armor bonus picked up
	if gm.PlayerStats().Armor != 1 {
		t.Errorf("expected player armor to increase to 1, got %d", gm.PlayerStats().Armor)
	}
	if len(pickedUpItems) != 1 || pickedUpItems[0].Def.Type != wad.ThingArmorBonus {
		t.Errorf("expected armor bonus in pickedUpItems, got %+v", pickedUpItems)
	}
	if len(logged) == 0 {
		t.Errorf("expected log message for armor bonus pickup")
	}

	// Verify item is marked collected
	if !gm.Items()[0].Collected {
		t.Errorf("expected item 0 to be marked Collected")
	}

	// Test MiniMap drawing with items
	screen := ebiten.NewImage(GameBufferWidth, GameBufferHeight)
	gm.miniMapLayer.SetVisible(true)
	gm.miniMapLayer.Draw(screen)
}
