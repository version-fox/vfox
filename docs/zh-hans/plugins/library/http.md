# Http标准库

`vfox`提供了一个简单的http能力，当前支持`Get`、`Head`两种请求类型。


**使用**
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