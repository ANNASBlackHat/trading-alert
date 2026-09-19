package mcp

import (
	"reflect"
	"testing"
)

// TestCoerceStr verifies that heterogeneous values (string, number, array,
// nil) all coerce into a plain string without error.
func TestCoerceStr(t *testing.T) {
	cases := []struct {
		name string
		in   any
		want string
	}{
		{"string", "strong", "strong"},
		{"float", 4.2, "4.2"},
		{"int64", int64(7), "7"},
		{"array", []any{"a", "b"}, "a, b"},
		{"nested", []any{"x", float64(3)}, "x, 3"},
		{"nil", nil, ""},
	}
	for _, c := range cases {
		if got := coerceStr(c.in); got != c.want {
			t.Errorf("coerceStr(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestToStrList verifies that arrays, scalars, and nil all coerce into a
// string slice without error.
func TestToStrList(t *testing.T) {
	cases := []struct {
		name string
		in   any
		want []string
	}{
		{"string", "single", []string{"single"}},
		{"array", []any{"a", "b"}, []string{"a", "b"}},
		{"empty string", "", nil},
		{"nil", nil, nil},
	}
	for _, c := range cases {
		got := toStrList(c.in)
		if !reflect.DeepEqual(normalizeList(got), normalizeList(c.want)) {
			t.Errorf("%s: toStrList(%v) = %v, want %v", c.name, c.in, got, c.want)
		}
	}
}

func normalizeList(s []string) []string {
	if len(s) == 0 {
		return []string{}
	}
	out := make([]string, len(s))
	copy(out, s)
	return out
}
