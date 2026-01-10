package domain

import "time"

// ProjectionExclusion tracks explicitly deleted projected transactions
// to prevent them from being regenerated on sync
type ProjectionExclusion struct {
	ID            int32     `json:"id"`
	WorkspaceID   int32     `json:"workspaceId"`
	TemplateID    int32     `json:"templateId"`
	ExcludedMonth time.Time `json:"excludedMonth"` // First day of excluded month
	CreatedAt     time.Time `json:"createdAt"`
}

// ProjectionExclusionRepository defines operations for projection exclusions
type ProjectionExclusionRepository interface {
	// Create creates a new exclusion record (idempotent)
	Create(workspaceID int32, templateID int32, excludedMonth time.Time) error

	// IsExcluded checks if a specific month is excluded for a template
	IsExcluded(workspaceID int32, templateID int32, excludedMonth time.Time) (bool, error)

	// DeleteByTemplate removes all exclusions for a template (used when template is deleted)
	DeleteByTemplate(templateID int32) error

	// GetByTemplate gets all exclusions for a template
	GetByTemplate(workspaceID int32, templateID int32) ([]*ProjectionExclusion, error)
}
