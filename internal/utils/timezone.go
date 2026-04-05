// Package utils provides shared, stateless helper functions for the clinic backend.
// Helpers here must have no dependencies on any internal domain package — they sit
// at the lowest layer of the dependency graph and may be imported by anyone.
package utils

import "time"

// ToUTC converts t (expressed in the given IANA timezone) to its UTC equivalent.
// If tz is empty or unrecognised, UTC is assumed and t is returned as-is in UTC.
//
// Usage:
//
//	localNoon := time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC) // pretend local
//	utc := utils.ToUTC(localNoon, "Asia/Baghdad")              // => 09:00 UTC
func ToUTC(t time.Time, tz string) time.Time {
	loc := loadLocation(tz)
	// Re-interpret t in the given location, then shift to UTC.
	local := time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), loc)
	return local.UTC()
}

// FromUTC converts t (which must be in UTC) to the wall-clock time in the given
// IANA timezone. If tz is empty or unrecognised, UTC is returned unchanged.
//
// Usage:
//
//	utc := time.Now().UTC()
//	local := utils.FromUTC(utc, "Asia/Baghdad") // => UTC+3
func FromUTC(t time.Time, tz string) time.Time {
	loc := loadLocation(tz)
	return t.In(loc)
}

// NowIn returns the current wall-clock time in the given IANA timezone.
// Equivalent to time.Now().UTC() then FromUTC, but expressed as a single call.
func NowIn(tz string) time.Time {
	return FromUTC(time.Now().UTC(), tz)
}

// loadLocation resolves an IANA timezone name to a *time.Location.
// Falls back to time.UTC on empty string or unrecognised identifier so that
// callers never need to handle a nil location.
func loadLocation(tz string) *time.Location {
	if tz == "" {
		return time.UTC
	}
	loc, err := time.LoadLocation(tz)
	if err != nil || loc == nil {
		return time.UTC
	}
	return loc
}
