package workitems

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/jackc/pgx/v5/pgconn"
)

type draftRelationDatabase struct {
	Database
	cleared []string
}

func (db *draftRelationDatabase) Exec(_ context.Context, query string, _ ...any) (pgconn.CommandTag, error) {
	words := strings.Fields(query)
	if len(words) < 2 || words[0] != "UPDATE" || !strings.Contains(query, "SET deleted_at=") {
		return pgconn.CommandTag{}, errors.New("unexpected draft mutation")
	}
	db.cleared = append(db.cleared, words[1])
	return pgconn.NewCommandTag("UPDATE 1"), nil
}

func TestDraftMalformedModulePreservesEarlierRelationChanges(t *testing.T) {
	db := &draftRelationDatabase{}
	r := &Repository{db: db, now: time.Now}
	input := djangoinput.Input{Fields: map[string]any{"cycle_id": nil, "module_ids": []any{"invalid"}}}
	err := r.replaceDraftRelations(context.Background(), Request{}, map[string]any{"id": "draft"}, issueChanges{assigneesSet: true, labelsSet: true}, input, false)
	var response *djangoapi.HTTPError
	if !errors.As(err, &response) || response.Status != 400 {
		t.Fatalf("module error=%v", err)
	}
	if got := strings.Join(db.cleared, ","); got != "draft_issue_assignees,draft_issue_labels,draft_issue_cycles,draft_issue_modules" {
		t.Fatalf("partial effects=%s", got)
	}
}
