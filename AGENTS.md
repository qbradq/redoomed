# AGENTS.md

Welcome to **ReDoomEd**! This document provides essential architectural context, conventions, and operational workflows for AI coding assistants and developers working on this codebase.

---

## 1. Project Overview

**ReDoomEd** is a modern Doom engine port and reimplementation written in pure Go. It combines:
- **[Ebitengine (v2)](https://github.com/hajimehoshi/ebiten)** for cross-platform 2D graphics, window management, and input handling.
- **2.5D Software Renderer** implementing classic Doom BSP front-to-back traversal, 1D column occlusion clipping, perspective-correct wall texturing, visplane floor/ceiling rendering, dynamic colormap/distance attenuation, and cylindrical skybox rendering.
- **WAD & Map Geometry Parser** for loading IWAD/PWAD lumps, vertexes, linedefs, sidedefs, sectors, subsectors, segs, nodes, things, patches, composite wall textures (`TEXTURE1`/`TEXTURE2`/`PNAMES`), flats, and font glyphs (`STCFNxxx`).
- **[Tengo Scripting](https://github.com/d5/tengo)** embedded REPL and autoexec script execution with persistent state and custom `game` bindings.
- **Quake-style Console** with interactive command line, scrollback, command history, and EGA color output.

---

## 2. Directory & Package Structure

```
.
├── main.go                       # Application entry point (1280x800 window setup, platform loop)
├── freedoom2.wad                 # Default bundled IWAD asset for testing & gameplay
├── go.mod, go.sum                # Go dependencies (Go 1.26.5, Ebitengine v2, Tengo v2)
├── pkg/
│   ├── platform/                 # App lifecycle & top-level game loop (ebiten.Game)
│   │   ├── app.go                # App struct, mode switching, IWAD loading, Tengo REPL wiring
│   │   └── app_test.go
│   ├── audio/                    # Music playback, MUS/MIDI/Vorbis/MP3 decoding, and synthesis
│   │   ├── manager.go            # MusicManager, audio context, volume, track looping
│   │   ├── mus.go                # Doom MUS to Standard MIDI File (SMF) converter
│   │   ├── synth.go              # Pure Go MIDI-to-PCM software synthesizer & stream decoders
│   │   └── audio_test.go
│   ├── mode/                     # Top-level application modes and layered UI stack
│   │   ├── mode.go               # Mode interface (Update, Draw)
│   │   ├── console.go            # Quake-style 640x400 console mode with unscii font & REPL
│   │   ├── console_test.go
│   │   ├── game.go               # 320x200 GameMode managing 7-layer stack, scaled to 1280x800
│   │   ├── game_test.go
│   │   └── layer.go              # Composable Layer interface & 7 layer implementations
│   ├── physics/                  # Collision detection, bounding box sampling, step/crack/ceiling checks, sliding
│   │   ├── actor.go              # Actor struct, player/monster physical properties and constructors
│   │   ├── collision.go          # CheckPosition, TryMove, SlideMove, Move, distance/overlap routines
│   │   └── physics_test.go
│   ├── render/                   # 2.5D Doom BSP raycasting / column software rasterizer
│   │   ├── renderer.go           # BSP traversal, 1D clipping, walls, visplanes, colormap, sky
│   │   └── renderer_test.go
│   ├── wad/                      # WAD file parsing, map geometry, textures, patches, fonts, music
│   │   ├── wad.go                # IWAD/PWAD file container & lump reader
│   │   ├── wad_test.go
│   │   ├── map.go                # Geometry lumps (VERTEXES, LINEDEFS, SECTORS, NODES, etc.)
│   │   ├── map_test.go
│   │   ├── music.go              # Music format detection and MapToMusic lookup tables
│   │   ├── music_test.go
│   │   ├── patch.go              # Doom Picture patch format decoder (column posts & transparency)
│   │   ├── texture.go            # TextureManager, TEXTURE1/2, PNAMES, flats, PLAYPAL, COLORMAP
│   │   └── font.go               # HUDFont (STCFNxxx patches converted to ebiten.Image)
│   ├── script/                   # Tengo scripting language integration
│   │   ├── repl.go               # Interactive REPL, symbol table, stdout redirection, game module
│   │   └── repl_test.go
│   ├── font/                     # Console text rendering
│   │   ├── font.go               # 8x8 fixed-width ConsoleFont using unscii-8.ttf via text/v2
│   │   └── font_test.go
│   ├── gfx/                      # Graphic & color definitions
│   │   └── colors.go             # Standard 16 EGA color constants (color.RGBA)
│   └── data/                     # Embedded binary assets (embed.FS)
│       ├── data.go               # FS embed definition for fonts/ and scripts/
│       ├── fonts/unscii-8.ttf    # Embedded 8x8 bitmapped TrueType font
│       └── scripts/
│           └── autoexec.tengo    # Startup Tengo script (welcome banner + game.StartMap("MAP01"))
```

---

## 3. Core Subsystems & Architecture

### A. Application Lifecycle & Modes (`pkg/platform`, `pkg/mode`)
- **Native Resolution**: 1280x800 window resolution (`ScreenWidth`, `ScreenHeight`).
- **`App`**: Implements `ebiten.Game`. Manages active `mode.Mode` (`ConsoleMode` vs `GameMode`), WAD file, fonts, and script REPL.
- **Mode Switching**:
  - Pressing **`~` / `` ` `` (Grave Accent)** toggles between `ConsoleMode` and `GameMode` (or previous mode).
- **`GameMode` Layer Stack** (ordered top-to-bottom):
  1. `CommonLayer`: Handles engine-level hotkeys (e.g. console toggle).
  2. `GameMenuLayer`: Pause and options menu (hidden by default).
  3. `GameControlsLayer`: Gameplay inputs (WASD/Arrows, Shift run, Q/E turn, Tab automap toggle).
  4. `HUDLayer`: Renders status bar (`STBAR` patch, 320x32) at bottom of screen.
  5. `MiniMapLayer`: 2D vector automap with line flag colors and player arrow; supports mouse wheel and Ctrl+/- zooming. Occludes lower layers when active.
  6. `LevelViewLayer`: 2.5D software-rendered level view via `render.Renderer`. Occludes lower layers when active.
  7. `IntermissionLayer`: Title/intermission full-screen graphic (e.g. `TITLEPIC` / `INTERPIC`).

### B. Collision Detection & Physics Engine (`pkg/physics`)
- **Actor Physics Properties**: `Radius` (16.0), `Height` (56.0), `EyeHeight` (41.0), `MaxStepHeight` (24.0).
- **Solid Wall Collision**: Line-to-point Euclidean segment distance checking against 1-sided linedefs and 2-sided linedefs with `LinedefBlocking` or `LinedefBlockMonsters`.
- **Step-Up Clamping**: Restricts vertical floor ascension to `MaxStepHeight` (24 Doom units); taller steps block movement.
- **Low Ceiling Clearance**: Prevents actors from moving into or under openings shorter than the actor's height (`CeilingZ - FloorZ < Height` or `CeilingZ - Z < Height`).
- **Floor Crack Avoidance**: Multi-point bounding box sampling queries all overlapping sectors and adopts `highestFloor`, ensuring actors standing over small cracks or trenches remain supported by adjacent floors.
- **Wall Sliding (`SlideMove`)**: Decomposes blocked diagonal vectors into unconstrained axial/tangential components for responsive movement along solid geometry.

### C. 2.5D Software Renderer (`pkg/render`)
- **Internal Resolution**: 320x168 active 2.5D viewport, rendered into a 320x200 buffer (`DefaultBufferHeight`).
- **Frustum & Field of View**: 90° horizontal FOV.
- **BSP Front-to-Back Traversal**: Traverses `mapData.Nodes` using partition lines and camera coordinates. Bounding box frustum culling prevents unnecessary sub-tree traversal.
- **1D Column Occlusion Clipping**: `ceilingClip` and `floorClip` arrays per column `x ∈ [0, 319]`. Once `ceilingClip[x] >= floorClip[x]-1`, column rendering terminates early.
- **Perspective-Correct Wall Texturing**: Computes exact 3D ray-line parametric intersection in camera space. Supports upper, middle, and lower wall textures, as well as `LinedefDontPegTop` and `LinedefDontPegBottom` alignment flags.
- **Visplanes (Floors & Ceilings)**: Span-based floor and ceiling drawing with 64x64 flat texture coordinate wrapping `(u & 63, v & 63)`.
- **Lighting & Attenuation**: Maps sector light levels (0–255) and distance to one of 32 colormap levels (from `COLORMAP` lump) for rich contrast and depth.
- **Sky Rendering**: Automatically maps Doom 1/2 episodes to `SKY1`–`SKY4`, projected as a 360° cylinder across 4 texture repetitions, rendered full-bright (colormap 0).
- **Framebuffer Output**: Direct byte-slice rasterization transferred to `ebiten.Image` via `WritePixels()`.

### D. WAD File & Map Geometry (`pkg/wad`)
- **Map Format**: Standard Doom map lumps (`VERTEXES`, `LINEDEFS`, `SIDEDEFS`, `SECTORS`, `SEGS`, `SSECTORS`, `NODES`, `THINGS`).
- **Texture Manager**: Resolves patches from `PNAMES`, composite textures from `TEXTURE1`/`TEXTURE2`, 64x64 flats, `PLAYPAL` palette 0 (768-byte RGB), and `COLORMAP`. Preloads map textures on map switch.
- **Spatial Queries**:
  - `FindSubsector(x, y)`: BSP traversal to locate the subsector containing point `(x, y)`.
  - `SectorAt(x, y)`: Finds the sector containing point `(x, y)`.
  - `Bounds()`: Computes 2D bounding box of map vertexes.
  - `Player1Start()`: Locates player 1 spawn thing (`Type == 1`).

### E. Scripting Engine (`pkg/script`)
- **Tengo REPL**: Interactive environment preserving symbol table and global variables across executions.
- **Registered Modules**:
  - Standard Tengo library modules (`math`, `text`, `times`, `rand`, `json`, etc.).
  - `game`: Custom module exposing `game.StartMap(name)` / `game.start_map(name)` and `game.exit()`.
  - `fmt`: Custom module redirecting `print`, `println`, `printf`, `sprintf`, `sprint`, `sprintln` to the console buffer.
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

# Run tests in a specific package
go test -v ./pkg/render
go test -v ./pkg/wad
```

---

## 6. Guidelines for Agents & Developers

1. **Maintain Pure Go & Ebitengine Compatibility**:
   - Avoid CGO or non-portable native libraries. Stick to standard Go and Ebitengine v2 APIs.
2. **Preserve Performance in the Render Loop**:
   - The 2.5D software renderer operates at 60 FPS in software; avoid heap allocations inside per-pixel loops (`renderSeg`, `drawFloorSpan`, `drawCeilingSpan`, `drawSkySpan`). Use precomputed lookup tables (e.g. `colAngles`, `colKx`, `colCos`, `paletteRGBA`).
3. **WAD Compatibility**:
   - Keep lump name lookup case-insensitive (always uppercase for lookups).
   - Handle missing textures/flats gracefully with sensible fallback colors rather than panicking.
4. **Testing**:
   - When adding new rendering features or map parsing logic, write tests with mock data or against `freedoom2.wad`. Ensure all tests pass via `go test ./...`.
