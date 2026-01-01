package util

import "time"

// PreviousMonth returns the year and month for the previous month
func PreviousMonth(year, month int) (int, int) {
	if month == 1 {
		return year - 1, 12
	}
	return year, month - 1
}

// IsHistoricalMonth returns true if the given year/month is before the current month
func IsHistoricalMonth(year, month int) bool {
	now := time.Now()
	currentYear := now.Year()
	currentMonth := int(now.Month())

	if year < currentYear {
		return true
	}
	if year == currentYear && month < currentMonth {
		return true
	}
	return false
}
