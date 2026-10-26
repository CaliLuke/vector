package workitems

import (
	"errors"
	"fmt"

	"github.com/CaliLuke/go-plane/internal/issuequery"
)

// BuildAppIssueFilters exposes the application's characterized GET filter
// translation for read repositories that aggregate the same issue queryset.
func BuildAppIssueFilters(filters map[string]string, present map[string]bool, where []string, args []any) ([]string, []any, error) {
	options := issuequery.Options{
		State: filters["state"], StateGroup: filters["state_group"], EstimatePoint: filters["estimate_point"],
		Priority: filters["priority"], Parent: filters["parent"], Labels: filters["labels"],
		Assignees: filters["assignees"], Mentions: filters["mentions"], CreatedBy: filters["created_by"],
		LoggedBy: filters["logged_by"], Name: filters["name"], CreatedAt: filters["created_at"],
		UpdatedAt: filters["updated_at"], StartDate: filters["start_date"], TargetDate: filters["target_date"],
		CompletedAt: filters["completed_at"], Type: filters["type"], Project: filters["project"],
		Cycle: filters["cycle"], Module: filters["module"], IntakeStatus: filters["intake_status"],
		InboxStatus: filters["inbox_status"], SubIssue: filters["sub_issue"], Subscriber: filters["subscriber"],
		StartTargetDate: filters["start_target_date"],
		AssigneesSet:    present["assignees"], LabelsSet: present["labels"], CycleSet: present["cycle"],
		ModuleSet: present["module"], SubscriberSet: present["subscriber"], TypeSet: present["type"],
	}
	return appFilters(options, where, args)
}

// App issue filters share the characterized pure filter algebra used by module lists.
func appFilters(options issuequery.Options, where []string, args []any) ([]string, []any, error) {
	return appFiltersWithTypes(options, where, args, nil)
}

