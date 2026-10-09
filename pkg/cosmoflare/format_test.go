package cosmoflare

import "testing"

func TestHumanCount(t *testing.T) {
	t.Parallel()
	for v, want := range map[float64]string{
		2945546702: "2.9B", 52332502: "52.3M", 3289: "3.3k", 522: "522", 0: "0",
	} {
		if got := HumanCount(v); got != want {
			t.Errorf("HumanCount(%v) = %q, want %q", v, got, want)
		}
	}
}
