package backgroundtasks

import (
	"strings"
	"testing"
	"time"
)

func TestNormalizeIssueSyncAdmission(t *testing.T) {
	t.Run("defaults omitted values", func(t *testing.T) {
		batchSize, offset, countdown, err := normalizeIssueSyncAdmission("issue description sync", 0, 7, 0)
		if err != nil || batchSize != issueSyncBatchDefault || offset != 7 || countdown != issueSyncDelayDefault {
			t.Fatalf("got batch=%d offset=%d countdown=%s err=%v", batchSize, offset, countdown, err)
		}
	})

	t.Run("caps oversized batches", func(t *testing.T) {
		batchSize, offset, countdown, err := normalizeIssueSyncAdmission("issue version sync", 8000, 2, time.Minute)
		if err != nil || batchSize != issueSyncBatchLimit || offset != 2 || countdown != time.Minute {
			t.Fatalf("got batch=%d offset=%d countdown=%s err=%v", batchSize, offset, countdown, err)
		}
	})

	for _, tc := range []struct {
		name      string
		batchSize int
		offset    int
		countdown time.Duration
	}{
		{name: "negative offset", batchSize: 1, offset: -1},
		{name: "negative batch size", batchSize: -1},
		{name: "negative countdown", batchSize: 1, countdown: -time.Second},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, _, _, err := normalizeIssueSyncAdmission("issue version sync", tc.batchSize, tc.offset, tc.countdown)
			if err == nil || !strings.Contains(err.Error(), "issue version sync requires") {
				t.Fatalf("error=%v", err)
			}
		})
	}
}
