# Strings 标准库

`vfox` 提供了一些字符串操作的工具。在Lua脚本中，你可以使用`require("vfox.strings")`来访问它。
例如：

**Usage**
```shell
local strings = require("vfox.strings")
local str_parts = strings.split("hello world", " ")
print(str_parts[1]) -- hello

assert(strings.has_prefix("hello world", "hello"), [[not strings.has_prefix("hello")]])
assert(strings.has_suffix("hello world", "world"), [[not strings.has_suffix("world")]])
assert(strings.trim("hello world", "world") == "hello ", "strings.trim()")
assert(strings.contains("hello world", "hello ") == true, "strings.contains()")

got = strings.trim_space(tt.input)

local str = strings.join({"1",3,"4"},";")
assert(str == "1;3;4", "strings.join()")
```