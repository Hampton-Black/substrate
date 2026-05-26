package server

import "time"

func relativeTime(t time.Time) string {
	now := time.Now().UTC()
	diff := now.Sub(t.UTC())

	if diff < time.Minute {
		return "just now"
	}
	if diff < time.Hour {
		return formatUnit(int(diff.Minutes()), "m")
	}
	if diff < 24*time.Hour {
		return formatUnit(int(diff.Hours()), "h")
	}
	return formatUnit(int(diff.Hours()/24), " days")
}

func formatUnit(n int, unit string) string {
	if n == 1 && unit == " days" {
		return "1 day ago"
	}
	return itoa(n) + unit + " ago"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
