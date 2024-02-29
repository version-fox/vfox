# HTML Library

The HTML library provided by VersionFox is based on [goquery](https://github.com/PuerkitoBio/goquery), with some
functionality encapsulated. You can use `require("html")` to access it, for example:


**Usage**
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
