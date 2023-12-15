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

function PLUGIN:InstallInfo(ctx)
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
        additional = {
            {
                name = "npm",
                version = "7.24.0",
                url = "https://github.com/npm/cli/archive/v7.24.0.tar.gz",
            }
        }
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
            note = v.lts and "LTS" or ""
        })
    end
    return result
end

function PLUGIN:EnvKeys(ctx)
    local version_path = ctx.path
    local npm_path = ctx.additional_path['npm']
    return {
        {
            key = "PATH",
            value = version_path .. "/bin"
        },
        {
            key = "PATH",
            value = npm_path .. "/bin"
        }
    }
end
