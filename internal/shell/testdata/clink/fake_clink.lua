-- A stand-in for the part of the Clink Lua API that clink_vfox.lua uses, so the
-- script can be exercised without Clink or Windows. It is loaded by
-- clink_test.go, which also replaces the io and os libraries.
--
-- The behaviour follows the Clink manual
-- (https://chrisant996.github.io/clink/clink.html#extending-clink) and
-- clink/lua/scripts/arguments.lua. It is deliberately small: it records the
-- argmatchers a script defines and answers "what would <Tab> complete here?".
-- Anything the script needs beyond this fails loudly (calling a missing API is
-- a Lua error), so this file gets extended on purpose rather than by accident.

local Matcher = {}
Matcher.__index = Matcher

-- 'word' .. matcher links the matcher to the word: once the word is entered,
-- the matcher parses the words that follow.
local Link = {}

local matchers = {} -- argmatchers registered by command name
local prompt_filters = {}
local prompt_coroutines = {} -- functions queued by clink.promptcoroutine

-- An argument position or the flags of a matcher: the words it offers (strings
-- and generator functions, in order) and the matchers its words link to.
local function new_slot()
    return { items = {}, links = {} }
end

-- Mirrors _argmatcher:_add(): tables are flattened, links are recorded by word
-- and the "onadvance" option is kept with the slot.
local function add(slot, item)
    if getmetatable(item) == Link then
        slot.links[item.word] = item.matcher
        table.insert(slot.items, item.word)
    elseif type(item) == 'table' then
        slot.onadvance = item.onadvance or slot.onadvance
        for _, v in ipairs(item) do
            add(slot, v)
        end
    else
        table.insert(slot.items, item)
    end
end

-- Every call adds one argument position; its arguments are the alternatives.
function Matcher:addarg(...)
    local slot = new_slot()
    add(slot, { ... })
    table.insert(self.args, slot)
    return self
end

-- Flags accumulate in a single slot.
function Matcher:addflags(...)
    self.flags = self.flags or new_slot()
    add(self.flags, { ... })
    return self
end

-- Besides disabling file completion, :nofiles() makes the matcher a dead end:
-- the parser never returns to the matcher that linked to it.
function Matcher:nofiles()
    self.dead_end = true
    return self
end

function Matcher:setdelayinit(fn)
    self.delayinit = fn
    return self
end

function Matcher.__concat(lhs, rhs)
    assert(type(lhs) == 'string' and getmetatable(rhs) == Matcher, 'only "word" .. argmatcher is supported')
    return setmetatable({ word = lhs, matcher = rhs }, Link)
end

clink = {}

function clink.argmatcher(name)
    if name and matchers[name] then
        return matchers[name]
    end
    local matcher = setmetatable({ args = {} }, Matcher)
    if name then
        matchers[name] = matcher
    end
    return matcher
end

function clink.promptfilter(priority)
    local filter = { priority = priority }
    table.insert(prompt_filters, filter)
    return filter
end

function clink.promptcoroutine(fn)
    table.insert(prompt_coroutines, fn)
end

-- Unlike Clink, which reports errors from coroutines and carries on, errors are
-- raised so that a failing script fails the test.
function clink.runcoroutineuntilcomplete(co)
    while coroutine.status(co) ~= 'dead' do
        local ok, err = coroutine.resume(co)
        if not ok then
            error(err, 0)
        end
    end
end

local function new_line_state(line, words)
    return {
        getline = function() return line end,
        getwordcount = function() return #words end,
        getword = function(_, index) return words[index] end,
        getendword = function() return words[#words] end,
        getcursor = function() return #line + 1 end,
    }
end

local function is_flag(matcher, word)
    return matcher.flags ~= nil and word:sub(1, 1) == '-'
end

-- The control surface used by the Go tests.
fakeclink = {}

-- Returns what pressing <Tab> would offer after `line` (a command line whose
-- last word is the one being completed; end it with a space to complete a new
-- word). Matches are filtered by the typed prefix, as Clink does.
--
-- Each call is one edit line: Clink runs an argmatcher's delay-init function
-- the first time the matcher is used in every edit line, not just once.
function fakeclink.complete(line)
    local words = {}
    for word in line:gmatch('%S+') do
        table.insert(words, word)
    end
    if #words == 0 or line:find('%s$') then
        table.insert(words, '') -- the word under the cursor
    end

    local command = words[1]
    local root = matchers[command]
    if not root then
        error('no argmatcher for ' .. command, 2)
    end
    if root.delayinit then
        root.delayinit(root, command)
    end

    local state = new_line_state(line, words)
    local user_data = {}
    local matcher, arg_index = root, 1

    -- Parse the words before the one under the cursor.
    for index = 2, #words - 1 do
        local word = words[index]
        local flag = is_flag(matcher, word)
        local slot = flag and matcher.flags or matcher.args[arg_index]
        local link = slot and slot.links[word]
        if link then
            matcher, arg_index = link, 1
        elseif not flag then
            local advance = slot and slot.onadvance and slot.onadvance(arg_index, word, index, state, user_data)
            if advance == nil then
                arg_index = arg_index + 1
            elseif advance ~= 0 then
                error('unsupported onadvance result: ' .. tostring(advance), 2)
            end
        end
    end

    -- Generate matches for the word under the cursor.
    local word = words[#words]
    local slot
    if is_flag(matcher, word) then
        slot = matcher.flags
    else
        slot = matcher.args[arg_index]
    end

    local matches = {}
    for _, item in ipairs(slot and slot.items or {}) do
        local offered = { item }
        if type(item) == 'function' then
            -- Same arguments as Clink: word, word_index, line_state, match_builder, user_data.
            offered = item(word, #words, state, {}, user_data)
            if type(offered) ~= 'table' then
                offered = {}
            end
        end
        -- Only the array part of a returned table is read, as Clink does.
        for _, match in ipairs(offered) do
            if type(match) == 'table' then
                match = match.match
            end
            if match:sub(1, #word) == word then
                table.insert(matches, match)
            end
        end
    end
    return matches
end

-- Runs the prompt filters the way Clink does when it draws a prompt: each
-- filter runs, then the functions they queued with clink.promptcoroutine run to
-- completion.
function fakeclink.filter_prompt()
    for _, filter in ipairs(prompt_filters) do
        if filter.filter then
            filter:filter('prompt> ')
        end
    end
    local queued = prompt_coroutines
    prompt_coroutines = {}
    for _, fn in ipairs(queued) do
        clink.runcoroutineuntilcomplete(coroutine.create(fn))
    end
end

function fakeclink.prompt_filter_priorities()
    local priorities = {}
    for _, filter in ipairs(prompt_filters) do
        table.insert(priorities, filter.priority)
    end
    return priorities
end
