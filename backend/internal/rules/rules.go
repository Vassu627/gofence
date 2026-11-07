package rules

import "sort"

// Rule is an active alert configuration.
// A nil VehicleID means the rule applies to every vehicle.
type Rule struct {
	ID         string
	GeofenceID string
	VehicleID  *string
	EventType  string
}

// Matches reports whether the rule should fire for this vehicle and event.
func Matches(rule Rule, vehicleID, eventType string) bool {
	if rule.VehicleID != nil && *rule.VehicleID != vehicleID {
		return false
	}
	return rule.EventType == eventType || rule.EventType == "both"
}

// Diff compares the geofence ids a vehicle occupied before and after a fix.
// Ids present only in curr are entries. Ids present only in prev are exits.
func Diff(prev, curr []string) (entered, exited []string) {
	previous := make(map[string]struct{}, len(prev))
	for _, id := range prev {
		previous[id] = struct{}{}
	}
	current := make(map[string]struct{}, len(curr))
	for _, id := range curr {
		current[id] = struct{}{}
	}
	for id := range current {
		if _, ok := previous[id]; !ok {
			entered = append(entered, id)
		}
	}
	for id := range previous {
		if _, ok := current[id]; !ok {
			exited = append(exited, id)
		}
	}
	sort.Strings(entered)
	sort.Strings(exited)
	if entered == nil {
		entered = []string{}
	}
	if exited == nil {
		exited = []string{}
	}
	return entered, exited
}
