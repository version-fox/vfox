# Hash Library

The `hash` library calculates file digests and verifies downloaded files in Lua plugins.

**Usage**

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

**Functions**

| Function | Description |
| :------- | :---------- |
| `hash.sha256_file(path)` | Returns the SHA-256 digest of a file. |
| `hash.sha512_file(path)` | Returns the SHA-512 digest of a file. |
| `hash.sha1_file(path)` | Returns the SHA-1 digest of a file. |
| `hash.md5_file(path)` | Returns the MD5 digest of a file. |
| `hash.sum_file(path, algorithm)` | Returns the digest of a file using `sha256`, `sha512`, `sha1`, or `md5`. |
| `hash.verify_sha256(path, expected)` | Returns whether the file SHA-256 matches `expected`. |
| `hash.verify_sha512(path, expected)` | Returns whether the file SHA-512 matches `expected`. |
| `hash.verify_sha1(path, expected)` | Returns whether the file SHA-1 matches `expected`. |
| `hash.verify_md5(path, expected)` | Returns whether the file MD5 matches `expected`. |
| `hash.verify_file(path, expected, algorithm)` | Returns whether the file digest matches `expected` using `sha256`, `sha512`, `sha1`, or `md5`. |
