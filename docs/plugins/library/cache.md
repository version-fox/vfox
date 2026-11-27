# Cache Library

`vfox` provides a small persistent key/value cache for plugins. In Lua scripts, use `require("vfox.cache")` to access it.

Values are stored as JSON, so a value can be a string, a finite number, a boolean or a table of those. Keys are strings (a number is converted to its string form). Setting `nil` stores `null`, which reads back as `nil`, like a missing key. The cache is saved to the
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

| Function                 | Description                                                                                                                                                                                              |
|--------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `cache.new(path)`        | Opens or creates the cache stored in the file at `path`. Returns the cache, or `nil` and an error.                                                                                                        |
| `c.set(key, value, ttl)` | Stores `value`. `ttl` is in seconds (fractions allowed): `-1` never expires, `0` expires at once, and the expiry cannot be later than the year 2262, so the largest ttl for an expiring entry shrinks by one second every second (the exact current value is in the error below). Returns `true`, or `nil` and an error. A ttl outside that range, a negative ttl other than `-1` or `NaN` raises an error: `expire time must be -1 (never expires) or a number of seconds from 0 up to <largest ttl>`. If the system clock is before 1970 or within a second of April 2262, no expiring entry can be stored and the error says the clock is outside the supported range instead; `-1` is not affected. A ttl that is not a number raises a type error. |
| `c.get(key)`             | Returns the value, or `nil` when the key is missing or expired. Returns `nil` and an error when the cache file cannot be read.                                                                            |
| `c.remove(key)`          | Removes the key. Returns `true`, or `false` and an error.                                                                                                                                                 |

**Errors**

A cache file that cannot be decoded (for example one that is empty or damaged) is reported and never overwritten. `cache.new` returns `nil` and `read cache "<path>": <reason>`, and `c.get`, `c.set` and `c.remove` return the same message if the file goes bad after `new`. Other failures, such as a rename that fails on Windows because another process holds the file open, are returned the same way.

**Limits**

`c.set` returns `nil` and an error message instead of storing the value when it is not allowed:

- A table nested more than 256 levels deep is refused.
- The JSON written for one value is limited to 64 MiB. The size is estimated conservatively from the encoded form, never below what is written: escaped characters such as `<`, `>` and `&` (six bytes each) and long keys count at their escaped size, and a table that is referenced several times counts once per reference. The estimate can be slightly above the real size (about one byte per array element, and some control characters), so a value just under 64 MiB can be refused.
- A table that is referenced more than once is written once per reference. The values written inside the second and later references are limited to 1,048,576 (the table itself is not counted, only what it contains). A two-element table has two values, so a list of 524,289 references to it is stored (524,288 repeats, 1,048,576 values) and 524,290 are refused, with only about 3 MB of JSON.
- Strings, table keys and the `key` argument must be valid UTF-8.
- A number must be finite: `NaN` and infinity are refused.
- Functions, userdata and threads, recursively nested tables, sparse arrays and tables that mix array and string keys are refused.

`json.encode` has no size limit for ordinary data. Its limits apply only to tables referenced more than once, plus a depth limit:

- Inside the second and later references to the same table, it applies the same 1,048,576-value limit as `c.set`, and also a separate 64 MiB limit on the estimated bytes written inside those repeats. Everything outside repeated references is free, so data without shared tables is never limited. This is different from `c.set`, where the 64 MiB limit covers the whole value.
- A table nested more than 10,000 levels deep makes it return `nil` and an error (`exceeded max depth`, the nesting limit of Go's `encoding/json`). Tables nested far deeper than that are not handled well: encoding can take very long or crash the process with a stack overflow. This is unchanged from before the cache module existed.
