# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**vfox** (Version Fox) is a cross-platform SDK version manager similar to `nvm`, `fvm`, `sdkman`, or `asdf-vm`. It allows developers to quickly install and switch between different runtime versions (Node.js, Java, Python, etc.) via a Lua-based plugin system.

## Build and Test Commands

```bash
# Build the project
go build .

# Run tests
go test ./...

# Run tests with coverage
go test ./... -coverprofile=coverage.out -covermode=atomic

# Run tests for specific package
go test ./internal/sdk -v

# Run with debug mode
vfox --debug <command>

# Install dependencies
go mod tidy
```

## High-Level Architecture

### Core Design: Three-Tier Scope System

vfox manages SDK versions across three scopes with **stable symlink/shim paths**:

1. **Global Scope**: Applies across all shells (stored in `~/.vfox/.vfox.toml`)
2. **Project Scope**: Project-specific (stored in `$PWD/.vfox.toml` or `$PWD/vfox.toml` in project root)
3. **Session Scope**: Shell session-specific (stored in `~/.vfox/tmp/{pid}/.vfox.toml`)

The key innovation: using **fixed symlink paths** (Unix) or **shim executables** (Windows) for all three scopes, rather than temporary directories. This ensures:
- Session configurations survive across shell restarts
- Virtual environments (venv) work correctly without PATH conflicts
- Priority is visible in the PATH string itself

**PATH Priority** (left to right): Project > Session > Global > System

### Dual-Path Architecture

**User Paths** (`~/.vfox/` via `VFOX_HOME`):
- `tmp/`: Session-specific temporary data
- `cache/`: Download cache (per-user)
- `config.yaml`: User-level configuration overrides
- `.vfox.toml`: Global scope version specifications

**Shared Paths** (via `VFOX_ROOT`):
- Default: `/opt/vfox` (Unix) or `C:\Program Files\vfox` (Windows)
- `installs/`: Actual SDK installations
- `plugins/`: Plugin definitions
- `config/`: Global configuration

**Working Paths** (dynamic):
- `ProjectShimPath`: `.vfox/sdk` (project-level symlinks in project directory)
- `SessionShimPath`: `~/.vfox/tmp/{pid}/sdk` (session-level symlinks)
- `GlobalShimPath`: `~/.vfox/sdk` (global-level symlinks)

### Component Layers

```
CLI Layer (cmd/)
    ↓
Manager Layer (internal/manager.go)
    ↓
    ├─→ SDK Module (internal/sdk/)
    ├─→ Plugin System (internal/plugin/)
    ├─→ PathMeta (internal/pathmeta/)
    └─→ Environment/Shell (internal/env/)
```

#### Key Components

**Manager** (`internal/manager.go`):
- Central orchestration component
- Manages SDK lifecycle (install, uninstall, use, unuse)
- Handles plugin operations (add, remove, update)
- Coordinates version resolution across scopes

**PathMeta** (`internal/pathmeta/path_meta.go`):
- Defines all path structures (user, shared, working)
- Separates concerns: user-specific vs. shared resources
- Provides stable paths for symlinks/shims

**SDK Module** (`internal/sdk/sdk.go`):
- Represents a single SDK (e.g., nodejs, java)
- Handles version installation and removal
- Interfaces with Lua plugins for SDK-specific operations
- Provides environment variables and paths for specific versions

**Plugin System** (`internal/plugin/`):
- **Lua-based**: Each SDK is managed by a Lua plugin
- **Registry**: Central plugin registry at `https://registry.vfox.dev`
- **Plugin formats**: Single `.lua` file or `.zip` archive
- **Compatibility**: Checks minimum runtime version requirements
- **Legacy support**: Can parse `.nvmrc`, `.node-version`, `.sdkmanrc`, `.tool-versions` (auto-migrated to `.vfox.toml`)

