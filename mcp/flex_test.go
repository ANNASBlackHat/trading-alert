package mcp

import (
	"testing"
)

func TestToBool(t *testing.T) {
	if !toBool(true) {
		t.Error("toBool(true) = false")
	}
	if toBool(false) {
		t.Error("toBool(false) = true")
	}
	if !toBool(float64(1)) {
		t.Error("toBool(1.0) = false")
	}
	if toBool(nil) {
		t.Error("toBool(nil) = true")
	}
}

func TestCoerceStr(t *testing.T) {
	cases := []struct {
		in   any
		want string
	}{
		{nil, ""},
		{"plain", "plain"},
		{float64(42), "42"},
		{int32(7), "7"},
		{[]any{"a", "b"}, "a, b"},
		{[]any{"x", float64(3)}, "x, 3"},
		{[]any{[]any{"inner1", "inner2"}}, "inner1, inner2"},
		{map[string]any{"k": "v"}, "map[k:v]"},
	}
	for _, c := range cases {
		if got := coerceStr(c.in); got != c.want {
			t.Errorf("coerceStr(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestToStrList(t *testing.T) {
	cases := []struct {
		in   any
		want []string
	}{
		{nil, nil},
		{"", nil},
		{"single", []string{"single"}},
		{[]any{"a", "b"}, []string{"a", "b"}},
		{[]any{[]any{"x", "y"}}, []string{"x, y"}},
		{[]any{float64(1), float64(2)}, []string{"1", "2"}},
		{float64(5), []string{"5"}},
	}
	for _, c := range cases {
		got := toStrList(c.in)
		if len(got) != len(c.want) {
			t.Errorf("toStrList(%v) = %v (len %d), want %v (len %d)", c.in, got, len(got), c.want, len(c.want))
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("toStrList(%v)[%d] = %q, want %q", c.in, i, got[i], c.want[i])
			}
		}
	}
}

func TestClassifyStance(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", "other"},
		{"bullish", "bull"},
		{"bearish", "bear"},
		{"neutral", "neutral"},
		{"long", "bull"},
		{"short", "bear"},
		{"buy", "bull"},
		{"sell", "bear"},
		{"overweight", "bull"},
		{"underweight", "bear"},
		{"mixed", "neutral"},
		{"wait-and-see", "neutral"},
		{"something weird", "other"},
	}
	for _, c := range cases {
		if got := ClassifyStance(c.in); got != c.want {
			t.Errorf("ClassifyStance(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestRound2(t *testing.T) {
	cases := []struct {
		in, want float64
	}{
		{0, 0},
		{0.1234, 0.12},
		{0.999, 1.0},
		{0.5, 0.5},
	}
	for _, c := range cases {
		if got := round2(c.in); got != c.want {
			t.Errorf("round2(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}
