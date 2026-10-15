--- Post-switch hook, called after `vfox use` (and `vfox reshim`) has updated
--- scope links. Plugins may sync version-independent shims here.
function PLUGIN:PostUse(ctx)
    local version = ctx.version
    local previousVersion = ctx.previousVersion
    local scope = ctx.scope
    local shimDir = ctx.shimDir
    local main = ctx.main
    local sdkInfo = ctx.sdkInfo
    assert(version ~= nil, "PostUse ctx.version missing")
    assert(scope ~= nil, "PostUse ctx.scope missing")
    assert(shimDir ~= nil and shimDir ~= "", "PostUse ctx.shimDir missing")
    assert(main ~= nil, "PostUse ctx.main missing")
end
