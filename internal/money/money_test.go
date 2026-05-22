package money

import "testing"

func mustParse(t *testing.T, s string) Money {
	t.Helper()
	m, err := Parse(s)
	if err != nil {
		t.Fatalf("Parse(%q): %v", s, err)
	}
	return m
}

func TestParseAccepts(t *testing.T) {
	for _, s := range []string{"15822.42", "150000.00", "0.01", "100", "100.5", "15822.420"} {
		if _, err := Parse(s); err != nil {
			t.Errorf("Parse(%q) returned unexpected error: %v", s, err)
		}
	}
}

func TestParseRejects(t *testing.T) {
	for _, s := range []string{"15822.423", "0.001", "abc", "", "1.2.3"} {
		if _, err := Parse(s); err == nil {
			t.Errorf("Parse(%q) should have failed", s)
		}
	}
}

func TestUnmarshalJSONExact(t *testing.T) {
	var m Money
	if err := m.UnmarshalJSON([]byte("15822.42")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := m.String(); got != "15822.42" {
		t.Errorf("got %q, want 15822.42", got)
	}
}

func TestUnmarshalJSONRejects(t *testing.T) {
	for _, bad := range []string{`"15822.42"`, "null", "abc", "1.234"} {
		var m Money
		if err := m.UnmarshalJSON([]byte(bad)); err == nil {
			t.Errorf("UnmarshalJSON(%s) should have failed", bad)
		}
	}
}

func TestMarshalJSONFixedScale(t *testing.T) {
	cases := map[string]string{
		"450000":   "450000.00",
		"150000.5": "150000.50",
		"0":        "0.00",
		"0.01":     "0.01",
	}
	for in, want := range cases {
		got, err := mustParse(t, in).MarshalJSON()
		if err != nil {
			t.Fatalf("MarshalJSON: %v", err)
		}
		if string(got) != want {
			t.Errorf("MarshalJSON(%q) = %s, want %s", in, got, want)
		}
	}
}

func TestAddIsExact(t *testing.T) {
	// The classic float trap: 0.1 + 0.2 must equal exactly 0.30.
	if !mustParse(t, "0.1").Add(mustParse(t, "0.2")).Equal(mustParse(t, "0.3")) {
		t.Error("0.1 + 0.2 did not equal 0.3 exactly")
	}
}

func TestDivIntRounds(t *testing.T) {
	cases := []struct {
		sum  string
		n    int
		want string
	}{
		{"2800000", 15, "186666.67"},
		{"28500000", 142, "200704.23"},
		{"1250000", 8, "156250"},
		{"450000", 3, "150000"},
	}
	for _, c := range cases {
		got := mustParse(t, c.sum).DivInt(c.n)
		if !got.Equal(mustParse(t, c.want)) {
			t.Errorf("%s / %d = %s, want %s", c.sum, c.n, got, c.want)
		}
	}
}

func TestDivIntByZero(t *testing.T) {
	if !mustParse(t, "100").DivInt(0).Equal(Zero) {
		t.Error("DivInt(0) should return Zero")
	}
}

func TestIsPositive(t *testing.T) {
	cases := map[string]bool{"0": false, "0.01": true, "-5": false, "100": true}
	for in, want := range cases {
		if got := mustParse(t, in).IsPositive(); got != want {
			t.Errorf("Parse(%q).IsPositive() = %v, want %v", in, got, want)
		}
	}
}