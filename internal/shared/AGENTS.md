# PROJECT KNOWLEDGE BASE

**Generated:** 2026-01-17
**Commit:** v0.x
**Project:** vfox (Version Fox) - Cross-platform SDK version manager
**Scope:** internal/shared/ - Shared utility packages

## OVERVIEW
Shared utility packages for common operations (no business logic)

## WHERE TO LOOK
| Task | Location | Notes |
|------|----------|-------|
| **Download utilities** | `util/downloader.go` | HTTP download with progress bar |
| **Decompression** | `util/decompressor.go` | tar.gz, zip, tar.xz extraction |
| **File operations** | `util/file.go` | File/directory utilities |
| **Version parsing** | `util/version.go` | Semantic version comparison |
| **Checksums** | `checksum.go` (root) | SHA256/512/MD5/SHA1 verification (non-standard) |
| **Logging** | `logger/logger.go` | Debug/Info/Warn/Error levels |
| **Caching** | `cache/cache.go` | FileCache with expiration, thread-safe |
| **Shim executables** | `shim/shim.go` | Windows shim generation |
| **Interactive UI** | `printer/select.go` | PageKVSelect with fuzzy search |
| **Clipboard** | `util/clipboard.go` | Cross-platform clipboard access |
| **CI detection** | `util/ci.go` | Detect CI environments |
| **TTY detection** | `util/tty.go` | Check if terminal is TTY |

## CONVENTIONS

### Import Rules
- **✅ Can be imported**: By any `internal/` package
- **❌ Cannot import**: Anything from `internal/` (only stdlib and external deps)
- **Purpose**: Utilities only - NO business logic

### Package Structure
- Most utilities in `util/` subdirectory
- `checksum.go` in root is non-standard (mixed package structure) - should be in `util/`
- Each subdirectory is a separate Go package (util, logger, cache, shim, printer)

### Thread Safety
- `cache/` uses `sync.RWMutex` for concurrent access
- Logger is not thread-safe for level changes (SetLevel not mutex-protected)

## ANTI-PATTERNS

### Architectural Violations
1. **Business logic in shared packages**: Should only contain pure utilities
2. **Imports from internal/**: shared packages must not import internal/ packages
3. **Mixed package structures**: checksum.go in root (should be in util/)

### Code Quality
1. **No side effects**: Utilities should be pure functions where possible
2. **No global state**: Avoid mutable globals (except logger level)
3. **No hidden dependencies**: Clear input/output contracts
