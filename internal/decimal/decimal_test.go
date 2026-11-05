package decimal

import "testing"

func TestRound(t *testing.T) {
	cases := []struct {
		in   float64
		prec int
		want float64
	}{
		{1.005, 2, 1.01},
		{1.004, 2, 1.0},
		{-1.005, 2, -1.01},
		{2.675, 2, 2.68},
		{0.125, 2, 0.13},
		{123.4567, 0, 123},
		{0.5, 0, 1},
		{-0.5, 0, -1},
		{1234.5, -1, 1230},
		{0.1 + 0.2, 2, 0.3},
		{-0.001, 2, 0},
	}
	for _, c := range cases {
		if got := Round(c.in, c.prec); got != c.want {
			t.Errorf("Round(%v, %d) = %v, want %v", c.in, c.prec, got, c.want)
		}
	}
}

func TestFormat(t *testing.T) {
	cases := []struct {
		in   float64
		prec int
		want string
	}{
		{1.005, 2, "1.01"},
		{1000, 2, "1000.00"},
		{0.16, 6, "0.160000"},
		{-0.001, 2, "0.00"},
		{2.5, 0, "3"},
	}
	for _, c := range cases {
		if got := Format(c.in, c.prec); got != c.want {
			t.Errorf("Format(%v, %d) = %q, want %q", c.in, c.prec, got, c.want)
		}
	}
}

func TestParse(t *testing.T) {
	if Parse(" 1,234.50 ") != 1234.5 || Parse("") != 0 || Parse("abc") != 0 {
		t.Fatal("parse failed")
	}
}
