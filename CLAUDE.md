# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

An Ebitengine game jam 2026 entry. Written in Go using the Ebitengine game engine. Supports both desktop and WebAssembly (WASM) builds.

**Module:** `github.com/noppikinatta/ebitenginegamejam2026`

## Commands

```bash
make run        # Run locally
make build      # Build WASM for web (output: release/game.wasm)
make gen        # Run go generate
make test       # Run tests (verbose)
make test-cov   # Generate coverage HTML report
```

Run a single test:
```bash
go test ./lang/... -run TestFoo -v
```

## Architecture

### Entry Point

`app/main.go` — sets up 1280×720 window, creates input (via `nyuuryoku`) and scene sequence, then calls `ebiten.RunGame`.

### Scene System

`scene/sequence.go` — wires up scenes with `bamenn.Sequence` (library for scene transitions). Scene order: Title → InGame → Result.

- `scene/title.go` — Title screen: shows title image, story text, waits for click
- `scene/lang.go` — Press **L** during any scene to toggle language with fade animation
- InGame and Result scenes need to be implemented

### Key Packages

| Package | Role |
|---|---|
| `scene/` | All game scenes and scene wiring |
| `drawing/` | Rendering utilities (text with caching, images, gauge bars, rectangles) |
| `geom/` | `PointF` struct with 2D vector math (Add, Subtract, Multiply, Angle, Distance, etc.) |
| `lang/` | Bilingual (EN/JA) localization via CSV; supports Go `text/template` in strings |
| `asset/` | Embedded assets via `//go:embed` (fonts, images, sounds, lang CSVs) |

### Drawing Conventions

- **Text**: Use `drawing.DrawText` — it caches rendered glyphs; never create `ebiten.Image` per frame for text.
- **Shapes**: Use `drawing.DrawRect` / `drawing.WhitePixel` which use `DrawTriangles` internally (avoids batch breaks).
- **Images**: Load via `drawing.LoadImage(asset.ImgFS, "path")` which handles embedded FS.

### Localization

Language CSV files are at `asset/lang/en.csv` and `asset/lang/ja.csv` (currently empty — populate these as text is added). Keys are used in Go as `lang.Text("key")`. Toggle at runtime with **L** key.

### Asset Structure

- `asset/img/` — embedded images (add files here and register in `asset/embed.go`)
- `asset/sound/bgm/` — OGG background music
- `asset/sound/` — WAV sound effects (currently `explosion.wav`)
- Audio sample rate is 48000 Hz (`asset/sound.go`)

### Dependencies

- `github.com/hajimehoshi/ebiten/v2` — game engine
- `github.com/noppikinatta/bamenn` — scene transition management
- `github.com/noppikinatta/nyuuryoku` — input handling abstraction
