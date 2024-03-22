--- !!! DO NOT EDIT OR RENAME !!!
PLUGIN = {}

--- !!! MUST BE SET !!!
--- Plugin name
PLUGIN.name = "java"
--- Plugin version
PLUGIN.version = "0.0.1"
-- Update URL, will deprecated in the future
---@deprecated
PLUGIN.updateUrl = "{URL}/sdk.lua"
-- Repository URL
PLUGIN.repository = "https://github.com/version-fox/vfox-plugin-template"

PLUGIN.notes = {
    "some notes",
    "some notes",
}

-- Some preset configurations
PLUGIN.presets = {
    "nodejs",
    "tinghua",
    "npmmirror"
}

--- !!! OPTIONAL !!!
--- Plugin description
PLUGIN.description = "xxx"
-- minimum compatible vfox version
PLUGIN.minRuntimeVersion = "0.2.2"
