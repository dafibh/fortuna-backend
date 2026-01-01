package util

import (
	"testing"
	"time"
)

func TestPreviousMonth_SameYear(t *testing.T) {
	tests := []struct {
		year      int
		month     int
		wantYear  int
		wantMonth int
	}{
		{2026, 6, 2026, 5},   // June -> May
		{2026, 12, 2026, 11}, // Dec -> Nov
		{2026, 2, 2026, 1},   // Feb -> Jan
	}

	for _, tt := range tests {
		gotYear, gotMonth := PreviousMonth(tt.year, tt.month)
		if gotYear != tt.wantYear || gotMonth != tt.wantMonth {
			t.Errorf("PreviousMonth(%d, %d) = (%d, %d), want (%d, %d)",
				tt.year, tt.month, gotYear, gotMonth, tt.wantYear, tt.wantMonth)
		}
	}
}

func TestPreviousMonth_YearBoundary(t *testing.T) {
	// January -> December of previous year
	gotYear, gotMonth := PreviousMonth(2026, 1)
	if gotYear != 2025 || gotMonth != 12 {
		t.Errorf("PreviousMonth(2026, 1) = (%d, %d), want (2025, 12)", gotYear, gotMonth)
	}
}

func TestIsHistoricalMonth(t *testing.T) {
	now := time.Now()
	currentYear := now.Year()
	currentMonth := int(now.Month())

	tests := []struct {
		name     string
		year     int
		month    int
		expected bool
	}{
		{
			name:     "current month is not historical",
			year:     currentYear,
			month:    currentMonth,
			expected: false,
		},
		{
			name:     "previous month is historical",
			year:     currentYear,
			month:    currentMonth - 1,
			expected: currentMonth > 1, // Only true if we're not in January
		},
		{
			name:     "previous year same month is historical",
			year:     currentYear - 1,
			month:    currentMonth,
			expected: true,
		},
		{
			name:     "previous year is historical",
			year:     currentYear - 1,
			month:    12,
			expected: true,
		},
		{
			name:     "future month is not historical",
			year:     currentYear,
			month:    currentMonth + 1,
			expected: false,
		},
		{
			name:     "next year is not historical",
			year:     currentYear + 1,
			month:    1,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip test cases that don't make sense due to month boundaries
			if tt.month < 1 || tt.month > 12 {
				t.Skip("Invalid month for test case")
			}
			got := IsHistoricalMonth(tt.year, tt.month)
			if got != tt.expected {
				t.Errorf("IsHistoricalMonth(%d, %d) = %v, want %v",
					tt.year, tt.month, got, tt.expected)
			}
		})
	}
}

func TestIsHistoricalMonth_YearBoundary(t *testing.T) {
	// December of previous year should always be historical
	now := time.Now()
	got := IsHistoricalMonth(now.Year()-1, 12)
	if !got {
		t.Errorf("IsHistoricalMonth(%d, 12) = false, want true", now.Year()-1)
	}

	// January of next year should never be historical
	got = IsHistoricalMonth(now.Year()+1, 1)
	if got {
		t.Errorf("IsHistoricalMonth(%d, 1) = true, want false", now.Year()+1)
	}
}
