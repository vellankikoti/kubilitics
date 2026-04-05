package events

import (
	"context"
	"fmt"
	"log"
	"time"
)

// SnapshotManager captures periodic cluster state.
type SnapshotManager struct {
	store     *Store
	clusterID string
	stopCh    chan struct{}
}

// NewSnapshotManager creates a new SnapshotManager.
func NewSnapshotManager(store *Store, clusterID string) *SnapshotManager {
	return &SnapshotManager{
		store:     store,
		clusterID: clusterID,
		stopCh:    make(chan struct{}),
	}
}

// Start takes a snapshot every interval (default 5 min) in a goroutine.
// The provided context is used for all database operations; cancelling it
// causes the goroutine to exit.
func (sm *SnapshotManager) Start(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 5 * time.Minute
	}

	go func() {
		// Take an initial snapshot immediately.
		if err := sm.TakeSnapshot(ctx); err != nil {
			log.Printf("events/snapshots: initial snapshot failed: %v", err)
		}

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-sm.stopCh:
				return
			case <-ticker.C:
				if err := sm.TakeSnapshot(ctx); err != nil {
					log.Printf("events/snapshots: snapshot failed: %v", err)
				}
			}
		}
	}()
}

// Stop stops the snapshot manager.
func (sm *SnapshotManager) Stop() {
	close(sm.stopCh)
}

// TakeSnapshot captures the current cluster state as a snapshot.
// Fields are initialized to zero; they will be populated from real K8s data
// when wired to the pipeline.
func (sm *SnapshotManager) TakeSnapshot(ctx context.Context) error {
	snap := &StateSnapshot{
		SnapshotID:       fmt.Sprintf("snap_%d", time.Now().UnixMilli()),
		Timestamp:        UnixMillis(),
		ClusterID:        sm.clusterID,
		TotalPods:        0,
		RunningPods:      0,
		TotalNodes:       0,
		ReadyNodes:       0,
		HealthScore:      0,
		SPOFCount:        0,
		WarningEvents:    0,
		ErrorEvents:      0,
		NamespaceStates:  JSONText("[]"),
		DeploymentStates: JSONText("[]"),
	}

	if err := sm.store.InsertSnapshot(ctx, snap); err != nil {
		return fmt.Errorf("take snapshot: %w", err)
	}

	return nil
}

// GetStateAt retrieves the nearest snapshot at or before the given timestamp.
func (sm *SnapshotManager) GetStateAt(ctx context.Context, timestamp int64) (*StateSnapshot, error) {
	return sm.store.GetSnapshotAt(ctx, sm.clusterID, timestamp)
}
