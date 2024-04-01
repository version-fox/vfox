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