func appFiltersWithTypes(options issuequery.Options, where []string, args []any, typeIDs map[string]bool) ([]string, []any, error) {
	estimatePoints, validEstimatePoints := issuequery.NormalizeLegacyUUIDList(options.EstimatePoint)
	if !validEstimatePoints {
		return nil, args, failure(400, "error", "Please provide valid detail")
	}
	for _, filter := range []string{options.CreatedAt, options.UpdatedAt, options.StartDate, options.TargetDate, options.CompletedAt} {
		if !issuequery.ValidLegacyDateBounds(filter) {
			return nil, args, failure(400, "error", "Please provide valid detail")
		}
	}
	intakeStatus, validIntakeStatus := issuequery.NormalizeLegacyIntegerList(options.IntakeStatus)
	inboxStatus, validInboxStatus := issuequery.NormalizeLegacyIntegerList(options.InboxStatus)
	if !validIntakeStatus || !validInboxStatus {
		return nil, args, fmt.Errorf("invalid intake status")
	}
	complexFilter, err := issuequery.BuildComplexFilterWithTypes(options.Filters, &args, typeIDs)
	if err != nil {
		return nil, args, err
	}
	if complexFilter != "" {
		where = append(where, complexFilter)
	}
	if options.LoggedBy != "" {
		return nil, args, errors.New("Issue has no logged_by field")
	}
	add := func(value, clause string) {
		if value == "" {
			return
		}
		args = append(args, value)
		where = append(where, fmt.Sprintf(clause, len(args)))
	}
	states, _ := issuequery.ValidUUIDFilter(options.State, false)
	for _, relation := range []struct {
		present bool
		table   string
	}{
		{options.AssigneesSet, "issue_assignees"},
		{options.CycleSet, "cycle_issues"},
		{options.ModuleSet, "module_issues"},
		{options.SubscriberSet, "issue_subscribers"},
	} {
		if relation.present {
			where = append(where, "(NOT EXISTS (SELECT 1 FROM "+relation.table+" presence_relation WHERE presence_relation.issue_id = issue.id) OR EXISTS (SELECT 1 FROM "+relation.table+" presence_relation WHERE presence_relation.issue_id = issue.id AND presence_relation.deleted_at IS NULL))")
		}
	}
	add(states, "issue.state_id::text = ANY(string_to_array($%d, ','))")
	add(issuequery.FilterNullTokens(options.StateGroup), `EXISTS (SELECT 1 FROM states filter_state WHERE filter_state.id = issue.state_id AND filter_state."group" = ANY(string_to_array($%d, ',')))`)
	add(estimatePoints, "issue.estimate_point_id::text = ANY(string_to_array($%d, ','))")
	add(issuequery.FilterNullTokens(options.Priority), "issue.priority = ANY(string_to_array($%d, ','))")
	assignees, unassigned := issuequery.ValidUUIDFilter(options.Assignees, true)
	if unassigned {
		where = append(where, "NOT EXISTS (SELECT 1 FROM issue_assignees filter_assignee WHERE filter_assignee.issue_id = issue.id)")
	}
	add(assignees, "EXISTS (SELECT 1 FROM issue_assignees filter_assignee WHERE filter_assignee.issue_id = issue.id AND filter_assignee.deleted_at IS NULL AND filter_assignee.assignee_id::text = ANY(string_to_array($%d, ',')))")
	labels, unlabeled := issuequery.ValidUUIDFilter(options.Labels, true)
	if unlabeled {
		where = append(where, "NOT EXISTS (SELECT 1 FROM issue_labels filter_label WHERE "+issuequery.LiveIssueLabelPredicate("issue", "filter_label")+")")
	}
	add(labels, "EXISTS (SELECT 1 FROM issue_labels filter_label WHERE "+issuequery.LiveIssueLabelPredicate("issue", "filter_label")+" AND filter_label.label_id::text = ANY(string_to_array($%d, ',')))")
	projects, _ := issuequery.ValidUUIDFilter(options.Project, false)
	add(projects, "issue.project_id::text = ANY(string_to_array($%d, ','))")
	loggedBy, anonymousLogger := issuequery.ValidUUIDFilter(options.LoggedBy, true)
	if anonymousLogger {
		where = append(where, "issue.logged_by_id IS NULL")
	}
	add(loggedBy, "issue.logged_by_id::text = ANY(string_to_array($%d, ','))")
	creators, anonymousCreator := issuequery.ValidUUIDFilter(options.CreatedBy, false)
	if anonymousCreator {
		where = append(where, "issue.created_by_id IS NULL")
	}
	add(creators, "issue.created_by_id::text = ANY(string_to_array($%d, ','))")
	parents, rootIssues := issuequery.ValidUUIDFilter(options.Parent, true)
	if rootIssues {
		where = append(where, "issue.parent_id IS NULL")
	}
	add(parents, "issue.parent_id::text = ANY(string_to_array($%d, ','))")
	mentions, _ := issuequery.ValidUUIDFilter(options.Mentions, false)
	add(mentions, "EXISTS (SELECT 1 FROM issue_mentions filter_mention WHERE filter_mention.issue_id = issue.id AND filter_mention.mention_id::text = ANY(string_to_array($%d, ',')))")
	if options.Name != "" {
		add("%"+issuequery.EscapeLikePattern(options.Name)+"%", `issue.name ILIKE $%d ESCAPE '\'`)
	}
	where, args = issuequery.AddDateFilters(where, args, options.CreatedAt, issuequery.RequesterDateSQL("issue.created_at"))
	where, args = issuequery.AddDateFilters(where, args, options.UpdatedAt, issuequery.RequesterDateSQL("issue.updated_at"))
	where, args = issuequery.AddDateFilters(where, args, options.StartDate, "issue.start_date")
	where, args = issuequery.AddDateFilters(where, args, options.TargetDate, "issue.target_date")
	where, args = issuequery.AddDateFilters(where, args, options.CompletedAt, issuequery.RequesterDateSQL("issue.completed_at"))
	cycles, noCycle := issuequery.ValidUUIDFilter(options.Cycle, true)
	if noCycle {
		where = append(where, "NOT EXISTS (SELECT 1 FROM cycle_issues filter_cycle WHERE filter_cycle.issue_id = issue.id)")
	}
	add(cycles, "EXISTS (SELECT 1 FROM cycle_issues filter_cycle WHERE filter_cycle.issue_id = issue.id AND filter_cycle.deleted_at IS NULL AND filter_cycle.cycle_id::text = ANY(string_to_array($%d, ',')))")
	modules, noModule := issuequery.ValidUUIDFilter(options.Module, true)
	if noModule {
		where = append(where, "NOT EXISTS (SELECT 1 FROM module_issues filter_module_relation WHERE filter_module_relation.issue_id = issue.id)")
	}
	add(modules, "EXISTS (SELECT 1 FROM module_issues filter_module_relation WHERE filter_module_relation.issue_id = issue.id AND filter_module_relation.deleted_at IS NULL AND filter_module_relation.module_id::text = ANY(string_to_array($%d, ',')))")
	if inboxStatus != "" {
		intakeStatus = inboxStatus
	}
	add(intakeStatus, "EXISTS (SELECT 1 FROM intake_issues filter_intake WHERE filter_intake.issue_id = issue.id AND filter_intake.status::text = ANY(string_to_array($%d, ',')))")
	if options.SubIssue == "false" {
		where = append(where, "issue.parent_id IS NULL")
	}
	subscribers, _ := issuequery.ValidUUIDFilter(options.Subscriber, false)
	add(subscribers, "EXISTS (SELECT 1 FROM issue_subscribers filter_subscriber WHERE filter_subscriber.issue_id = issue.id AND filter_subscriber.deleted_at IS NULL AND filter_subscriber.subscriber_id::text = ANY(string_to_array($%d, ',')))")
	if options.StartTargetDate == "true" {
		where = append(where, "issue.start_date IS NOT NULL", "issue.target_date IS NOT NULL")
	}
	if options.TypeSet {
		switch options.Type {
		case "backlog":
			where = append(where, `EXISTS (SELECT 1 FROM states type_state WHERE type_state.id = issue.state_id AND type_state."group" = 'backlog')`)
		case "active":
			where = append(where, `EXISTS (SELECT 1 FROM states type_state WHERE type_state.id = issue.state_id AND type_state."group" IN ('unstarted', 'started'))`)
		default:
			where = append(where, `EXISTS (SELECT 1 FROM states type_state WHERE type_state.id = issue.state_id AND type_state."group" IN ('backlog', 'unstarted', 'started', 'completed', 'cancelled'))`)
		}
	}

	return where, args, nil
}

// V2 has no legacy query-filter contract. Explicit structured filters must still
// be compiled (including errors), never silently ignored based on their fields.
func listFilterOptions(options issuequery.Options, v2 bool) issuequery.Options {
	if v2 {
		return issuequery.Options{Filters: options.Filters}
	}
	return options
}
