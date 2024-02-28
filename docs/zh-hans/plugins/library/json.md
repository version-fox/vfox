# Json标准库

`vfox`提供的`json`库是基于[gopher-json](https://github.com/PuerkitoBio/goquery)实现的。


**使用**
```shell
local json = require("json")

local obj = { "a", 1, "b", 2, "c", 3 }
local jsonStr = json.encode(obj)
local jsonObj = json.decode(jsonStr)
for i = 1, #obj do
    assert(obj[i] == jsonObj[i])
end
```
