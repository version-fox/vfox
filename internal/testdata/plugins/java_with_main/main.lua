--- Common libraries provided by VersionFox (optional)
local http = require("http")
local json = require("json")
local html = require("html")

--- The following two parameters are injected by VersionFox at runtime
--- Operating system type at runtime (Windows, Linux, Darwin)
RUNTIME = {
    --- Operating system type at runtime (Windows, Linux, Darwin)
    osType = "",
    --- Operating system architecture at runtime (amd64, arm64, etc.)
    archType = "",
    --- vfox runtime version
    version = "",
}

PLUGIN = {
    --- Plugin name
    name = "java",
    --- Plugin author
    author = "Lihan",
    --- Plugin version
    version = "0.0.1",
    --- Plugin description
    description = "xxx",
    -- Update URL
    updateUrl = "{URL}/sdk.lua",
    -- minimum compatible vfox version
    minRuntimeVersion = "0.2.2",
    legacyFilenames = {
        ".node-version",
        ".nvmrc"
    }
}

--- Returns some pre-installed information, such as version number, download address, local files, etc.
--- If checksum is provided, vfox will automatically check it for you.
--- @param ctx table
--- @field ctx.version string User-input version
--- @return table Version information
function PLUGIN:PreInstall(ctx)
    print(json.encode(RUNTIME))
    local version = ctx.version
    return {
        --- Version number
        version = "version",
        --- remote URL or local file path [optional]
        url = "xxx",
        --- SHA256 checksum [optional]
        sha256 = "xxx",
        --- md5 checksum [optional]
        md5 = "xxx",
        --- sha1 checksum [optional]
        sha1 = "xxx",
        --- sha512 checksum [optional]
        sha512 = "xx",
        --- additional need files [optional]
        addition = {
            {
                --- additional file name !
                name = "xxx",
                --- remote URL or local file path [optional]
                url = "xxx",
                --- SHA256 checksum [optional]
                sha256 = "xxx",
                --- md5 checksum [optional]
                md5 = "xxx",
                --- sha1 checksum [optional]
                sha1 = "xxx",
                --- sha512 checksum [optional]
                sha512 = "xx",
            }
        }
    }
end

--- Extension point, called after PreInstall, can perform additional operations,
--- such as file operations for the SDK installation directory or compile source code
--- Currently can be left unimplemented!
function PLUGIN:PostInstall(ctx)
    --- ctx.rootPath SDK installation directory
    local rootPath = ctx.rootPath
    local sdkInfo = ctx.sdkInfo['sdk-name']
    local path = sdkInfo.path
    local version = sdkInfo.version
    local name = sdkInfo.name
end

--- Return all available versions provided by this plugin
--- @param ctx table Empty table used as context, for future extension
--- @return table Descriptions of available versions and accompanying tool descriptions
function PLUGIN:Available(ctx)
    return {
        {
            version = "xxxx",
            note = "LTS",
            addition = {
                {
                    name = "npm",
                    version = "8.8.8",
                }
            }
        }
    }
end

--- Each SDK may have different environment variable configurations.
--- This allows plugins to define custom environment variables (including PATH settings)
--- Note: Be sure to distinguish between environment variable settings for different platforms!
--- @param ctx table Context information
--- @field ctx.path string SDK installation directory
function PLUGIN:EnvKeys(ctx)
    --- this variable is same as ctx.sdkInfo['plugin-name'].path
    local mainPath = ctx.path
    local sdkInfo = ctx.sdkInfo['sdk-name']
    local path = sdkInfo.path
    local version = sdkInfo.version
    local name = sdkInfo.name
    return {
        {
            key = "JAVA_HOME",
            value = mainPath
        },
        {
            key = "PATH",
            value = mainPath .. "/bin"
        },
        {
            key = "PATH",
            value = mainPath .. "/bin2"
        }
    }
end

--- When user invoke `use` command, this function will be called to get the
--- valid version information.
--- @param ctx table Context information
function PLUGIN:PreUse(ctx)
    --- user input version
    local version = ctx.version
    --- installed sdks
    local sdkInfo = ctx.installedSdks['xxxx']
    local path = sdkInfo.path
    local name = sdkInfo.name
    local sdkVersion = sdkInfo.version

    --- working directory
    local cwd = ctx.cwd

    printTable(ctx)

    --- user input scope
    local scope = ctx.scope

    if (scope == "global") then
        print("return 9.9.9")
        return {
            version = "9.9.9",
        }
    end

    if (scope == "project") then
        print("return 10.0.0")
        return {
            version = "10.0.0",
        }
    end

    print("return 1.0.0")

    return {
        version = "1.0.0"
    }
end

function PLUGIN:ParseLegacyFile(ctx)
    printTable(ctx)
    local filename = ctx.filename
    local filepath = ctx.filepath
    if filename == ".node-version" then
        return {
            version = "14.17.0"
        }
    else
        return {
            version = "0.0.1"
        }
    end

end
