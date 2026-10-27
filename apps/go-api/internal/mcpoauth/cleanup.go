package mcpoauth

import (
	"context"
	"time"
)

// RunCleanup removes expired proofs in bounded batches until shutdown. Callers
// own its lifetime and wait for it before closing the primary database pool.
// Authorization always checks expiry independently of this housekeeping task.
func (s *Links) RunCleanup(ctx context.Context, report func(error)) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		if ctx.Err() != nil {
			return
		}
		attempt, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := s.PruneExpired(attempt)
		cancel()
		if err != nil && ctx.Err() == nil && report != nil {
			report(ErrUnavailable)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
