local util = require("util")
--- Returns some pre-installed information, such as version number, download address, local files, etc.
--- If checksum is provided, vfox will automatically check it for you.
--- @param ctx table Context; ctx.version is the user-input version
--- @return table Version information
function PLUGIN:PreInstall(ctx)
    return util:PreInstall(ctx)
end
