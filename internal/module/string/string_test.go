package string

import (
	lua "github.com/yuin/gopher-lua"
	"testing"
)

func TestSplit(t *testing.T) {
	const str = `
	local strings = require("vfox.strings")
    local str_parts = strings.split("hello world", " ")
    assert(type(str_parts) == 'table')
    assert(#str_parts == 2, string.format("%d ~= 2", #str_parts))
    assert(str_parts[1] == "hello", string.format("%s ~= hello", str_parts[1]))
    assert(str_parts[2] == "world", string.format("%s ~= world", str_parts[2]))
	`
	eval(str, t)
}
func TestHasPrefix(t *testing.T) {
	const str = `
	local strings = require("vfox.strings")
	assert(strings.has_prefix("hello world", "hello"), [[not strings.has_prefix("hello")]])
	`
	eval(str, t)
}
func TestHasSuffix(t *testing.T) {
	const str = `
	local strings = require("vfox.strings")
	assert(strings.has_suffix("hello world", "world"), [[not strings.has_suffix("world")]])
	`
	eval(str, t)
}

func TestTrim(t *testing.T) {
	const str = `
	local strings = require("vfox.strings")
    assert(strings.trim("hello world", "world") == "hello ", "strings.trim()")
    assert(strings.trim("hello world", "hello ") == "world", "strings.trim()")
    assert(strings.trim_prefix("hello world", "hello ") == "world", "strings.trim()")
    assert(strings.trim_suffix("hello world", "hello ") == "hello world", "strings.trim()")
	`
	eval(str, t)
}

func TestContains(t *testing.T) {
	const str = `
	local strings = require("vfox.strings")
	assert(strings.contains("hello world", "hello ") == true, "strings.contains()")
`
	eval(str, t)
}

func TestJoin(t *testing.T) {
	const str = `
	local strings = require("vfox.strings")
	local str = strings.join({"1",3,"4"},";")
	assert(str == "1;3;4", "strings.join()")
`
	eval(str, t)
}

func TestTrimSpace(t *testing.T) {
	const str = `
	local strings = require("vfox.strings")
	tests = {
        {
            name = "string with trailing whitespace",
            input = "foo bar    ",
            expected = "foo bar",
        },
        {
            name = "string with leading whitespace",
            input = "   foo bar",
            expected = "foo bar",
        },
        {
            name = "string with leading and trailing whitespace",
            input = "   foo bar   ",
            expected = "foo bar",
        },
        {
            name = "string with no leading or trailing whitespace",
            input = "foo bar",
            expected = "foo bar",
        },
    }

    for _, tt in ipairs(tests) do
		got = strings.trim_space(tt.input)
		assert(got == tt.expected, string.format([[expected "%s"; got "%s"]], expected, got))
    end
`
	eval(str, t)
}

func eval(str string, t *testing.T) {
	s := lua.NewState()
	defer s.Close()

	Preload(s)
	if err := s.DoString(str); err != nil {
		t.Error(err)
	}
}
