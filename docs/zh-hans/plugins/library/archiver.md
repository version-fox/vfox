# Archiver 标准库

`vfox` 提供了解压工具, 支持`tar.gz`、`tgz`、`tar.xz`、`zip`、`7z`。在Lua脚本中，你可以使用`require("vfox.archiver")`来访问它。
例如：

**Usage**
```shell
local archiver = require("vfox.archiver")
local err = archiver.decompress("testdata/test.zip", "testdata/test")
```