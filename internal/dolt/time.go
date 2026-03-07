package dolt

import "time"

// doltTimeFormats lists the time formats dolt may use in JSON output.
var doltTimeFormats = []string{
	"2006-01-02 15:04:05.000000",
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05.000000Z",
	"2006-01-02T15:04:05Z",
	time.RFC3339,
}

// parseDoltTime attempts to parse a dolt timestamp string.
// Returns zero time if parsing fails.
func parseDoltTime(s string) time.Time {
	for _, format := range doltTimeFormats {
		if t, err := time.Parse(format, s); err == nil {
			return t
		}
	}
	return time.Time{}
}
