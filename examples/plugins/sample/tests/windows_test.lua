local test = require("vfox.test")
local plugin = test.load({
    os = "windows",
    arch = "amd64",
    env = { VFOX_SAMPLE_MIRROR = "https://mirror.example.com" },
})

local archive = plugin:PreInstall({ version = "1.2.3" })
assert(archive.url == "https://mirror.example.com/sample-1.2.3-windows-amd64.zip")
local keys = plugin:EnvKeys({ main = { path = "sdk-directory" } })
assert(keys[1].key == "PATH" and keys[1].value == "sdk-directory/bin")

-- No HTTP handler: an accidental network request fails in the default offline mode.
local ok, err = pcall(function() plugin:Available({}) end)
assert(not ok and tostring(err):find("HTTP is disabled offline", 1, true))
