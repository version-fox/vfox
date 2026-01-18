# internal/env Knowledge Base

**Generated:** 2026-01-17
**Commit:** v0.x
**Package:** env - Environment variable management and shell integration

## OVERVIEW
Core package for scope-aware environment variable management and shell integration with dual merge semantics (Paths append, Vars override).

## STRUCTURE
```
internal/env/
├── env.go              # Vars, Envs structures, scope-aware merging
├── context.go          # RuntimeEnvContext, config loading, HTTP client
├── scope.go            # Global, Project, Session scope definitions
├── path.go             # PATH handling with SortedSet, ToBinPaths
├── state.go            # ConfigState for config file change tracking
├── vfox_toml_chain.go  # Chain of configs with priority lookup
├── env_unix.go         # Unix-specific exports
├── env_win.go          # Windows-specific exports
├── symlink_unix.go     # Unix symlink handling
├── symlink_windows.go  # Windows shim handling
├── flag.go             # Environment variable name constants
└── *_test.go           # Unit tests
```

## WHERE TO LOOK
| Task | Location | Notes |
|------|----------|-------|
| **Environment merging** | `env.go:MergeByScopePriority` | Dual-merge: Paths append, Vars override |
| **Scope management** | `scope.go` | Global=0, Project=1, Session=2 |
| **PATH operations** | `path.go` | SortedSet-based, ToBinPaths for executables |
| **Config loading** | `context.go:LoadVfoxTomlByScope` | Per-scope config loader |
| **Config chain** | `vfox_toml_chain.go` | Multi-config with priority lookup |
| **Change tracking** | `state.go:HasChanged` | mtime-based, PATH caching |
| **Shell exports** | `env_*.go` | ToExport() generates shell-specific output |

## CONVENTIONS

### Scope-Aware Environment Merging (CRITICAL)
**PATH (append):** First added = HIGHEST, appears FIRST in PATH. Order: Project → Session → Global.
**Vars (override):** Last added wins = HIGHEST. Order: Global → Session → Project (reverse of PATH).

### Config Chain Priority
Configs added in order (first = lowest priority). Tool lookup searches tail-to-head. `GetToolConfig()` returns scope.

### State Caching
Tracks config file mtimes, caches PATH for user changes, stores shell output for fast reload.

## ANTI-PATTERNS (THIS PACKAGE)

### Merging Violations
1. Wrong PATH/vars merge order breaks priority semantics (different orders required)
2. Same order for both PATH and Vars (dual semantics mandatory)

### State Management
1. Ignoring PATH or project changes causes stale state
2. State operations must be fast (JSON on disk)

### Config Chain
1. Always use `GetToolConfig()` (tail-to-head priority lookup)
2. Don't merge entire chain just to find one tool
3. Always check scope returned from `GetToolConfig()`