**Shell Integration** (`internal/env/`):
- **Activation**: Generates shell-specific initialization scripts
- **Hooks**: Auto-switch versions when changing directories (chpwd, precmd, prompt hooks)
- **Supported shells**: Bash, Zsh, Fish, PowerShell, Clink, Nushell
- **Environment management**: Exports PATH and other environment variables

## Development Patterns

### Test-First Development
When implementing new features:
1. Write the test first based on the expected behavior
2. Implement the minimum code to make the test pass
3. Refactor and ensure tests still pass
4. Run full test suite before committing: `go test ./...`

### Manager Initialization Pattern
```go
manager, err := internal.NewSdkManager()
if err != nil {
    return err
}
defer manager.Close()
```

### Error Handling Pattern
```go
if err != nil {
    return cli.Exit(fmt.Sprintf("error message: %w", err), 1)
}
```

### SDK Operations Pattern
```go
sdk, err := manager.LookupSdk("nodejs")
if err != nil {
    return err
}
// Use sdk.Install(), sdk.Use(), etc.
```

## Package Organization

The `internal/` directory contains the core packages organized by responsibility:

### Core Packages

- **`manager`**: Central orchestration layer, coordinates all SDK operations
- **`sdk`**: SDK abstraction, handles installation, removal, and environment lookup
- **`plugin`**: Lua plugin system, loads and executes plugin scripts
- **`env`**: Environment variable management and shell integration
- **`pathmeta`**: Path metadata and configuration file management (TOML, legacy formats)
- **`config`**: vfox application configuration (proxy, registry, storage settings)

### Shared Packages (`internal/shared/`)

These are utility packages that can be imported by any internal package:
- **`util`**: Common utility functions
- **`logger`**: Logging utilities
- **`cache`**: Caching utilities
- **`shim`**: Shim executables for Windows
- **`printer`**: Formatted output utilities

### Key Principles

1. **No circular dependencies**: Follow the layered architecture described above
2. **Clear separation of concerns**: Each package has a single, well-defined responsibility
3. **Configuration via pathmeta**: All version config loading goes through `pathmeta.LoadConfig()`
4. **Shared utilities only**: `shared/*` packages should not contain business logic
5. **Test-driven development**: Every implementation must have corresponding unit tests in `*_test.go` files

## Important Design Decisions

### 1. Shared Root Architecture
- Single shared location for all SDK installations (`VFOX_ROOT`)
- Separates user-specific data from shared SDK installations
- Falls back to `~/.vfox` if `VFOX_ROOT` is not set
- Supports multi-user scenarios

### 2. Symlink vs. Shim Strategy
- **Unix**: Uses symbolic links for SDK binaries
- **Windows**: Uses shim executables
- Three-tier shim directories ensure proper priority without complex logic

### 3. Plugin-Based Extensibility
Each SDK is managed by a Lua plugin that defines:
- How to list available versions
- How to install/uninstall
- How to provide environment variables
- How to parse legacy config files

### 4. Scope Priority Resolution
Version resolution follows: Project > Session > Global > System
This is enforced by PATH order, making priority transparent and debuggable.

### 5. Package Dependency Architecture
The codebase follows a strict layered architecture with **no circular dependencies**:

**Dependency hierarchy** (top to bottom):
```
manager (orchestration layer)
  ↓
sdk (SDK abstraction)
  ↓
plugin (Lua plugin system)
  ↓
env (environment variables & shell integration)
  ↓
pathmeta, config (metadata & configuration)
  ↓
shared/* (utilities: logger, util, cache, shim)
```

