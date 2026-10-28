package backgroundtasks

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// Each event captures one endpoint's project and immutable relation label. It
// remains meaningful after a definition rename or a related item deletion.
func relationActivityRows(input IssueActivityInput) ([]issueActivityRow, error) {
	var snapshot struct {
		RelatedID  string `json:"related_id"`
		Label      string `json:"label"`
		Identifier string `json:"identifier"`
	}
	if input.IssueID == nil || json.Unmarshal(input.RequestedData, &snapshot) != nil || snapshot.Label == "" || snapshot.Identifier == "" {
		return nil, fmt.Errorf("%w: invalid relation snapshot", ErrPermanentIssueActivity)
	}
	if id, err := uuid.Parse(snapshot.RelatedID); err != nil || id == uuid.Nil {
		return nil, fmt.Errorf("%w: invalid related identifier", ErrPermanentIssueActivity)
	}
	verb := strings.TrimPrefix(input.Type, "workitem_relation.activity.")
	row := issueActivityRow{IssueID: input.IssueID, ActorID: input.ActorID, Epoch: input.Epoch, Verb: verb, Field: snapshot.Label, Comment: verb + " " + snapshot.Label + " relation"}
	if verb == "created" {
		row.NewIdentifier = &snapshot.RelatedID
		row.NewValue = &snapshot.Identifier
	} else {
		row.OldIdentifier = &snapshot.RelatedID
		row.OldValue = &snapshot.Identifier
	}
	return []issueActivityRow{row}, nil
}
