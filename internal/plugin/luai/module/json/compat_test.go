package json

import (
	"bytes"
	"encoding/json"
	"errors"
	"math/rand"
	"testing"

	lua "github.com/yuin/gopher-lua"
)

// legacyValue is the encoder as it was before EncodeStrict and the repeat cap were
// added. TestEncodeMatchesLegacy keeps plain Encode compatible with it on every input
// where the legacy encoder reports no error, plus the error cases that are unchanged.
type legacyValue struct {
	lua.LValue
	visited map[*lua.LTable]bool
}

func (j legacyValue) MarshalJSON() (data []byte, err error) {
	switch converted := j.LValue.(type) {
	case lua.LBool:
		data, err = json.Marshal(bool(converted))
	case lua.LNumber:
		data, err = json.Marshal(float64(converted))
	case *lua.LNilType:
		data = []byte(`null`)
	case lua.LString:
		data, err = json.Marshal(string(converted))
	case *lua.LTable:
		if j.visited[converted] {
			return nil, errNested
		}
		j.visited[converted] = true
		key, value := converted.Next(lua.LNil)
		switch key.Type() {
		case lua.LTNil:
			data = []byte(`[]`)
		case lua.LTNumber:
			arr := make([]legacyValue, 0, converted.Len())
			expectedKey := lua.LNumber(1)
			for key != lua.LNil {
				if key.Type() != lua.LTNumber {
					return nil, errInvalidKeys
				}
				if expectedKey != key {
					return nil, errSparseArray
				}
				arr = append(arr, legacyValue{value, j.visited})
				expectedKey++
				key, value = converted.Next(key)
			}
			data, err = json.Marshal(arr)
		case lua.LTString:
			obj := make(map[string]legacyValue)
			for key != lua.LNil {
				if key.Type() != lua.LTString {
					return nil, errInvalidKeys
				}
				obj[key.String()] = legacyValue{value, j.visited}
				key, value = converted.Next(key)
			}
			data, err = json.Marshal(obj)
		default:
			err = errInvalidKeys
		}
	default:
		err = invalidTypeError(j.LValue.Type())
	}
	return
}

func legacyEncode(v lua.LValue) ([]byte, error) {
	return json.Marshal(legacyValue{v, make(map[*lua.LTable]bool)})
}

func randomValue(s *lua.LState, r *rand.Rand, depth int) lua.LValue {
	switch k := r.Intn(7); {
	case k == 0:
		return lua.LBool(r.Intn(2) == 0)
	case k == 1:
		return lua.LNumber(r.Float64() * 1000)
	case k == 2:
		return lua.LString(string([]byte{byte(r.Intn(256)), 'a', byte(r.Intn(256))}))
	case k == 3:
		return lua.LNil
	case depth > 5:
		return lua.LNumber(1)
	}
	t := s.NewTable()
	n := r.Intn(4)
	if r.Intn(2) == 0 {
		for i := 0; i < n; i++ {
			t.Append(randomValue(s, r, depth+1))
		}
	} else {
		for i := 0; i < n; i++ {
			t.RawSetString(string(rune('a'+i)), randomValue(s, r, depth+1))
		}
	}
	return t
}

func TestEncodeMatchesLegacy(t *testing.T) {
	s := lua.NewState()
	defer s.Close()
	r := rand.New(rand.NewSource(1))
	for i := 0; i < 20000; i++ {
		v := randomValue(s, r, 0)
		want, wantErr := legacyEncode(v)
		got, err := Encode(v)
		if (err == nil) != (wantErr == nil) || !bytes.Equal(got, want) {
			t.Fatalf("input %d: got %q (%v), legacy %q (%v)", i, got, err, want, wantErr)
		}
	}
	// a self-referencing table is still reported as a cycle
	loop := s.NewTable()
	loop.RawSetString("self", loop)
	if _, err := Encode(loop); !errors.Is(err, errNested) {
		t.Fatalf("cycle: got %v", err)
	}
	// 300 levels (the legacy encoder has no depth cap)
	var deep lua.LValue = lua.LNumber(1)
	for i := 0; i < 300; i++ {
		x := s.NewTable()
		x.Append(deep)
		deep = x
	}
	got, err := Encode(deep)
	want, wantErr := legacyEncode(deep)
	if err != nil || wantErr != nil || !bytes.Equal(got, want) {
		t.Fatalf("deep: %v %v", err, wantErr)
	}
}