**Key rules**:
- ✅ **Downward dependencies only**: Higher layers can import lower layers
- ❌ **No upward dependencies**: Lower layers MUST NOT import higher layers
- ❌ **No peer dependencies**: Packages at the same level MUST NOT import each other
- ✅ **shared/* packages**: Can be imported by any package, but import nothing from internal/

**Examples**:
- `manager` can import `sdk`, `plugin`, `env`, `pathmeta`
- `sdk` can import `plugin`, `env`, but NOT `manager`
- `plugin` can import `env`, but NOT `sdk` or `manager`
- `env` cannot import `sdk`, `plugin`, or `manager`
- `pathmeta` can only import `shared/*`

## Environment Variables

- `VFOX_HOME`: User home directory (default: `~/.vfox`)
- `VFOX_ROOT`: Shared installation root (default: platform-specific)
- `VFOX_PLUGIN`: Custom plugin path
- `VFOX_CACHE`: Custom cache location
- `VFOX_TEMP`: Custom temp directory

## Configuration Files

vfox uses TOML format for version configuration with two supported file names:

- **Project config**: `.vfox.toml` (priority 1) or `vfox.toml` (priority 2) in project root
- **Global config**: `~/.vfox/.vfox.toml`
- **Session config**: `~/.vfox/tmp/{pid}/.vfox.toml`
- **User config**: `~/.vfox/config.yaml` (vfox settings, not SDK versions)
- **Legacy support**: `.tool-versions`, `.nvmrc`, `.node-version`, `.sdkmanrc` (auto-detected and migrated)

### TOML Configuration Format

**Simple format** (most common):
```toml
[tools]
nodejs = "21.5.1"
python = "3.11.0"
go = "1.21.5"
```

**Complex format** (with additional attributes):
```toml
[tools]
java = { version = "21", vendor = "openjdk" }
go = { version = "1.21.5", experimental = false }
```

### File Priority

When multiple config files exist in a directory:
1. `.vfox.toml` has priority over `vfox.toml`
2. `.vfox.toml` or `vfox.toml` have priority over legacy `.tool-versions`
3. If `.tool-versions` is read, it's automatically migrated to `.vfox.toml` (original file preserved)

## Plugin Development

Plugins are Lua scripts that must implement specific functions:
- `Available()`: List available versions
- `PreInstall(version)`: Prepare installation
- `PostInstall(version)`: Finalize installation
- `EnvKeys(version)`: Return environment variables

Test plugins locally before publishing to the registry.

## Key Files for Understanding

1. **`internal/manager.go`**: Core orchestration and SDK lifecycle
2. **`internal/pathmeta/path_meta.go`**: Path architecture and dual-path design
3. **`internal/pathmeta/vfox_toml.go`**: TOML configuration file format and parsing
4. **`internal/pathmeta/config_loader.go`**: Config file loading with priority and migration
5. **`internal/sdk/sdk.go`**: SDK abstraction and operations
6. **`internal/plugin/plugin.go`**: Plugin system and Lua integration
7. **`cmd/commands/activate.go`**: Shell activation and hook setup
8. **`internal/env/manager.go`**: Environment variable management

## Testing

**Testing Philosophy**: Every implementation must have corresponding unit tests.

### Testing Requirements
- **Coverage requirement**: All new code must have unit test coverage
- **Test file naming**: Tests for `internal/pkg/file.go` should be in `internal/pkg/file_test.go`
- **Run tests before committing**: `go test ./...`
- **Run tests with coverage**: `go test ./... -coverprofile=coverage.out -covermode=atomic`
- **Run tests for specific package**: `go test ./internal/sdk -v`

### Testing Focus Areas
- Cross-platform compatibility (macOS, Linux, Windows)
- Shell integration across different shells (Bash, Zsh, Fish, PowerShell, Clink, Nushell)
- Symlink/shim behavior on respective platforms
- Configuration file loading and migration (TOML, legacy formats)
- SDK installation and version resolution

### Example Test Structure
```go
// internal/sdk/sdk_test.go
package sdk

import (
    "testing"
)

func TestSdkInstall(t *testing.T) {
    // Test implementation
}
```

## Release Process

- Versioning follows semantic versioning
- Uses `goreleaser` for releases (configured in `.goreleaser.yaml`)
- Builds for multiple platforms and architectures
- Automatic release to GitHub, Homebrew, and package managers
