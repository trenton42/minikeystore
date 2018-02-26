package storage

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestCheckKey(t *testing.T) {
	s := New()
	_, err := s.checkkey("key", "string", false)
	if err == nil {
		t.Error("Key does not exist, but no error returned")
	}

	for _, key := range []string{"string", "map", "list"} {
		i, err := s.checkkey(key, key, true)
		if err != nil {
			t.Errorf("Unexpected error creating item %v", err)
		}
		if i.Type != key {
			t.Errorf("Type mismatch: expected %s got %s", key, i.Type)
		}
	}

	_, err = s.checkkey("string", "map", false)
	if err == nil {
		t.Error("Did not catch type mismatch. Sent string, expected map")
	}

	i, err := s.checkkey("string", "string", false)
	if i.Type != "string" {
		t.Errorf("Expected string, got %s", i.Type)
	}
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestUpdateIndex(t *testing.T) {
	s := New()

	var checks = []struct {
		key    string
		add    bool
		result []string
	}{
		{"aaa", true, []string{"aaa"}},
		{"aaa", true, []string{"aaa"}},
		{"zzz", true, []string{"aaa", "zzz"}},
		{"bbb", true, []string{"aaa", "bbb", "zzz"}},
		{"ccc", false, []string{"aaa", "bbb", "zzz"}},
		{"bbb", false, []string{"aaa", "zzz"}},
		{"aaa", false, []string{"zzz"}},
		{"zzz", false, []string{}},
		{"zzz", false, []string{}},
	}

	for ind, c := range checks {
		s.updateIndex(c.key, c.add)
		if !checkSlice(s.index, c.result) {
			t.Errorf("[run %d] Index does not match: %v != %v", ind, s.index, c.result)
		}
	}
}

func TestSet(t *testing.T) {
	s := New()

	var tests = []struct {
		key      string
		hasError bool
		value    interface{}
		Type     string
	}{
		{"a", false, "string", "string"},
		{"b", false, []string{"string"}, "list"},
		{"e", false, []interface{}{"string"}, "list"},
		{"c", false, map[string]string{"key": "value"}, "map"},
		{"f", false, map[string]interface{}{"key": "value"}, "map"},
		{"d", true, 7, "int"},
		{"b", false, "new string", "string"},
	}

	for i, test := range tests {
		err := s.Set(test.key, test.value)
		if (err != nil) != test.hasError {
			t.Errorf("[Run %d] Expected error: %t. Had error: %t", i, test.hasError, (err == nil))
		}
		if err != nil {
			continue
		}
		val, ok := s.items[test.key]
		if !ok {
			t.Errorf("Key %s not found", test.key)
		}
		if val.Type != test.Type {
			t.Errorf("[Run %d] Type %v != %v", i, val.Type, test.Type)
		}

		res, err := s.Get(test.key)

		if err != nil {
			t.Errorf("[run %d] Unexpected error: %v", i, err)
		}

		out, err := json.Marshal(test.value)
		if err != nil {
			t.Errorf("[run %d] Error unmarshalling: %v", i, err)
		}
		if bytes.Compare(out, res) != 0 {
			t.Errorf("[run %d] Values do not match %v != %v", i, res, out)
		}
	}
}

func TestGet(t *testing.T) {
	s := New()

	_, err := s.Get("missing")
	if err == nil {
		t.Error("No error on getting missing key")
	}
	s.items["invalid"] = &Item{Type: "invalid"}

	_, err = s.Get("invalid")
	if err == nil || err.Error() != "unknown type" {
		t.Errorf("Missing or wrong error %v", err)
	}
}

func TestDelete(t *testing.T) {
	s := New()
	for _, k := range []string{"a", "b", "c", "d", "e"} {
		s.Set(k, "somevalue")
	}

	var tests = []struct {
		key    string
		length int
		index  []string
	}{
		{"z", 5, []string{"a", "b", "c", "d", "e"}},
		{"c", 4, []string{"a", "b", "d", "e"}},
		{"a", 3, []string{"b", "d", "e"}},
	}

	for i, test := range tests {
		s.Delete(test.key)
		if len(s.items) != test.length {
			t.Errorf("[run %d] wrong length: %d != %d", i, len(s.items), test.length)
		}
	}

}

func TestAppend(t *testing.T) {
	s := New()
	s.Set("a", []string{})
	s.Set("b", []string{"z"})
	s.Set("c", "string")

	var tests = []struct {
		key      string
		append   string
		hasError bool
		value    []string
	}{
		{"a", "b", false, []string{"b"}},
		{"missing", "b", false, []string{"b"}},
		{"b", "b", false, []string{"z", "b"}},
		{"c", "b", true, []string{}},
	}

	for i, test := range tests {
		err := s.Append(test.key, test.append)
		if (err != nil) != test.hasError {
			t.Errorf("[run %d] Error missmatch: Expected: %t, had: %t", i, test.hasError, (err != nil))
		}
		if err != nil {
			continue
		}

		if !checkSlice(s.items[test.key].listValue, test.value) {
			t.Errorf("[run %d] Values do not match: %v != %v", i, test.value, s.items[test.key].listValue)
		}
	}

}
func TestPop(t *testing.T) {
	s := New()
	s.Set("a", []string{"a", "b", "c"})
	s.Set("b", []string{})
	s.Set("c", "string")

	var tests = []struct {
		key      string
		ret      string
		hasError bool
		value    []string
	}{
		{"a", "c", false, []string{"a", "b"}},
		{"missing", "", true, []string{}},
		{"b", "", true, []string{"z", "b"}},
		{"c", "", true, []string{}},
	}

	for i, test := range tests {
		ret, err := s.Pop(test.key)
		if (err != nil) != test.hasError {
			t.Errorf("[run %d] Error missmatch: Expected: %t, had: %t", i, test.hasError, (err != nil))
		}
		if err != nil {
			continue
		}
		if ret != test.ret {
			t.Errorf("[run %d] mismatched return: %s != %s", i, test.ret, ret)
		}
		if !checkSlice(s.items[test.key].listValue, test.value) {
			t.Errorf("[run %d] Values do not match: %v != %v", i, test.value, s.items[test.key].listValue)
		}
	}

}

func TestMapGet(t *testing.T) {
	s := New()
	s.Set("a", map[string]string{"a": "b", "b": "c"})
	s.Set("b", map[string]string{})
	s.Set("c", "string")
	s.items["m"] = &Item{Type: "map", mapValue: nil}

	var tests = []struct {
		key      string
		mapkey   string
		value    string
		hasError bool
	}{
		{"a", "a", "b", false},
		{"b", "a", "", true},
		{"c", "a", "", true},
		{"d", "a", "", true},
		{"m", "a", "", true},
	}

	for i, test := range tests {
		val, err := s.MapGet(test.key, test.mapkey)
		if (err != nil) != test.hasError {
			t.Errorf("[run %d] Error mismatch: expected: %t had error %t", i, test.hasError, (err != nil))
		}
		if err != nil {
			continue
		}
		if val != test.value {
			t.Errorf("[run %d] value mismatch: %s != %s", i, test.value, val)
		}
	}
}

func TestMapSet(t *testing.T) {
	s := New()
	s.Set("a", map[string]string{"a": "b", "b": "c"})
	s.Set("b", map[string]string{})
	s.Set("c", "string")
	s.items["m"] = &Item{Type: "map", mapValue: nil}

	var tests = []struct {
		key      string
		mapkey   string
		value    string
		result   map[string]string
		hasError bool
	}{
		{"a", "a", "z", map[string]string{"a": "z", "b": "c"}, false},
		{"a", "m", "z", map[string]string{"a": "z", "b": "c", "m": "z"}, false},
		{"b", "a", "a", map[string]string{"a": "a"}, false},
		{"m", "a", "a", map[string]string{"a": "a"}, false},
		{"c", "a", "a", nil, true},
		{"d", "a", "a", map[string]string{"a": "a"}, false},
	}

	for i, test := range tests {
		err := s.MapSet(test.key, test.mapkey, test.value)
		if (err != nil) != test.hasError {
			t.Errorf("[run %d] Error mismatch: expected: %t had error %t", i, test.hasError, (err != nil))
		}
		if err != nil {
			continue
		}
		val := s.items[test.key].mapValue
		if !checkMap(val, test.result) {
			t.Errorf("[run %d] value mismatch: %v != %v", i, test.result, val)
		}
	}
}

func TestMapDelete(t *testing.T) {
	s := New()
	s.Set("a", map[string]string{"a": "b", "b": "c"})
	s.Set("b", map[string]string{})
	s.Set("c", "string")
	s.items["m"] = &Item{Type: "map", mapValue: nil}

	var tests = []struct {
		key      string
		mapkey   string
		result   map[string]string
		hasError bool
	}{
		{"a", "a", map[string]string{"b": "c"}, false},
		{"a", "m", map[string]string{"b": "c"}, false},
		{"b", "a", map[string]string{}, false},
		{"m", "a", map[string]string{}, false},
		{"c", "a", nil, true},
		{"d", "a", nil, true},
	}

	for i, test := range tests {
		err := s.MapDelete(test.key, test.mapkey)
		if (err != nil) != test.hasError {
			t.Errorf("[run %d] Error mismatch: expected: %t had error %t", i, test.hasError, (err != nil))
		}
		if err != nil {
			continue
		}
		val := s.items[test.key].mapValue
		if !checkMap(val, test.result) {
			t.Errorf("[run %d] value mismatch: %v != %v", i, test.result, val)
		}
	}
}

func TestGetIndex(t *testing.T) {
	s := New()
	s.index = []string{"aaa", "aaa:bbb:ccc", "abc", "ccc:aaa:bbb"}
	s.index.Sort()

	var tests = []struct {
		search string
		result []string
	}{
		{"abc", []string{"abc"}},
		{"cba", []string{}},
		{"*d*", []string{}},
		{"a*", []string{"aaa", "aaa:bbb:ccc", "abc"}},
		{"*", []string{"aaa", "aaa:bbb:ccc", "abc", "ccc:aaa:bbb"}},
		{"*ccc*", []string{"aaa:bbb:ccc", "ccc:aaa:bbb"}},
	}

	for i, test := range tests {
		res := s.GetIndex(test.search)
		if !checkSlice(res, test.result) {
			t.Errorf("[run %d] Index does not match: %v != %v", i, test.result, res)
		}
	}
}

func checkSlice(a []string, b []string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func checkMap(a map[string]string, b map[string]string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if a[k] != b[k] {
			return false
		}
	}

	return true
}
