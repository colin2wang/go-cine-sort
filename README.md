# Cine Sort · Movie Organizer

![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white) ![Wails](https://img.shields.io/badge/Wails-v2-DF2835?logo=wails&logoColor=white) ![Vue](https://img.shields.io/badge/Vue-3.5-4FC08D?logo=vuedotjs&logoColor=white) ![TypeScript](https://img.shields.io/badge/TypeScript-5.7-3178C6?logo=typescript&logoColor=white) ![SQLite](https://img.shields.io/badge/SQLite-003B57?logo=sqlite&logoColor=white)

![Wails](https://img.shields.io/badge/Wails-v2-red?logo=wails&logoColor=white) ![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go&logoColor=white) ![Vue](https://img.shields.io/badge/Vue-3.5-4FC08D?logo=vuedotjs&logoColor=white) ![TypeScript](https://img.shields.io/badge/TypeScript-5.7-3178C6?logo=typescript&logoColor=white) ![SQLite](https://img.shields.io/badge/SQLite-1.62-003B57?logo=sqlite&logoColor=white) ![Vite](https://img.shields.io/badge/Vite-6.0-646CFF?logo=vite&logoColor=white) ![pnpm](https://img.shields.io/badge/pnpm-12.18-F69220?logo=pnpm&logoColor=white)

Cine Sort is a desktop movie library organizer. Pick a movie folder and the app automatically detects video files, fetches film metadata online, generates a normalized directory layout with metadata files, downloads official posters, records everything in a SQLite history database, and optionally archives the source videos into the organized directory.

## Tech Stack

| Layer | Choice |
| --- | --- |
| Desktop framework | Wails **2.17.1** |
| Backend | Go 1.26+, stdlib `net/http` + `regexp`, `gopkg.in/yaml.v3`, `modernc.org/sqlite` (pure Go, no CGO) |
| Frontend | Vue 3 + TypeScript + Vite + vue-router, **pnpm** |
| Icons | `oksvg` + `rasterx` renders a single SVG → PNG/ICO |

## Getting Started

```powershell
# Dev mode (hot reload)
wails dev

# Generate icons (standalone step)
.\generate-icon.ps1

# Build a single-file exe
.\build.ps1
```

`build.ps1` runs in order: verifies `go` / `node` / `pnpm` and the Wails CLI version (auto `go install ...@v2.17.1` if missing) → generates icons → `pnpm install --frozen-lockfile` → `pnpm run build` → `wails build -clean -s`. Output:

```
build/bin/go-cine-sort.exe
```

Common flags:

```powershell
.\build.ps1 -SkipIcon       # reuse existing icons
.\build.ps1 -SkipInstall    # reuse frontend/node_modules
.\build.ps1 -SkipIcon -SkipInstall
```

`generate-icon.ps1` uses `frontend/src/assets/logo.svg` as the single icon source and outputs:

| File | Purpose |
| --- | --- |
| `build/appicon.png` | 512×512 Wails app icon |
| `build/windows/icon.ico` | 16/24/32/48/64/128/256 multi-size, embedded in the exe |
| `frontend/public/logo.svg` | Page favicon (auto-synced) |

No ImageMagick, Inkscape, or Python required.

## Configuration

On first run, a config file is generated at `%APPDATA%\go-cine-sort\config.yml`. Edit it on the **Settings** page or by hand as YAML. The database and logs live in the same directory (`cine_sort.db`, `logs/`). No drive letters are hardcoded.

Key options:

```yaml
movie_folder: ""                                    # movie directory (required)
directory_format: "{chinese_title}.{english_title}.{year}"
recursive: true
archive_move: false          # archive/move videos, off by default
conflict_strategy: rename    # rename / skip / overwrite
create_rating_file: true
create_sid_file: true
download_posters: true
fetch_imdb: true
```

Template variables: `{chinese_title}`, `{english_title}`, `{year}`. Cleaning rules (`extensions` / `cleanup_patterns` / `tech_patterns` / `language_patterns`) are embedded in `internal/config/defaults.yml` and can be overridden from the user config.

## Project Layout

```
main.go / app.go              Wails entry and frontend bindings
wails.json                    Project config
build.ps1                     Packaging script
generate-icon.ps1             Icon generation script
build/                        Icons and Windows manifest assets
tools/icongen/                Pure-Go SVG → PNG/ICO renderer
internal/
  config/                    Config loading and precompiled regexes (embedded defaults)
  model/                     Domain types and the frontend/backend JSON contract
  netclient/                 Shared-cookie HTTP client and Douban PoW verification
  scanner/                   Video scanning, filename cleanup, and year scoring
  douban/                    Search, detail parsing, poster download
  imdb/                      GraphQL ratings
  organizer/                 Directory naming, metadata files, archive move
  store/                     SQLite persistence
  logging/                   Leveled logging with a frontend event sink
  service/                   Batch / SID pipeline orchestration
  app/                       Wails event names and bridge callbacks
frontend/src/
  views/                     Batch organize, single title, history, settings
  components/                Sidebar, top bar, status bar, tables, progress, logs, toasts
  store/                     Task state and global toasts
  assets/logo.svg            Single SVG icon source
  api/                       Wails bindings and event wrappers
```

## Testing

```powershell
go test ./... -count=1
pnpm --dir frontend run typecheck
```

The parsing layers run offline assertions against real Douban HTML fixtures extracted into the project's test data — no network required.
