package domain

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestFrequencyConstants(t *testing.T) {
	tests := []struct {
		name      string
		frequency Frequency
		expected  string
	}{
		{"monthly frequency", FrequencyMonthly, "monthly"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.frequency) != tt.expected {
				t.Errorf("Frequency constant %s = %s, want %s", tt.name, tt.frequency, tt.expected)
			}
		})
	}
}

func TestRecurringTemplateEndDateNullable(t *testing.T) {
	// Verify EndDate can be nil (runs forever)
	template := RecurringTemplate{
		ID:          1,
		WorkspaceID: 1,
		Description: "Test Template",
		Amount:      decimal.NewFromInt(100),
		CategoryID:  1,
		AccountID:   1,
		Frequency:   "monthly",
		StartDate:   time.Now(),
		EndDate:     nil, // NULL - runs forever
	}

	if template.EndDate != nil {
		t.Errorf("Expected EndDate to be nil, got %v", template.EndDate)
	}
}

func TestRecurringTemplateWithEndDate(t *testing.T) {
	// Verify EndDate can be set
	endDate := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	template := RecurringTemplate{
		ID:          1,
		WorkspaceID: 1,
		Description: "Test Template",
		Amount:      decimal.NewFromInt(100),
		CategoryID:  1,
		AccountID:   1,
		Frequency:   "monthly",
		StartDate:   time.Now(),
		EndDate:     &endDate,
	}

	if template.EndDate == nil {
		t.Error("Expected EndDate to be set, got nil")
	}
	if !template.EndDate.Equal(endDate) {
		t.Errorf("Expected EndDate to be %v, got %v", endDate, *template.EndDate)
	}
}

func TestCreateRecurringTemplateInputValidation(t *testing.T) {
	// Test that input struct can be created with all required fields
	input := CreateRecurringTemplateInput{
		WorkspaceID: 1,
		Description: "Monthly Rent",
		Amount:      decimal.NewFromInt(1500),
		CategoryID:  5,
		AccountID:   2,
		Frequency:   "monthly",
		StartDate:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:     nil,
	}

	if input.WorkspaceID != 1 {
		t.Errorf("Expected WorkspaceID 1, got %d", input.WorkspaceID)
	}
	if input.Description != "Monthly Rent" {
		t.Errorf("Expected Description 'Monthly Rent', got %s", input.Description)
	}
	if !input.Amount.Equal(decimal.NewFromInt(1500)) {
		t.Errorf("Expected Amount 1500, got %s", input.Amount.String())
	}
}

func TestUpdateRecurringTemplateInputValidation(t *testing.T) {
	endDate := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)
	input := UpdateRecurringTemplateInput{
		Description: "Updated Rent",
		Amount:      decimal.NewFromInt(1600),
		CategoryID:  5,
		AccountID:   2,
		Frequency:   "monthly",
		StartDate:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:     &endDate,
	}

	if input.Description != "Updated Rent" {
		t.Errorf("Expected Description 'Updated Rent', got %s", input.Description)
	}
	if input.EndDate == nil {
		t.Error("Expected EndDate to be set, got nil")
	}
}
