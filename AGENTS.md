# AGENTS.md

Welcome to **ReDoomEd**! This document provides essential architectural context, conventions, and operational workflows for AI coding assistants and developers working on this codebase.

---

## 1. Project Overview

**ReDoomEd** is a modern Doom engine port, level editor, and reimplementation written in pure Go. It combines:
- **[Ebitengine (v2)](https://github.com/hajimehoshi/ebiten)** for cross-platform 2D graphics, window management, and input handling.
- **2.5D Software Renderer** implementing classic Doom BSP front-to-back traversal, 1D column occlusion clipping, 2D depth buffering, perspective-correct wall texturing, 2-sided transparent/masked mid-texture rendering, visplane floor/ceiling rendering, dynamic colormap/distance attenuation, cylindrical skybox rendering, and item/sprite rendering.
- **WAD & Map Geometry Parser** for loading IWAD/PWAD lumps, vertexes, linedefs, sidedefs, sectors, subsectors, segs, nodes, things, patches, composite wall textures (`TEXTURE1`/`TEXTURE2`/`PNAMES`), flats, item definitions, and font glyphs (`STCFNxxx`).
- **Player Stats & Inventory System** managing health, armor, ammo pools (bullets, shells, rockets, cells), 9 weapon types, 6 key types, backpacks, and item pickup collision/dispatching.
- **Interactive 2D Map Editor Mode** (`640x400`) featuring vertex, linedef, sector, and thing editing sub-modes, pan/zoom canvas, grid alignment, hover inspection, and a 15-slot icon toolbar.
- **[Tengo Scripting](https://github.com/d5/tengo)** embedded REPL and line special trigger system with persistent state and custom `game`, `map`, `player`, and `fmt` module bindings.
- **Quake-style Console** with interactive command line, scrollback, command history, and EGA color output.

---

## 2. Directory & Package Structure

```
.
├── main.go                       # Application entry point (1280x800 window setup, platform loop)
├── freedoom2.wad                 # Bundled IWAD asset for testing & gameplay
├── doom1.wad, DoomShareware.wad  # Alternative shareware IWAD assets
├── go.mod, go.sum                # Go dependencies (Go 1.26.5, Ebitengine v2, Tengo v2)
├── src/                          # Design source assets
│   └── gfx/editor_icons.aseprite # Aseprite source for editor toolbar icons
├── pkg/
│   ├── platform/                 # App lifecycle & top-level game loop (ebiten.Game)
│   │   ├── app.go                # App struct, mode switching (Console/Game/Editor), IWAD loading, REPL wiring
│   │   └── app_test.go
│   ├── audio/                    # Music playback, MUS/MIDI/Vorbis/MP3 decoding, and synthesis
│   │   ├── manager.go            # MusicManager, audio context, volume, track looping
│   │   ├── mus.go                # Doom MUS to Standard MIDI File (SMF) converter
│   │   ├── synth.go              # Pure Go MIDI-to-PCM software synthesizer & stream decoders
│   │   └── audio_test.go
│   ├── player/                   # Player statistics, inventory state, and item pickups
│   │   ├── stats.go              # PlayerStats (Health, Armor, Ammo, Keys, Weapons, Slots)
│   │   ├── stats_test.go
│   │   ├── items.go              # Item pickup dispatching, ammo caps, and pickup messages
│   │   └── items_test.go
│   ├── physics/                  # Collision detection, bounding box sampling, step/crack/ceiling checks, sliding, items
│   │   ├── actor.go              # Actor struct, player/monster physical properties and constructors
│   │   ├── collision.go          # CheckPosition, TryMove, SlideMove, Move, distance/overlap routines
│   │   ├── use.go                # UseLine player interaction raycasting (64-unit range)
│   │   ├── item.go               # CheckItemTouch, TouchItems (2D distance & 3D vertical span checking)
│   │   ├── item_test.go
│   │   ├── use_test.go
│   │   └── physics_test.go
│   ├── render/                   # 2.5D Doom BSP raycasting / column software rasterizer
│   │   ├── renderer.go           # BSP traversal, 1D clipping, 2D depth buffer, walls, masked mid-textures, visplanes, colormap, sky, items
│   │   └── renderer_test.go
│   ├── mode/                     # Top-level application modes and layered UI stack
│   │   ├── mode.go               # Mode interface (Update, Draw)
│   │   ├── console.go            # Quake-style 640x400 console mode with unscii font & REPL
│   │   ├── console_test.go
│   │   ├── game.go               # 320x200 GameMode managing 7-layer stack, scaled to 1280x800
│   │   ├── game_test.go
│   │   ├── layer.go              # Composable Layer interface & 7 layer implementations
│   │   ├── editor.go             # 640x400 2D Map Editor mode (Vertex, Line, Sector, Thing modes, toolbar, pan/zoom)
│   │   ├── editor_test.go
│   │   ├── hud_assets.go         # HUD status bar numbers, ammo counters, and graphic compositing
│   │   └── item_test.go
│   ├── wad/                      # WAD file parsing, map geometry, textures, patches, items, fonts, music
│   │   ├── wad.go                # IWAD/PWAD file container & lump reader
│   │   ├── wad_test.go
│   │   ├── map.go                # Geometry lumps (VERTEXES, LINEDEFS, SECTORS, NODES, SEGS, SSECTORS, THINGS)
│   │   ├── map_test.go
│   │   ├── items.go              # Item definitions, categories, thing IDs, sprite names, ItemEntity state
│   │   ├── items_test.go
│   │   ├── music.go              # Music format detection and MapToMusic lookup tables
│   │   ├── music_test.go
│   │   ├── patch.go              # Doom Picture patch format decoder (column posts & transparency)
│   │   ├── texture.go            # TextureManager, TEXTURE1/2, PNAMES, flats, PLAYPAL, COLORMAP
│   │   └── font.go               # HUDFont (STCFNxxx patches converted to ebiten.Image)
│   ├── script/                   # Tengo scripting language integration
│   │   ├── repl.go               # Interactive REPL, symbol table, stdout redirection, game/map/player/fmt modules
│   │   ├── repl_test.go
│   │   ├── map_module.go         # ID-based "map" module exposing geometry, sectors, lines, tags, neighbor queries
│   │   ├── map_test.go
│   │   ├── player_module.go      # "player" module exposing stats, health, armor, ammo, keys, weapons
│   │   ├── cache.go              # ScriptCache for caching compiled scripts & line special dispatchers
│   │   ├── cache_test.go
│   │   └── line_special_test.go
│   ├── font/                     # Console text rendering
│   │   ├── font.go               # 8x8 fixed-width ConsoleFont using unscii-8.ttf via text/v2
│   │   └── font_test.go
│   ├── gfx/                      # Graphic & color definitions
│   │   └── colors.go             # Standard 16 EGA color constants (color.RGBA)
│   └── data/                     # Embedded binary assets (embed.FS)
│       ├── data.go               # FS embed definition for fonts/, scripts/, and gfx/
│       ├── fonts/unscii-8.ttf    # Embedded 8x8 bitmapped TrueType font
│       ├── gfx/editor_icons.png  # Embedded 128x128 editor toolbar icons atlas
│       └── scripts/
│           ├── autoexec.tengo    # Startup Tengo script (welcome banner + game.StartMap("MAP01"))
│           └── lines/            # Line special scripts (line_nnn.tengo for doors, lifts, stairs, etc.)
```

---

## 3. Core Subsystems & Architecture

### A. Application Lifecycle & Modes (`pkg/platform`, `pkg/mode`)
- **Native Resolution**: 1280x800 logical window resolution (`ScreenWidth`, `ScreenHeight`).
- **`App`**: Implements `ebiten.Game`. Manages active `mode.Mode` (`ConsoleMode`, `GameMode`, `EditorMode`), IWAD loading, audio, and script REPL.
- **Mode Switching & Keybindings**:
  - **`` ` `` / `~` (Grave Accent)**: Toggles `ConsoleMode` overlay on top of the current mode.
  - **`F12`**: Toggles between `EditorMode` and `GameMode`.
  - Mouse cursor is automatically captured (`CursorModeCaptured`) in `GameMode` and released (`CursorModeVisible`) in `ConsoleMode` and `EditorMode`.
- **`GameMode` Layer Stack** (ordered top-to-bottom):
  1. `CommonLayer`: Engine-level hotkeys (Grave Accent console toggle, F12 editor toggle).
  2. `GameMenuLayer`: Pause and options menu.
  3. `GameControlsLayer`: Gameplay inputs (WASD movement, Turn, Shift run, E/Spacebar use, Tab automap toggle, mouse look).
  4. `HUDLayer`: Status bar (`STBAR`, 320x32) displaying health, armor, ammo counters, keys, and weapon slots.
  5. `MiniMapLayer`: 2D vector automap with line flag colors and player arrow; supports mouse wheel and Ctrl+/- zooming.
  6. `LevelViewLayer`: 2.5D software-rendered level view via `render.Renderer`.
  7. `IntermissionLayer`: Full-screen title/intermission graphic (`TITLEPIC` / `INTERPIC`).

### B. 2D Map Editor Mode (`pkg/mode/editor.go`)
- **Base Resolution**: 640x400, scaled to 1280x800.
- **Layout**:
  - **Editing Grid Window** (`400x392` at `[0, 0]`): 2D top-down map view with grid snapping and real-time geometry rendering.
  - **User Panel** (`240x392` at `[400, 0]`): 15-slot icon toolbar (16x16 icons from `editor_icons.png`), mode selector, and property inspector.
  - **Status Bar** (`640x8` at `[0, 392]`): Shows cursor world coordinates, grid size, zoom scale, and selected element info.
- **Editing Sub-Modes**:
  - `EditModeVertex`: Highlight and inspect map vertexes.
  - `EditModeLine`: Highlight and inspect linedefs and sidedef texture assignments.
  - `EditModeSector`: Highlight and inspect sector floor/ceiling heights, lights, and flats.
  - `EditModeThing`: Highlight and inspect thing placements, flags, and item types.
- **Navigation & Controls**:
  - Panning: Middle-mouse drag or right-click drag.
  - Zooming: Mouse wheel, toolbar buttons, or +/- keys across 12 discrete zoom levels (1/64 to 32 pixels/unit).
  - Grid: Configurable grid size from 1 to 256 map units.

### C. Player Statistics, Inventory & Items (`pkg/player`, `pkg/physics`, `pkg/wad`)
- **Player Stats (`pkg/player/stats.go`)**:
  - Health: 0–100 (standard), up to 200 (super / soulsphere / megasphere).
  - Armor: 0–100 (Green armor / 1/3 absorption), up to 200 (Blue armor / 1/2 absorption).
  - Ammo Pools: Bullets (max 200/400), Shells (max 50/100), Rockets (max 50/100), Cells (max 300/600).
  - Weapons: Fist, Pistol, Shotgun, Chaingun, Rocket Launcher, Plasma Rifle, BFG9000, Chainsaw, Super Shotgun.
  - Keys: Blue, Yellow, and Red Keycards and Skull Keys.
- **Item System (`pkg/wad/items.go`, `pkg/player/items.go`)**:
  - Items cataloged into categories: `Key`, `Health`, `Armor`, `Ammo`, `Weapon`, `Powerup`.
  - Item entities tracked in `MapData.Items` as `ItemEntity` structs.
  - Collision & Collection: `physics.CheckItemTouch` tests 2D radial overlap and 3D vertical span overlap `[FloorZ, FloorZ + Height]`. On touch, `PlayerStats.TryPickupItem` validates benefit and applies effects.

### D. Collision Detection & Physics Engine (`pkg/physics`)
- **Actor Properties**: `Radius` (16.0), `Height` (56.0), `EyeHeight` (41.0), `MaxStepHeight` (24.0).
- **Solid Wall Collision**: Euclidean line segment distance checking against 1-sided linedefs and 2-sided linedefs with `LinedefBlocking` or `LinedefBlockMonsters`.
- **Step-Up Clamping**: Restricts vertical floor ascension to `MaxStepHeight` (24 Doom units); taller steps block movement.
- **Low Ceiling Clearance**: Prevents actors from moving into or under openings shorter than the actor's height (`CeilingZ - FloorZ < Height` or `CeilingZ - Z < Height`).
- **Floor Crack Avoidance**: Multi-point bounding box sampling queries all overlapping sectors and adopts `highestFloor`.
- **Wall Sliding (`SlideMove`)**: Decomposes blocked diagonal movement into tangential slide vectors.
- **Line Interaction (`UseLine`)**: Casts an interaction ray (`DefaultUseRange = 64.0`) to find usable linedefs.

### E. 2.5D Software Renderer (`pkg/render`)
- **Internal Resolution**: 320x168 active 2.5D viewport in a 320x200 buffer (`DefaultBufferHeight`).
- **Frustum & Field of View**: 90° horizontal FOV.
- **BSP Traversal & Culling**: Front-to-back traversal through `MapData.Nodes` with node bounding box frustum culling.
- **1D Column Occlusion & 2D Depth Buffer**:
  - 1D `ceilingClip` and `floorClip` arrays per column `x ∈ [0, 319]`.
  - 2D `depthBuffer` (`viewWidth × bufHeight`) for exact per-pixel occlusion between solid walls, visplanes, masked middle textures, and sprites/items.
- **Perspective-Correct Wall Texturing**: Computes exact 3D ray-line parametric intersections in camera space with upper, lower, and middle texture alignment (`LinedefDontPegTop`, `LinedefDontPegBottom`).
- **Masked 2-Sided Middle Textures**: Queues transparent 2-sided middle textures (iron grates, window bars) during BSP traversal and renders them in a back-to-front pass with column clipping and color keying.
- **Visplanes (Floors & Ceilings)**: Span-based floor and ceiling drawing with 64x64 flat texture coordinate wrapping `(u & 63, v & 63)`.
- **Lighting & Attenuation**: Maps sector light level and Euclidean distance to 32 colormap levels from `COLORMAP`.
- **Sky Rendering**: Cylindrical 360° skybox mapped to `SKY1`–`SKY4`, rendered full-bright.
- **Sprites & Items**: Perspective-correct sprite rasterization with depth testing against the software depth buffer.

### F. WAD File & Map Geometry (`pkg/wad`)
- **Map Format**: Standard Doom map lumps (`VERTEXES`, `LINEDEFS`, `SIDEDEFS`, `SECTORS`, `SEGS`, `SSECTORS`, `NODES`, `THINGS`).
- **Texture Manager**: Resolves composite textures from `TEXTURE1`/`TEXTURE2` and `PNAMES`, 64x64 flats, `PLAYPAL` palettes, and `COLORMAP`. Preloads map textures on map switch.
- **Spatial Queries**:
  - `FindSubsector(x, y)`: BSP traversal to locate the subsector containing point `(x, y)`.
  - `SectorAt(x, y)` / `SectorIndexAt(x, y)`: Finds the sector containing point `(x, y)`.
  - `Bounds()`: Computes 2D bounding box of map vertexes.
  - `Player1Start()`: Locates player 1 spawn thing (`Type == 1`).

### G. Scripting Engine (`pkg/script`)
- **Tengo REPL**: Interactive environment preserving symbol table and global variables across executions.
- **Registered Modules**:
  - `game`: `StartMap(name)`, `exit()`, `play_music(track)`, `stop_music()`, `set_music_volume(v)`, `set_noclip(bool)`.
  - `map`: Geometry inspection and manipulation (sectors, lines, tags, floor/ceiling heights, neighbor sector queries).
  - `player`: Player stats query and manipulation (health, armor, ammo, keys, weapons, give/take items).
  - `fmt`: Redirects `print`, `println`, `printf`, `sprintf` to console output.
- **Line Specials (`pkg/data/scripts/lines/`)**:
  - Line specials (e.g. doors, lifts, stairs, teleporters, exits) are implemented as individual Tengo scripts (`line_001.tengo`, `line_011.tengo`, `line_062.tengo`, etc.) compiled and cached by `ScriptCache`.
- **Autoexec**: Runs `scripts/autoexec.tengo` on engine startup.

---

## 4. Coordinate Systems & Conventions

- **Doom World Space**:
  - $X$: East (+X) / West (-X)
  - $Y$: North (+Y) / South (-Y)
  - $Z$: Floor-to-ceiling elevation (Up)
  - **Angles**: In degrees, where $0^\circ = \text{East}$, $90^\circ = \text{North}$, $180^\circ = \text{West}$, $270^\circ = \text{South}$.
- **Screen Space**:
  - $X$: $0$ (left) to $\text{width}-1$ (right)
  - $Y$: $0$ (top) to $\text{height}-1$ (bottom)
- **Player Camera**:
  - Default eye height is $41.0$ Doom units above sector floor.

---

## 5. Development & Testing Workflows

### Build & Run
```bash
# Run ReDoomEd
go run .

# Build executable binary
go build -o redoomed .
```

### Running Tests
```bash
# Run all package tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests in specific packages
go test -v ./pkg/render
go test -v ./pkg/wad
go test -v ./pkg/player
go test -v ./pkg/physics
go test -v ./pkg/mode
go test -v ./pkg/script
```

---

## 6. Guidelines for Agents & Developers

1. **Maintain Pure Go & Ebitengine Compatibility**:
   - Avoid CGO or non-portable native libraries. Stick to standard Go and Ebitengine v2 APIs.
2. **Preserve Performance in the Render Loop**:
   - The 2.5D software renderer operates at 60 FPS in software; avoid heap allocations inside per-pixel loops (`renderSeg`, `drawFloorSpan`, `drawCeilingSpan`, `drawSkySpan`, `renderMaskedSegs`). Use precomputed lookup tables (`colAngles`, `colKx`, `colCos`, `paletteRGBA`).
3. **WAD Compatibility**:
   - Keep lump name lookup case-insensitive (always uppercase for lookups).
   - Handle missing textures/flats gracefully with sensible fallback colors rather than panicking.
4. **Testing**:
   - When adding new rendering features, game logic, or map parsing logic, write tests with mock data or against `freedoom2.wad`. Ensure all tests pass via `go test ./...`.
