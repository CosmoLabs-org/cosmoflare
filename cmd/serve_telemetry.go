package cmd

import (
	"fmt"
	"strings"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// serveZoneD1Warned holds the last zone/d1 warning logged so the serve loop
// logs it once per distinct rule set instead of every tick.
var serveZoneD1Warned string

// serveZoneD1Warning returns the log line serve emits when enabled rules use
// zone/d1 conditions: serve does not collect that telemetry (FEAT-049 O17),
// so those rules are evaluated by 'alerts watch' and 'alerts check' only.
// Empty when no enabled rule is affected.
func serveZoneD1Warning(rules []*cosmoflare.AlertRule) string {
	var names []string
	for _, r := range rules {
		if r == nil || !r.Enabled {
			continue
		}
		if d, ok := cosmoflare.LookupAlertCondition(r.Condition); ok && (d.Scope == "zone" || d.Scope == "d1") {
			names = append(names, r.Name)
		}
	}
	if len(names) == 0 {
		return ""
	}
	return fmt.Sprintf("[alerts] zone/d1 rules (%s) are evaluated by 'alerts watch' and 'alerts check' only; serve skips them", strings.Join(names, ", "))
}
