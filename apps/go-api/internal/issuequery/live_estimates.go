package issuequery

import "fmt"

// LiveNumericEstimatePredicate resolves a numeric point in the issue's own
// catalogue, without waiting for asynchronous deletion of dependent rows.
// Aliases are internal SQL identifiers, never request input.
func LiveNumericEstimatePredicate(issueAlias, pointAlias string) string {
	return fmt.Sprintf(`%[2]s.id = %[1]s.estimate_point_id
  AND %[2]s.deleted_at IS NULL
  AND %[2]s.project_id = %[1]s.project_id
  AND %[2]s.workspace_id = %[1]s.workspace_id
  AND EXISTS (
    SELECT 1 FROM estimates live_estimate
    WHERE live_estimate.id = %[2]s.estimate_id
      AND live_estimate.deleted_at IS NULL
      AND live_estimate.project_id = %[1]s.project_id
      AND live_estimate.workspace_id = %[1]s.workspace_id
      AND live_estimate.type = 'points'
  )`, issueAlias, pointAlias)
}
