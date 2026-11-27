package json

import (
	"encoding/json"
	"math/rand"
	"strings"
	"testing"

	lua "github.com/yuin/gopher-lua"
)

// encodeCharged runs the encoder and also returns the bytes it charged.
func encodeCharged(v lua.LValue, strict bool) ([]byte, int, error) {
	e := &encoder{visited: map[*lua.LTable]bool{}, done: map[*lua.LTable]bool{}, strict: strict}
	out, err := json.Marshal(jsonValue{LValue: v, enc: e})
	return out, e.bytes, err
}

var budgetAlphabet = []string{"a", "z", "<", ">", "&", `"`, `\`, "\n", "\r", "\t", "\b", "\f", "\x00", "\x1f", "\x7f", "é", "€", "\u2028", "\u2029", "😀", "\xff", "\xc3"}

func randomString(r *rand.Rand, valid bool) string {
	var b strings.Builder
	for i, n := 0, r.Intn(12); i < n; i++ {
		c := budgetAlphabet[r.Intn(len(budgetAlphabet))]
		if valid && strings.HasPrefix(c, "\xff") || valid && c == "\xc3" {
			c = "x"
		}
		b.WriteString(c)
	}
	return b.String()
}

// The charge for a string must cover what encoding/json writes for it, and must
// not be far above it.
func TestQuotedLenCoversOutput(t *testing.T) {
	r := rand.New(rand.NewSource(7))
	for i := 0; i < 50000; i++ {
		s := randomString(r, false)
		out, _ := json.Marshal(s)
		got := quotedLen(s)
		if got < len(out) {
			t.Fatalf("%q: charged %d, wrote %d (%s)", s, got, len(out), out)
		}
		if got > 6*len(s)+2 {
			t.Fatalf("%q: charged %d, far above %d", s, got, len(out))
		}
	}
	for _, s := range []string{"", "<", strings.Repeat("<", 1000), "\u2028", "\xff", "\x00", "\"\\"} {
		out, _ := json.Marshal(s)
		if quotedLen(s) < len(out) {
			t.Fatalf("%q: charged %d, wrote %d", s, quotedLen(s), len(out))
		}
	}
}

func randomTree(s *lua.LState, r *rand.Rand, depth int, pool *[]*lua.LTable) lua.LValue {
	switch k := r.Intn(8); {
	case k == 0:
		return lua.LBool(r.Intn(2) == 0)
	case k == 1:
		return lua.LNumber(r.NormFloat64() * 1e6)
	case k == 2 || k == 3:
		return lua.LString(randomString(r, true))
	case k == 4:
		return lua.LNil
	case k == 5 && len(*pool) > 0:
		return (*pool)[r.Intn(len(*pool))] // a shared table, written once per reference
	case depth > 4:
		return lua.LNumber(1)
	}
	t := s.NewTable()
	n := r.Intn(5)
	if r.Intn(2) == 0 {
		for i := 0; i < n; i++ {
			t.Append(randomTree(s, r, depth+1, pool))
		}
	} else {
		for i := 0; i < n; i++ {
			t.RawSetString(randomString(r, true)+string(rune('a'+i)), randomTree(s, r, depth+1, pool))
		}
	}
	*pool = append(*pool, t)
	return t
}

// For every valid input the strict encoder must have charged at least the bytes it
// wrote (strings, keys, numbers, brackets, commas and colons included), and not
// absurdly more.
func TestStrictChargeCoversOutput(t *testing.T) {
	s := lua.NewState()
	defer s.Close()
	r := rand.New(rand.NewSource(11))
	checked := 0
	for i := 0; i < 20000; i++ {
		var pool []*lua.LTable
		v := randomTree(s, r, 0, &pool)
		out, charged, err := encodeCharged(v, true)
		if err != nil {
			continue // mixed keys and the like are rejected before any budget question
		}
		checked++
		if charged < len(out) {
			t.Fatalf("input %d: charged %d, wrote %d: %s", i, charged, len(out), out)
		}
		if charged > 4*len(out)+16 {
			t.Fatalf("input %d: charged %d, wrote only %d: %s", i, charged, len(out), out)
		}
	}
	if checked < 5000 {
		t.Fatalf("only %d inputs were valid", checked)
	}
}

func TestStrictBudgetBoundary(t *testing.T) {
	old := maxBytes
	defer func() { maxBytes = old }()
	s := lua.NewState()
	defer s.Close()
	// an escaped string is charged at its written size: 10 "<" are 60 bytes plus quotes
	str := lua.LString(strings.Repeat("<", 10))
	out, charged, err := encodeCharged(str, true)
	if err != nil || len(out) != 62 || charged != 62 {
		t.Fatalf("wrote %d charged %d err %v", len(out), charged, err)
	}
	maxBytes = 62
	if _, err := EncodeStrict(str); err != nil {
		t.Fatalf("exactly the budget must fit: %v", err)
	}
	maxBytes = 61
	if _, err := EncodeStrict(str); err == nil {
		t.Fatal("one byte over the budget must be refused")
	}
	// the same string twice in a list: 62*2 plus brackets and a comma
	list := s.NewTable()
	list.Append(str)
	list.Append(str)
	// 127 bytes are written; one byte more is charged for the last comma slot
	maxBytes = 128
	if _, err := EncodeStrict(list); err != nil {
		t.Fatalf("list should fit 128: %v", err)
	}
	maxBytes = 127
	if _, err := EncodeStrict(list); err == nil {
		t.Fatal("charge is 128, so 127 must be refused")
	}
	maxBytes = 100
	if _, err := EncodeStrict(list); err == nil {
		t.Fatal("two escaped copies must exceed 100")
	}
	// keys are charged by their escaped size as well
	obj := s.NewTable()
	obj.RawSetString(strings.Repeat("&", 40), lua.LNumber(1))
	// {"&&&...":1} writes 246 bytes (242 for the escaped key); 247 are charged
	maxBytes = 247
	if _, err := EncodeStrict(obj); err != nil {
		t.Fatalf("object of 246 written bytes must fit 247: %v", err)
	}
	maxBytes = 246
	if _, err := EncodeStrict(obj); err == nil {
		t.Fatal("charge is 247, so 246 must be refused")
	}
	maxBytes = 100
	if _, err := EncodeStrict(obj); err == nil {
		t.Fatal("an escaped key above the budget must be refused")
	}
}

// Plain encode charges only what is written for repeat references, so ordinary data
// stays free while a shared table cannot expand without bound.
func TestPlainEncodeRepeatBytesAreBudgeted(t *testing.T) {
	old := maxBytes
	maxBytes = 1 << 20
	defer func() { maxBytes = old }()
	s := lua.NewState()
	defer s.Close()
	big := lua.LString(strings.Repeat("x", 300<<10))
	// no sharing: three 300 KiB strings are over the budget in total, and still fine
	plain := s.NewTable()
	for i := 0; i < 3; i++ {
		t2 := s.NewTable()
		t2.Append(lua.LString(strings.Repeat("x", 300<<10)))
		plain.Append(t2)
	}
	if _, err := Encode(plain); err != nil {
		t.Fatalf("data without shared tables must not be limited: %v", err)
	}
	// one shared table referenced twice: the second copy is charged
	shared := s.NewTable()
	shared.Append(big)
	two := s.NewTable()
	two.Append(shared)
	two.Append(shared)
	if _, err := Encode(two); err != nil {
		t.Fatalf("one repeat of 300 KiB fits in 1 MiB: %v", err)
	}
	for i := 0; i < 4; i++ {
		two.Append(shared)
	}
	if _, err := Encode(two); err == nil {
		t.Fatal("five repeats of 300 KiB must exceed 1 MiB")
	}
}

// The maintainer's input: a 1 MiB leaf under a chain where each table references the
// level below twice, which doubles the output per level.
func TestPlainEncodeSharedChainWithLargeLeaf(t *testing.T) {
	const str = `
	local json = require("JSON")
	local big = string.rep("x", 1024 * 1024)
	local t = { big }
	for i = 1, 8 do t = { a = t, b = t } end
	local out, err = json.encode(t)
	assert(out == nil and string.find(err, "too large") or out == nil and string.find(err, "64 MiB"), tostring(err))
	`
	s := lua.NewState()
	defer s.Close()
	s.PreloadModule("JSON", loader)
	if err := s.DoString(str); err != nil {
		t.Error(err)
	}
}
