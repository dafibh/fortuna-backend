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

// CalculateActualDate returns the actual date for a target day in a given month,
// handling months with fewer days (e.g., day 31 in February returns Feb 28/29)
func CalculateActualDate(year int, month time.Month, targetDay int) time.Time {
	// Get last day of month by going to day 0 of next month
	lastDay := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()

	actualDay := targetDay
	if actualDay > lastDay {
		actualDay = lastDay
	}

	return time.Date(year, month, actualDay, 0, 0, 0, 0, time.UTC)
}
