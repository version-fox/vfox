# JSON Library

Based on [gopher-json](https://github.com/layeh/gopher-json)

**Usage**
```lua
local json = require("json")

local obj = { "a", 1, "b", 2, "c", 3 }
local jsonStr = json.encode(obj)
local jsonObj = json.decode(jsonStr)
for i = 1, #obj do
    assert(obj[i] == jsonObj[i])
end
```
