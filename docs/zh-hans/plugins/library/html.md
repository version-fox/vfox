# Html标准库

`vfox`提供的`html`库是基于[goquery](https://github.com/PuerkitoBio/goquery)实现的。


**使用**
```shell
local html = require("html")
local doc = html.parse("<html><body><div id='t2' name='123'>456</div><div>222</div></body></html>")
local s = doc:find("div"):eq(1)
local f = doc:find("div"):eq(0)
local ss = doc:find("div"):eq(2)
print(ss:text() == "")
assert(s:text() == "222")	
assert(f:text() == "456")
```
