--- Parse the legacy file found by vfox to determine the version of the tool.
--- Useful to extract version numbers from files like JavaScript's package.json or Golangs go.mod.
function PLUGIN:ParseLegacyFile(ctx)
    printTable(ctx)
    local filename = ctx.filename
    local filepath = ctx.filepath

    installed = ctx.getInstalledVersions()
    if #installed > 0 then
        print("Installed: " .. installed[1])
        return {
            version = "check-installed"
        }
    end

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