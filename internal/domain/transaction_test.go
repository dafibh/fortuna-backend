package domain

import (
	"testing"
)

func TestCCStateConstants(t *testing.T) {
	tests := []struct {
		name     string
		state    CCState
		expected string
	}{
		{"pending state", CCStatePending, "pending"},
		{"billed state", CCStateBilled, "billed"},
		{"settled state", CCStateSettled, "settled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.state) != tt.expected {
				t.Errorf("CCState constant %s = %s, want %s", tt.name, tt.state, tt.expected)
			}
		})
	}
}

func TestSettlementIntentConstants(t *testing.T) {
	tests := []struct {
		name     string
		intent   SettlementIntent
		expected string
	}{
		{"immediate intent", SettlementIntentImmediate, "immediate"},
		{"deferred intent", SettlementIntentDeferred, "deferred"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.intent) != tt.expected {
				t.Errorf("SettlementIntent constant %s = %s, want %s", tt.name, tt.intent, tt.expected)
			}
		})
	}
}

func TestCCStateValuesMatchDatabaseConstraints(t *testing.T) {
	// These values must match the CHECK constraint in the database:
	// CHECK (cc_state IN ('pending', 'billed', 'settled'))
	validStates := map[CCState]bool{
		CCStatePending: true,
		CCStateBilled:  true,
		CCStateSettled: true,
	}

	// Verify we have exactly 3 valid states
	if len(validStates) != 3 {
		t.Errorf("Expected 3 CCState values, got %d", len(validStates))
	}

	// Verify string values match database constraint
	dbConstraintValues := []string{"pending", "billed", "settled"}
	for _, dbVal := range dbConstraintValues {
		found := false
		for state := range validStates {
			if string(state) == dbVal {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Database constraint value %q not found in CCState constants", dbVal)
		}
	}
}

func TestSettlementIntentValuesMatchDatabaseConstraints(t *testing.T) {
	// These values must match the CHECK constraint in the database:
	// CHECK (settlement_intent IN ('immediate', 'deferred'))
	validIntents := map[SettlementIntent]bool{
		SettlementIntentImmediate: true,
		SettlementIntentDeferred:  true,
	}

	// Verify we have exactly 2 valid intents
	if len(validIntents) != 2 {
		t.Errorf("Expected 2 SettlementIntent values, got %d", len(validIntents))
	}

	// Verify string values match database constraint
	dbConstraintValues := []string{"immediate", "deferred"}
	for _, dbVal := range dbConstraintValues {
		found := false
		for intent := range validIntents {
			if string(intent) == dbVal {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Database constraint value %q not found in SettlementIntent constants", dbVal)
		}
	}
}

func TestTransactionDefaultSource(t *testing.T) {
	// Verify default source value
	tx := Transaction{}
	if tx.Source != "" {
		t.Errorf("Expected default Source to be empty string, got %q", tx.Source)
	}
}

func TestTransactionIsProjectedDefault(t *testing.T) {
	// Verify default IsProjected value
	tx := Transaction{}
	if tx.IsProjected != false {
		t.Errorf("Expected default IsProjected to be false, got %v", tx.IsProjected)
	}
}
