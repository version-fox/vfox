local test = require("vfox.test")
local status = 200
local requests = 0

local plugin = test.load({
    os = "linux",
    arch = "amd64",
    http = function(request)
        requests = requests + 1
        assert(request.method == "GET")
        assert(request.url == "https://example.com/releases")

        -- Use the plugin directory so fixtures work from any working directory.
        local file = assert(io.open(RUNTIME.pluginDirPath .. "/tests/fixtures/releases.json", "r"))
        local body = file:read("*a")
        assert(file:close())
        return { status_code = status, body = body }
    end,
})

local versions = plugin:Available({})
assert(#versions == 2)
assert(versions[1].version == "1.2.3")
assert(versions[2].version == "1.2.2")

local archive = plugin:PreInstall({ version = "latest" })
assert(archive.version == "1.2.3")
assert(archive.url == "https://example.com/sample-1.2.3-linux-amd64.zip")
assert(requests == 2)

status = 503
local ok, err = pcall(function() plugin:Available({}) end)
assert(not ok and tostring(err):find("HTTP 503", 1, true))

status = 200
assert(plugin:Available({})[1].version == "1.2.3")
