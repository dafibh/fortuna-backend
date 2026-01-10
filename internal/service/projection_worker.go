package service

import (
	"context"
	"sync"
	"time"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/rs/zerolog"
)

// ProjectionWorker is a background worker that periodically generates projections
type ProjectionWorker struct {
	projectionService *ProjectionService
	workspaceRepo     domain.WorkspaceRepository
	logger            zerolog.Logger
	interval          time.Duration
	monthsAhead       int
	stopCh            chan struct{}
	doneCh            chan struct{}
	mu                sync.Mutex
	running           bool
}

// ProjectionWorkerConfig holds configuration for the projection worker
type ProjectionWorkerConfig struct {
	Interval    time.Duration // How often to run projection sync
	MonthsAhead int           // How many months ahead to project
}

// DefaultProjectionWorkerConfig returns sensible defaults
func DefaultProjectionWorkerConfig() ProjectionWorkerConfig {
	return ProjectionWorkerConfig{
		Interval:    1 * time.Hour, // Run every hour
		MonthsAhead: 12,            // Project 12 months ahead
	}
}

// NewProjectionWorker creates a new projection worker
func NewProjectionWorker(
	projectionService *ProjectionService,
	workspaceRepo domain.WorkspaceRepository,
	logger zerolog.Logger,
	config ProjectionWorkerConfig,
) *ProjectionWorker {
	if config.Interval <= 0 {
		config.Interval = 1 * time.Hour
	}
	if config.MonthsAhead <= 0 {
		config.MonthsAhead = DefaultProjectionMonths
	}

	return &ProjectionWorker{
		projectionService: projectionService,
		workspaceRepo:     workspaceRepo,
		logger:            logger.With().Str("component", "projection_worker").Logger(),
		interval:          config.Interval,
		monthsAhead:       config.MonthsAhead,
		stopCh:            make(chan struct{}),
		doneCh:            make(chan struct{}),
	}
}

// Start begins the background projection sync
func (w *ProjectionWorker) Start(ctx context.Context) {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return
	}
	w.running = true
	w.mu.Unlock()

	w.logger.Info().
		Dur("interval", w.interval).
		Int("months_ahead", w.monthsAhead).
		Msg("Starting projection worker")

	go w.run(ctx)
}

// Stop gracefully stops the projection worker
func (w *ProjectionWorker) Stop() {
	w.mu.Lock()
	if !w.running {
		w.mu.Unlock()
		return
	}
	w.mu.Unlock()

	w.logger.Info().Msg("Stopping projection worker")
	close(w.stopCh)
	<-w.doneCh
	w.logger.Info().Msg("Projection worker stopped")
}

// run is the main loop for the projection worker
func (w *ProjectionWorker) run(ctx context.Context) {
	defer close(w.doneCh)

	// Run immediately on startup
	w.syncAllWorkspaces(ctx)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.mu.Lock()
			w.running = false
			w.mu.Unlock()
			return
		case <-w.stopCh:
			w.mu.Lock()
			w.running = false
			w.mu.Unlock()
			return
		case <-ticker.C:
			w.syncAllWorkspaces(ctx)
		}
	}
}

// syncAllWorkspaces generates projections for all workspaces
func (w *ProjectionWorker) syncAllWorkspaces(ctx context.Context) {
	w.logger.Debug().Msg("Starting projection sync for all workspaces")
	startTime := time.Now()

	// Get all workspaces
	workspaces, err := w.workspaceRepo.GetAllWorkspaces()
	if err != nil {
		w.logger.Error().Err(err).Msg("Failed to get workspaces for projection sync")
		return
	}

	totalGenerated := 0
	totalSkipped := 0
	totalErrors := 0

	for _, ws := range workspaces {
		select {
		case <-ctx.Done():
			w.logger.Info().Msg("Context cancelled, stopping sync")
			return
		case <-w.stopCh:
			w.logger.Info().Msg("Stop signal received, stopping sync")
			return
		default:
		}

		result, err := w.projectionService.GenerateProjections(ws.ID, w.monthsAhead)
		if err != nil {
			w.logger.Error().
				Err(err).
				Int32("workspace_id", ws.ID).
				Msg("Failed to generate projections for workspace")
			totalErrors++
			continue
		}

		totalGenerated += result.Generated
		totalSkipped += result.Skipped
		totalErrors += len(result.Errors)

		if result.Generated > 0 {
			w.logger.Debug().
				Int32("workspace_id", ws.ID).
				Int("generated", result.Generated).
				Int("skipped", result.Skipped).
				Msg("Generated projections for workspace")
		}
	}

	elapsed := time.Since(startTime)
	w.logger.Info().
		Int("workspaces", len(workspaces)).
		Int("total_generated", totalGenerated).
		Int("total_skipped", totalSkipped).
		Int("total_errors", totalErrors).
		Dur("elapsed", elapsed).
		Msg("Completed projection sync")
}

// SyncWorkspace manually triggers projection sync for a specific workspace
// This can be called when a recurring template is created/updated
func (w *ProjectionWorker) SyncWorkspace(workspaceID int32) (*ProjectionResult, error) {
	w.logger.Debug().Int32("workspace_id", workspaceID).Msg("Manual projection sync triggered")
	return w.projectionService.GenerateProjections(workspaceID, w.monthsAhead)
}

// IsRunning returns whether the worker is currently running
func (w *ProjectionWorker) IsRunning() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.running
}
