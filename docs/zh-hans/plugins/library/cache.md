# Cache 标准库

`vfox` 为插件提供了一个简单的持久化键值缓存。在 Lua 脚本中，你可以使用 `require("vfox.cache")` 来访问它。

值以 JSON 形式保存，因此可以是字符串、数字、布尔值或由它们组成的 table。每次修改后缓存都会写入传给 `new` 的文件，
因此在多次运行之间依然有效。

**Usage**

```lua
local cache = require("vfox.cache")

local c, err = cache.new("/path/to/cache-file")

-- 第三个参数是以秒为单位的过期时间，-1（默认值）表示永不过期
c.set("key", { "a", "b" }, 3600)

local value = c.get("key") -- key 不存在或已过期时返回 nil

c.remove("key")
```
