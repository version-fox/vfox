-- Last Modification: 2023-12-10
-- Description: Java script for version manager
-- Author: Lihan
-- Api doc: https://api.adoptium.net/q/swagger-ui

local http = require("http")
local json = require("json")


SearchUrl = "https://api.adoptium.net/v3/assets/latest/%s/hotspot?os=%s&architecture=%s"
AvailableVersionsUrl = "https://api.adoptium.net/v3/info/available_releases"

function download_url(ctx)
    os_type = ctx.os_type
    arch_type = ctx.arch_type
    version = ctx.version
    http.get({
        url = url,
        headers = {
            ["Accept"] = "application/vnd.github.v3+json",
        },

    })

    if arch_type == "amd64" then
        arch_type = "x64"
    end
    return string.format(DownloadUrl, version, version, os_type, arch_type, file_ext(ctx))
end

function search(ctx)
    os_type = ctx.os_type
    arch_type = ctx.arch_type
    version = ctx.version
    if arch_type == "amd64" then
        arch_type = "x64"
    end
    if os_type == 'darwin' then
        os_type = 'mac'
    end
    local url = string.format(SearchUrl, version, os_type, arch_type)
    local resp, errMsg = http.get({ url = url })
    if errMsg ~= nil then
        print("Error: " .. errMsg)
        return nil, errMsg
    end
    local jsonBody = json.decode(resp.body)
    local result = {}
    for k, v in pairs(jsonBody) do
        result[k] = v.release_name
    end
    return result
end

function name()
    return "java"
end

function env_keys(ctx)
    version_path = ctx.version_path
    return {
        {
            key = "PATH",
            value = version_path .. "/bin"
        }
    }
end