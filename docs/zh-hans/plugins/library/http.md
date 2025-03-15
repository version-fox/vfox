# Http 标准库

`vfox`提供了一个简单的 http 能力，当前支持`Get`、`Head`两种请求类型，以及文件下载。

**使用**

```lua
local http = require("http")
--- get 请求, 不要用此请求进行文件下载!!!
local resp, err = http.get({
    url = "https://httpbin.org/json",
    headers = {
      ['Host'] = "localhost"
    }
})
--- 返回参数
assert(err == nil)
assert(resp.status_code == 200)
assert(resp.headers['Content-Type'] == 'application/json')
assert(resp.body == 'xxxxxxxx')

--- head 请求
resp, err = http.head({
    url = "https://httpbin.org/json",
    headers = {
      ['Host'] = "localhost"
    }
})
assert(err == nil)
assert(resp.status_code == 200)
assert(resp.content_length ~= 0)

--- 下载文件, vfox >= 0.4.0
err = http.download_file({
    url = "https://version-fox.github.io/vfox-plugins/index.json",
    headers = {}
}, "/usr/local/file")
assert(err == nil, [[must be nil]] )

```
