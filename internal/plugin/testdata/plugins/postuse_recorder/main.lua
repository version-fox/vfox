--- Test plugin that records PostUse invocations for SDK-level tests.
--- The trace file path comes from VFOX_TEST_POSTUSE_TRACE.
PLUGIN = {
    name = "postuse_recorder",
    version = "0.0.1",
    description = "PostUse recorder for tests",
    author = "test",
    minRuntimeVersion = "0.0.0",
}

function PLUGIN:Available(ctx)
    return {}
end

function PLUGIN:PreInstall(ctx)
    return {}
end

function PLUGIN:EnvKeys(ctx)
    return {
        { key = "PATH", value = ctx.main.path .. "/bin" },
    }
end

function PLUGIN:PostUse(ctx)
    local trace = os.getenv("VFOX_TEST_POSTUSE_TRACE")
    assert(trace ~= nil and trace ~= "", "VFOX_TEST_POSTUSE_TRACE not set")
    local f = assert(io.open(trace, "a"))
    f:write(ctx.version .. "|" .. ctx.scope .. "|" .. ctx.previousVersion .. "|"
        .. ctx.main.name .. "|" .. ctx.main.version .. "|" .. ctx.shimDir .. "\n")
    f:close()
    local marker = assert(io.open(ctx.shimDir .. "/postuse-marker", "w"))
    marker:write(ctx.version)
    marker:close()
end
