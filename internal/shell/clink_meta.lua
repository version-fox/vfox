---@meta
-- Type definitions for lua-language-server (the lint job in ci.yml). Clink adds
-- these functions to Lua's os table; clink_vfox.lua uses them. Nothing loads this
-- file. https://chrisant996.github.io/clink/clink.html#os.setenv

---Sets the environment variable `name` to `value`, or removes it when `value` is nil.
---@param name string
---@param value string?
---@return boolean ok
function os.setenv(name, value) end

---Returns the ID of the Clink process.
---@return integer pid
function os.getpid() end
