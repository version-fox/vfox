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

**Limits**

`c.set` returns `nil` and an error message instead of storing the value when it is too big or too deep:

- A table nested more than 256 levels deep is refused.
- The JSON written for one value is limited to 64 MiB. The size is estimated conservatively from the encoded form, never below what is written: escaped characters such as `<`, `>` and `&` (six bytes each) and long keys count at their escaped size, and a table that is referenced several times counts once per reference. The estimate can be slightly above the real size (about one byte per array element, and some control characters), so a value just under 64 MiB can be refused.
- Strings and table keys must be valid UTF-8.

`json.encode` has no depth or size limit for ordinary data. The only limit on it is for tables that are referenced more than once: the values written for the repeated references are capped (about one million values or 64 MiB), so a small shared table cannot expand into an enormous output.
