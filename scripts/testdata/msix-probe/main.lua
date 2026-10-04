-- A local SDK fixture for MSIX integration tests. No network access is needed.
PLUGIN = {
    name = "msix-probe",
    version = "0.0.1",
    description = "MSIX environment integration test fixture",
}

function PLUGIN:Available(ctx)
    return { { version = "2.0.0" }, { version = "1.0.0" } }
end

function PLUGIN:PreInstall(ctx)
    return { version = ctx.version }
end

function PLUGIN:PostInstall(ctx)
    local sdk = ctx.sdkInfo["msix-probe"]
    local marker = assert(io.open(sdk.path .. "/probe.txt", "w"))
    marker:write(sdk.version)
    marker:close()
end

function PLUGIN:EnvKeys(ctx)
    return {
        { key = "PATH", value = ctx.main.path },
        { key = "VFOX_MSIX_E2E_HOME", value = ctx.main.path },
        { key = "VFOX_MSIX_E2E_VERSION", value = ctx.main.version },
    }
end
