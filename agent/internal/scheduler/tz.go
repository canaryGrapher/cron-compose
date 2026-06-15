package scheduler

import "time"

// loadLocation returns the IANA timezone or UTC if the name is unknown.
func loadLocation(name string) *time.Location {
	if name == "" {
		return time.UTC
	}
	if loc, err := time.LoadLocation(name); err == nil {
		return loc
	}
	return time.UTC
}
