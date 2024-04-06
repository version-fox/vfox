# Archiver Library

`vfox` provides a decompression tool that supports `tar.gz`, `tgz`, `tar.xz`, `zip`, and `7z`. In Lua scripts, you can
use `require("vfox.archiver")` to access it.

**Usage**

```shell
local archiver = require("vfox.archiver")
local err = archiver.decompress("testdata/test.zip", "testdata/test")
```