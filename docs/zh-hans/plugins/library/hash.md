# Hash 标准库

`hash` 库用于在 Lua 插件中计算文件摘要并校验下载文件。

**使用**

```lua
local hash = require("hash")

local file = "/path/to/sdk.tar.gz"
local expected = "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"

local actual, err = hash.sha256_file(file)
assert(err == nil, err)

local ok, err = hash.verify_sha256(file, expected)
assert(err == nil, err)
assert(ok == true)
```

**函数**

| 函数 | 说明 |
| :--- | :--- |
| `hash.sha256_file(path)` | 返回文件的 SHA-256 摘要。 |
| `hash.sha512_file(path)` | 返回文件的 SHA-512 摘要。 |
| `hash.sha1_file(path)` | 返回文件的 SHA-1 摘要。 |
| `hash.md5_file(path)` | 返回文件的 MD5 摘要。 |
| `hash.sum_file(path, algorithm)` | 使用 `sha256`、`sha512`、`sha1` 或 `md5` 返回文件摘要。 |
| `hash.verify_sha256(path, expected)` | 校验文件 SHA-256 是否与 `expected` 匹配。 |
| `hash.verify_sha512(path, expected)` | 校验文件 SHA-512 是否与 `expected` 匹配。 |
| `hash.verify_sha1(path, expected)` | 校验文件 SHA-1 是否与 `expected` 匹配。 |
| `hash.verify_md5(path, expected)` | 校验文件 MD5 是否与 `expected` 匹配。 |
| `hash.verify_file(path, expected, algorithm)` | 使用 `sha256`、`sha512`、`sha1` 或 `md5` 校验文件摘要是否与 `expected` 匹配。 |
