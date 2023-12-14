---  Default global variable
---  OS_TYPE:  windows, linux, darwin
---  ARCH_TYPE: 386, amd64, arm, arm64  ...
local http = require("http")
local json = require("json")

OS_TYPE = ""
ARCH_TYPE = ""

nodeDownloadUrl = "https://nodejs.org/dist/v%s/node-v%s-%s-%s%s"

VersionSourceUrl = "https://nodejs.org/dist/index.json"

PLUGIN = {
    name = "node",
    author = "Lihan",
    version = "0.0.1",
    updateUrl = "https://raw.githubusercontent.com/aooohan/ktorm-generator/main/build.gradle.lua",
}

function PLUGIN:DownloadUrl(ctx)
    local version = ctx.version

    local arch_type = ARCH_TYPE
    local ext = ".tar.gz"
    if arch_type == "amd64" then
        arch_type = "x64"
    end
    if OS_TYPE == "windows" then
        ext = ".zip"
    end
    return string.format(nodeDownloadUrl, version, version, OS_TYPE, arch_type, ext)
end

function PLUGIN:Available(ctx)
    local resp, err = http.get({
        url = VersionSourceUrl
    })
    if err ~= nil or resp.status_code ~= 200 then
        return {}
    end
    local body = json.decode(resp.body)
    local result = {}
    for _, v in ipairs(body) do
        table.insert(result, {
            version = string.gsub(v.version, "^v", ""),
            note = v.lts and "LTS" or ""
        })
    end
    return result
end

function PLUGIN:EnvKeys(ctx)
    version_path = ctx.version_path
    return {
        {
            key = "PATH",
            value = version_path .. "/bin"
        }
    }
end
