# HTTP Library

`vfox` provides a simple HTTP library, currently supporting only GET and HEAD requests and download file. In the Lua script, you can
use `require("http")` to access it. For example:

**Usage**

```lua
local http = require("http")
--- get request, do not use this request to download files!!!
local resp, err = http.get({
    url = "https://httpbin.org/json",
    headers = {
      ['Host'] = "localhost"
    }
})
--- return parameters
assert(err == nil)
assert(resp.status_code == 200)
assert(resp.headers['Content-Type'] == 'application/json')
assert(resp.body == 'xxxxx')

--- head request
resp, err = http.head({
    url = "https://httpbin.org/json",
    headers = {
      ['Host'] = "localhost"
    }
})
assert(err == nil)
assert(resp.status_code == 200)
assert(resp.content_length ~= 0)

--- Download file, vfox >= 0.4.0
err = http.download_file({
    url = "https://version-fox.github.io/vfox-plugins/index.json",
    headers = {}
}, "/usr/local/file")
assert(err == nil, [[must be nil]] )

```
