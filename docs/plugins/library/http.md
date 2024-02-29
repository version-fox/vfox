# HTTP Library

`vfox` provides a simple HTTP library, currently supporting only GET and HEAD requests. In the Lua script, you can
use `require("http")` to access it. For example:

**Usage**
```shell
local http = require("http")
assert(type(http) == "table")
assert(type(http.get) == "function")
local resp, err = http.get({
    url = "http://ip.jsontest.com/",
    headers = {
      ['Host'] = "localhost"
    }
})
assert(err == nil)
assert(resp.status_code == 200)
assert(resp.headers['Content-Type'] == 'application/json')
assert(resp.body == '{"ip": "xxx.xxx.xxx.xxx"}')
```