package handler

import (
	"strings"
	"time"
)

// lastWord returns the last whitespace-separated token in s.
// Matches the Python: item["place_name"].split()[-1]
func lastWord(s string) string {
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return s
	}
	return fields[len(fields)-1]
}

// timeNowMillis returns current time in epoch milliseconds.
func timeNowMillis() int64 {
	return time.Now().UnixMilli()
}
