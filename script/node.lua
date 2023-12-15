---  Default global variable
---  OS_TYPE:  windows, linux, darwin
---  ARCH_TYPE: 386, amd64, arm, arm64  ...
local http = require("http")
local json = require("json")

OS_TYPE = ""
ARCH_TYPE = ""

nodeDownloadUrl = "https://nodejs.org/dist/v%s/node-v%s-%s-%s%s"
npmDownloadUrl = "https://github.com/npm/cli/archive/v%s.%s"

VersionSourceUrl = "https://nodejs.org/dist/index.json"

PLUGIN = {
    name = "node",
    author = "Lihan",
    version = "0.0.1",
    updateUrl = "https://raw.githubusercontent.com/aooohan/ktorm-generator/main/build.gradle.lua",
}

function PLUGIN:PreInstall(ctx)
    local version = ctx.version

    local arch_type = ARCH_TYPE
    local ext = ".tar.gz"
    if arch_type == "amd64" then
        arch_type = "x64"
    end
    if OS_TYPE == "windows" then
        ext = ".zip"
    end
    local node_url = string.format(nodeDownloadUrl, version, version, OS_TYPE, arch_type, ext)
    --local npm_url = string.format(npmDownloadUrl, version, ext)
    return {
        version = version,
        url = node_url,
    }
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
            note = v.lts and "LTS" or "",
            additional = {
                {
                    name = "npm",
                    version = v.npm,
                }
            }
        })
    end
    return result
end

--- Expansion point
function PLUGIN:PostInstall(ctx)
    local rootPath = ctx.rootPath
    local sdkInfo = ctx.sdkInfo['node']
    local path = sdkInfo.path
    local version = sdkInfo.version
    local name = sdkInfo.name
end

function PLUGIN:EnvKeys(ctx)
    local version_path = ctx.path
    return {
        {
            key = "PATH",
            value = version_path .. "/bin"
        },
    }
end
