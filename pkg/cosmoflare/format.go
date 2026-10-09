package cosmoflare

import (
	"fmt"
	"strconv"
)

// HumanCount abbreviates a count to one decimal with a k/M/B suffix
// (2.9B, 52.3M, 3.3k); values under 1000 print as plain numbers. Shared by
// alert pages and the usage view so the tiers cannot drift apart.
func HumanCount(v float64) string {
	switch {
	case v >= 1e9:
		return fmt.Sprintf("%.1fB", v/1e9)
	case v >= 1e6:
		return fmt.Sprintf("%.1fM", v/1e6)
	case v >= 1e3:
		return fmt.Sprintf("%.1fk", v/1e3)
	default:
		return strconv.FormatFloat(v, 'f', -1, 64)
	}
}
