package backgroundtasks

import (
	"fmt"
	"time"
)

const (
	issueSyncBatchDefault = 5000
	issueSyncBatchLimit   = 5000
	issueSyncDelayDefault = 5 * time.Minute
)

func normalizeIssueSyncAdmission(kind string, batchSize, offset int, countdown time.Duration) (int, int, time.Duration, error) {
	if batchSize == 0 {
		batchSize = issueSyncBatchDefault
	}
	if batchSize > issueSyncBatchLimit {
		batchSize = issueSyncBatchLimit
	}
	if offset < 0 || batchSize < 1 || countdown < 0 {
		return 0, 0, 0, fmt.Errorf("%s requires non-negative offset and countdown and positive batch size", kind)
	}
	if countdown == 0 {
		countdown = issueSyncDelayDefault
	}
	return batchSize, offset, countdown, nil
}
