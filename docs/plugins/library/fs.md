# FS Library

`vfox` exposes basic path utilities to Lua plugins. Use `require("fs")` to access them. Every API works with both files and directories without exposing file-content reading or writing.

## Copy, remove, and move paths

```lua
local fs = require("fs")

fs.copy(source_path, destination_path)
fs.move(source_path, destination_path)
fs.remove(path)
```

## Create a symbolic link

```lua
local fs = require("fs")
fs.symlink(source_path, link_path)
```

Paths may be absolute or relative to the current `vfox` process. Every operation returns `true` on success and raises a Lua error on failure. Directory copies and removals are recursive. Like `mv`, `move` renames when its destination is a new path and moves the source under its original basename when the destination is an existing directory.

Directory copies preserve symbolic links, including their stored targets, without traversing them. The source path passed to `symlink` is resolved relative to the current process directory, even when the link is created in a subdirectory.
