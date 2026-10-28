package backgroundtasks

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type workLogActivitySnapshot struct {
	ID          string `json:"id"`
	Duration    int64  `json:"duration"`
	Description string `json:"description"`
}

func workLogActivityRows(input IssueActivityInput) ([]issueActivityRow, error) {
	if input.IssueID == nil {
		return nil, fmt.Errorf("%w: work log item is required", ErrPermanentIssueActivity)
	}
	verb := strings.TrimPrefix(input.Type, "work_log.activity.")
	row := issueActivityRow{IssueID: input.IssueID, ActorID: input.ActorID, Epoch: input.Epoch, Verb: verb, Field: "work_log", Comment: verb + " a work log"}
	snapshot := func(raw json.RawMessage) (*string, *string, error) {
		var value workLogActivitySnapshot
		if json.Unmarshal(raw, &value) != nil || value.Duration <= 0 || value.Duration > 2147483647 {
			return nil, nil, fmt.Errorf("%w: invalid work log snapshot", ErrPermanentIssueActivity)
		}
		if id, err := uuid.Parse(value.ID); err != nil || id == uuid.Nil {
			return nil, nil, fmt.Errorf("%w: invalid work log identifier", ErrPermanentIssueActivity)
		}
		encoded, err := json.Marshal(value)
		if err != nil {
			return nil, nil, err
		}
		text := string(encoded)
		return &text, &value.ID, nil
	}
	var err error
	if verb != "created" {
		row.OldValue, row.OldIdentifier, err = snapshot(input.CurrentInstance)
		if err != nil {
			return nil, err
		}
	}
	if verb != "deleted" {
		row.NewValue, row.NewIdentifier, err = snapshot(input.RequestedData)
		if err != nil {
			return nil, err
		}
	}
	if row.OldIdentifier != nil && row.NewIdentifier != nil && *row.OldIdentifier != *row.NewIdentifier {
		return nil, fmt.Errorf("%w: work log identity changed", ErrPermanentIssueActivity)
	}
	return []issueActivityRow{row}, nil
}
