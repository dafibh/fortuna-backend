package domain

import (
	"testing"
)

func TestTransactionGroupValidate(t *testing.T) {
	tests := []struct {
		name    string
		group   TransactionGroup
		wantErr error
	}{
		{
			name:    "valid group",
			group:   TransactionGroup{Name: "Groceries", Month: "2026-01"},
			wantErr: nil,
		},
		{
			name:    "valid group december",
			group:   TransactionGroup{Name: "Year End", Month: "2026-12"},
			wantErr: nil,
		},
		{
			name:    "empty name",
			group:   TransactionGroup{Name: "", Month: "2026-01"},
			wantErr: ErrGroupNameEmpty,
		},
		{
			name:    "invalid month format - no dash",
			group:   TransactionGroup{Name: "Test", Month: "202601"},
			wantErr: ErrInvalidMonthFormat,
		},
		{
			name:    "invalid month format - full date",
			group:   TransactionGroup{Name: "Test", Month: "2026-01-15"},
			wantErr: ErrInvalidMonthFormat,
		},
		{
			name:    "invalid month format - month 00",
			group:   TransactionGroup{Name: "Test", Month: "2026-00"},
			wantErr: ErrInvalidMonthFormat,
		},
		{
			name:    "invalid month format - month 13",
			group:   TransactionGroup{Name: "Test", Month: "2026-13"},
			wantErr: ErrInvalidMonthFormat,
		},
		{
			name:    "invalid month format - empty",
			group:   TransactionGroup{Name: "Test", Month: ""},
			wantErr: ErrInvalidMonthFormat,
		},
		{
			name:    "invalid month format - text",
			group:   TransactionGroup{Name: "Test", Month: "January"},
			wantErr: ErrInvalidMonthFormat,
		},
		{
			name:    "invalid month format - short year",
			group:   TransactionGroup{Name: "Test", Month: "26-01"},
			wantErr: ErrInvalidMonthFormat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.group.Validate()
			if err != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTransactionGroupErrorMessages(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{"group not found", ErrGroupNotFound, "transaction group not found"},
		{"group name empty", ErrGroupNameEmpty, "group name cannot be empty"},
		{"invalid month format", ErrInvalidMonthFormat, "month must be in YYYY-MM format"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.expected {
				t.Errorf("error message = %q, want %q", tt.err.Error(), tt.expected)
			}
		})
	}
}

func TestTransactionGroupDefaults(t *testing.T) {
	group := TransactionGroup{}
	if group.AutoDetected != false {
		t.Errorf("Expected default AutoDetected to be false, got %v", group.AutoDetected)
	}
	if group.LoanProviderID != nil {
		t.Errorf("Expected default LoanProviderID to be nil, got %v", group.LoanProviderID)
	}
	if group.ChildCount != 0 {
		t.Errorf("Expected default ChildCount to be 0, got %d", group.ChildCount)
	}
	if !group.TotalAmount.IsZero() {
		t.Errorf("Expected default TotalAmount to be zero, got %s", group.TotalAmount.String())
	}
}
