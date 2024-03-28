# Strings Library

`vfox` provides some utils for string manipulation. In the Lua script, you can use `require("vfox.strings")` to access it.
For example:

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