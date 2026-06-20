# File Library

`vfox` exposes basic path utilities to Lua plugins. Use `require("file")` to access them. Every API works with both files and directories without exposing file-content reading or writing.

## Copy, remove, and move paths

```lua
local file = require("file")

file.copy(source_path, destination_path)
file.move(source_path, destination_path)
file.remove(path)
```

## Create a symbolic link

```lua
local file = require("file")
file.symlink(source_path, link_path)
```

Paths may be absolute or relative to the current `vfox` process. Every operation returns `true` on success and raises a Lua error on failure. Directory copies and removals are recursive.
