package cyclereads

import (
	"context"
	"strings"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
)

var apiAuditFields = strings.Fields("id created_at updated_at deleted_at created_by updated_by workspace project")
var apiCycleFields = strings.Fields("name description start_date end_date owned_by view_props sort_order external_source external_id progress_snapshot archived_at logo_props timezone version total_issues cancelled_issues completed_issues started_issues unstarted_issues backlog_issues total_estimates completed_estimates started_estimates")

func (r *Repository) apiSerialize(ctx context.Context, row map[string]any, kind, fields, expand string) (map[string]any, error) {
	if kind == "issue" {
		return djangoapi.SerializeIssue(ctx, r.database, row, fields, expand)
	}
	declared := append([]string{}, apiAuditFields...)
	if kind == "cycle" {
		declared = append(declared, apiCycleFields...)
	} else {
		declared = append(declared, "issue", "cycle", "sub_issues_count")
	}
	return djangoapi.Serialize(ctx, r.database, row, declared, fields, expand, false)
}
