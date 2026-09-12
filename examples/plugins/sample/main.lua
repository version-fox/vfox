-- A small plugin for exercising the core test runner; no real SDK is downloaded.
local http = require("http")
local json = require("json")

PLUGIN = {
    name = "sample",
    version = "0.0.1",
    description = "Plugin testing example",
}

function PLUGIN:Available(ctx)
    local response, err = http.get({ url = "https://example.com/releases" })
    if err then error(err) end
    assert(response.status_code == 200, "HTTP " .. response.status_code)

    local data = assert(json.decode(response.body))
    local versions = {}
    for _, version in ipairs(data.versions) do
        table.insert(versions, { version = version })
    end
    assert(#versions > 0, "No versions found")
    return versions
end

function PLUGIN:PreInstall(ctx)
    local version = ctx.version
    if version == "latest" then
        version = self:Available({})[1].version
    end
    assert(version:match("^%d+%.%d+%.%d+$"), "Invalid version: " .. version)
    local mirror = os.getenv("VFOX_SAMPLE_MIRROR") or "https://example.com"
    return {
        version = version,
        url = mirror .. "/sample-" .. version .. "-" .. RUNTIME.osType .. "-" .. RUNTIME.archType .. ".zip",
    }
end

function PLUGIN:EnvKeys(ctx)
    return { { key = "PATH", value = ctx.main.path .. "/bin" } }
end
