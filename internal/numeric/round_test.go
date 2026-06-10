package numeric

import "testing"

func TestRound4(t *testing.T) {
	cases := []struct {
		in   float64
		want float64
	}{
		{0, 0},
		{0.123456, 0.1235},
		{0.12344, 0.1234},
		{1.99999, 2.0},
		{-0.55555, -0.5556},
	}
	for _, c := range cases {
		got := Round4(c.in)
		if got != c.want {
			t.Errorf("Round4(%v) = %v; want %v", c.in, got, c.want)
		}
	}
}
