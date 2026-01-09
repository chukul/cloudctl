package internal

import "time"

const (
	// DisplayTimeFormat is the standard time format used across the application
	DisplayTimeFormat = "2006-01-02 15:04:05"
	// LogTimeFormat is the short time format used in daemon logs
	LogTimeFormat = "15:04:05"
)

// BangkokLocation is the fixed timezone for Asia/Bangkok (UTC+7)
var BangkokLocation = time.FixedZone("Asia/Bangkok", 7*60*60)

// InBKK returns the given time converted to Bangkok time zone
func InBKK(t time.Time) time.Time {
	return t.In(BangkokLocation)
}

// FormatBKK formats the given time in the standard display format (BKK time)
func FormatBKK(t time.Time) string {
	return InBKK(t).Format(DisplayTimeFormat)
}

// FormatBKKLog formats the given time in the short log format (BKK time)
func FormatBKKLog(t time.Time) string {
	return InBKK(t).Format(LogTimeFormat)
}
