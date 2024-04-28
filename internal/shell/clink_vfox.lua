-- https://chrisant996.github.io/clink/clink.html#extending-clink
local vfox_sdk_table = {}
clink.argmatcher('vfox'):nofiles():setdelayinit(function(vfox)
    if #vfox_sdk_table ~= 0 then
        return
    end

    local vfox_available = io.popen('vfox available')
    for line in vfox_available:lines() do
        local sdk, yes = line:gsub('%c%[%d+m', ''):match('^(%a+)%s+(%u%u%u?)')
        if sdk and yes and (yes == 'YES' or yes == 'NO') then
            table.insert(vfox_sdk_table, sdk)
        end
    end
    vfox_available:close()

    local function vfox_ls_func()
        local pre, ls = '', {}
        local vfox_ls = io.popen('vfox ls')
        for line in vfox_ls:lines() do
            local txt = line:gsub('%c%[%d+m', ''):match('^%A+(%a.+)')
            if txt then
                if txt:find('v') == 1 then
                    ls[pre] = true
                    table.insert(ls, pre .. '@' .. txt:sub(2))
                else
                    pre = txt
                    ls[pre] = false
                end
            end
        end
        vfox_ls:close()
        return ls
    end
    local function vfox_sdk_func()
        local ls, res = vfox_ls_func(), {}
        for k, v in pairs(ls) do
            if type(v) == 'boolean' then
                table.insert(res, k)
            end
        end
        return res
    end

    local vfox_sdk = clink.argmatcher():nofiles():addarg(vfox_sdk_func):addflags('--help', '-h')
    local vfox_use = clink.argmatcher():nofiles():addarg(function()
        local ls, res = vfox_ls_func(), {}
        for k, v in pairs(ls) do
            if v then
                table.insert(res, v == true and k or v)
            end
        end
        return res
    end):addflags('--global', '-g', '--session', '-s', '--project', '-p', '--help', '-h')
    local vfox_help = clink.argmatcher():nofiles():addflags('--help', '-h')
    local vfox_shell = clink.argmatcher():nofiles():addarg('bash', 'zsh', 'pwsh', 'fish', 'clink')
    local vfox_uninstall = clink.argmatcher():nofiles():addarg(vfox_ls_func):addflags('--help', '-h')
    local vfox_install = clink.argmatcher():nofiles():addarg({
        onadvance = function() return 0 end,
        vfox_sdk_func,
    }):addflags('--all', '-a', '--help', '-h')

    vfox:addarg(
        'add' .. clink.argmatcher():nofiles():addarg({
            onadvance = function() return 0 end,
            function(word, word_index, line_state)
                local res, line = {}, line_state:getline()
                for _, v in ipairs(vfox_sdk_table) do
                    if not line:find(v) then
                        table.insert(res, v)
                    end
                end
                return res
            end
        }):addflags('--source', '-s', '--alias', '--help', '-h'),
        'use' .. vfox_use, 'u' .. vfox_use,
        'info' .. vfox_sdk,
        'remove' .. vfox_sdk,
        'search' .. vfox_sdk,
        'update' .. clink.argmatcher():nofiles():addarg(vfox_sdk_func):addflags('--all', '-a', '--help', '-h'),
        'available' .. vfox_help,
        'upgrade' .. vfox_help,
        'current' .. vfox_sdk, 'c' .. vfox_sdk,
        'list' .. vfox_sdk, 'ls' .. vfox_sdk,
        'uninstall' .. vfox_uninstall, 'un' .. vfox_uninstall,
        'install' .. vfox_install, 'i' .. vfox_install,
        'env' .. clink.argmatcher():nofiles():addflags(
            '--shell' .. vfox_shell, '-s' .. vfox_shell,
            '--cleanup', '-c',
            '--json', '-j',
            '--help', '-h'
        ),
        'activate' .. vfox_shell,
        'config' .. clink.argmatcher():nofiles():addarg(function()
            local res, vfox_config = {}, io.popen('vfox config -l')
            for line in vfox_config:lines() do
                local txt = line:gsub('%c%[%d+m', ''):match('^(%S+)')
                if txt then
                    table.insert(res, txt)
                end
            end
            vfox_config:close()
            return res
        end):addflags('--list', '-l', '--unset', '-un', '--help', '-h'),
        'cd' .. clink.argmatcher():nofiles():addarg(vfox_sdk_func):addflags('--plugin', '-p', '--help', '-h'),
        'help', 'h'
    ):addflags('--debug', '--help', '-h', '--version', '-v', '-V')
end)

local vfox_setenv = function(str)
    local key, val = str:match('^set "(.+)=(.*)"')
    if key and val then
        return os.setenv(key, val ~= '' and val or nil)
    end
end

os.setenv('__VFOX_PID', os.getpid())
local vfox_activate = io.popen('vfox activate clink')
for line in vfox_activate:lines() do
    if not vfox_setenv(line) then
        os.execute(line)
    end
end
vfox_activate:close()

local vfox_prompt = clink.promptfilter(30)
function vfox_prompt:filter(prompt)
    local env = io.popen('vfox env -s clink')
    for line in env:lines() do
        vfox_setenv(line)
    end
    env:close()
end
