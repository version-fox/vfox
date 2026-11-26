# Cache Library

`vfox` provides a small persistent key/value cache for plugins. In Lua scripts, use `require("vfox.cache")` to access it.

Values are stored as JSON, so a value can be a string, number, boolean or a table of those. Setting `nil` stores `null`, which reads back as `nil`, like a missing key. The cache is saved to the
file you give to `new` after every change, and survives between runs. Each write goes to a temporary file that is renamed over the cache file, so a reader never sees a partly written file. On Unix the file is `0600`; Windows ignores the mode bits.
Two processes that write at the same moment can overwrite each other's change: the last write wins.

**Usage**

```lua
local cache = require("vfox.cache")

local c, err = cache.new("/path/to/cache-file")

-- the third argument is the time to live in seconds, -1 (the default) never expires
c.set("key", { "a", "b" }, 3600)

local value = c.get("key") -- nil when the key is missing or expired

c.remove("key")
```

**API**

| Function                 | Description                                                                                         |
|--------------------------|-----------------------------------------------------------------------------------------------------|
| `cache.new(path)`        | Opens or creates the cache stored in the file at `path`. Returns the cache, or `nil` and an error.   |
| `c.set(key, value, ttl)` | Stores `value`. `ttl` is in seconds (fractions allowed), `-1` never expires. Returns `true`, or `nil` and an error.     |
| `c.get(key)`             | Returns the value, or `nil` when the key is missing or expired.                                     |
| `c.remove(key)`          | Removes the key. Returns `true`, or `false` and an error.                                           |
