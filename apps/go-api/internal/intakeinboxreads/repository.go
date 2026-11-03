// Package intakeinboxreads contains the read-only draft, intake, inbox, and description-version boundary.
package intakeinboxreads

import (
	"context"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/CaliLuke/go-plane/internal/apicontract"
	"github.com/CaliLuke/go-plane/internal/datetime"
	"github.com/CaliLuke/go-plane/internal/issuequery"

	"uuid"

	"github.com/jackc/pgx/v5"
)

var (
	ErrForbidden       = errors.New("active membership with an allowed role required")
	ErrGuestRestricted = errors.New("guest cannot read another creator's intake issue")
	ErrNotFound        = errors.New("intake or work item not found")
	ErrBadRequest      = errors.New("invalid filter parameter")
	ErrInvalidPerPage  = errors.New("invalid per_page parameter")
	ErrMissingIntake   = errors.New("active project intake required")
	ErrInvalidCursor   = errors.New("invalid cursor parameter")
	ErrInvalidDetail   = errors.New("invalid filter detail")
)

type database interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

type Repository struct{ database database }

type ListOptions struct {
	Cursor         string
	CursorSet      bool
	PerPage        string
	PerPageSet     bool
	OrderBy        string
	Timezone       string
	Filters        map[string]string
	StatusSet      bool
	PresentFilters map[string]bool
}

func NewRepository(database database) (*Repository, error) {
	if database == nil {
		return nil, fmt.Errorf("intake and inbox reads database is required")
	}
	return &Repository{database: database}, nil
}

func (repository *Repository) DraftIssueList(ctx context.Context, userID uuid.UUID, slug string, options ListOptions) (jsontext.Value, error) {
	return repository.draftIssueList(ctx, userID, slug, options)
}

func (repository *Repository) DraftIssueDetail(ctx context.Context, userID uuid.UUID, slug string, draftID uuid.UUID) (jsontext.Value, error) {
	if err := repository.authorizeDraftIssueDetail(ctx, userID, slug, draftID); err != nil {
		return nil, err
	}
	var data []byte
	err := repository.database.QueryRow(ctx, draftIssueDetailSQL, userID, slug, draftID).Scan(&data)
	return scannedJSON(data, err, "get draft issue")
}

func (repository *Repository) InboxIssueList(ctx context.Context, userID uuid.UUID, slug string, projectID uuid.UUID, options ListOptions) (jsontext.Value, error) {
	return repository.intakeIssueList(ctx, userID, slug, projectID, options)
}

func (repository *Repository) InboxIssueDetail(ctx context.Context, userID uuid.UUID, slug string, projectID, issueID uuid.UUID) (jsontext.Value, error) {
	return repository.intakeIssueDetail(ctx, userID, slug, projectID, issueID)
}

func (repository *Repository) InboxList(ctx context.Context, userID uuid.UUID, slug string, projectID uuid.UUID) (jsontext.Value, error) {
	return repository.intakeContainerList(ctx, userID, slug, projectID)
}

func (repository *Repository) InboxDetail(ctx context.Context, _ uuid.UUID, slug string, projectID, intakeID uuid.UUID) (jsontext.Value, error) {
	return repository.intakeContainerDetail(ctx, slug, projectID, intakeID)
}

func (repository *Repository) IntakeIssueList(ctx context.Context, userID uuid.UUID, slug string, projectID uuid.UUID, options ListOptions) (jsontext.Value, error) {
	return repository.intakeIssueList(ctx, userID, slug, projectID, options)
}

func (repository *Repository) IntakeIssueDetail(ctx context.Context, userID uuid.UUID, slug string, projectID, issueID uuid.UUID) (jsontext.Value, error) {
	return repository.intakeIssueDetail(ctx, userID, slug, projectID, issueID)
}

func (repository *Repository) DescriptionVersionList(ctx context.Context, userID uuid.UUID, slug string, projectID, issueID uuid.UUID, cursor *string) (jsontext.Value, error) {
	if err := repository.authorizeIntakeIssue(ctx, userID, slug, projectID, issueID); err != nil {
		return nil, err
	}
	pageSize, page, err := parseDescriptionCursor(cursor)
	if err != nil {
		return nil, err
	}
	start := page * pageSize
	var data []byte
	err = repository.database.QueryRow(ctx, descriptionVersionListSQL, slug, projectID, issueID, pageSize, start, page).Scan(&data)
	return scannedJSON(data, err, "list intake work item description versions")
}

func (repository *Repository) DescriptionVersionDetail(ctx context.Context, userID uuid.UUID, slug string, projectID, issueID, versionID uuid.UUID) (jsontext.Value, error) {
	if err := repository.authorizeIntakeIssue(ctx, userID, slug, projectID, issueID); err != nil {
		return nil, err
	}
	var data []byte
	err := repository.database.QueryRow(ctx, descriptionVersionDetailSQL, slug, projectID, issueID, versionID).Scan(&data)
	return scannedJSON(data, err, "get intake work item description version")
}

func (repository *Repository) IntakeList(ctx context.Context, userID uuid.UUID, slug string, projectID uuid.UUID) (jsontext.Value, error) {
	return repository.intakeContainerList(ctx, userID, slug, projectID)
}

func (repository *Repository) IntakeDetail(ctx context.Context, _ uuid.UUID, slug string, projectID, intakeID uuid.UUID) (jsontext.Value, error) {
	return repository.intakeContainerDetail(ctx, slug, projectID, intakeID)
}

func (repository *Repository) draftIssueList(ctx context.Context, userID uuid.UUID, slug string, options ListOptions) (jsontext.Value, error) {
	if err := repository.authorizeWorkspace(ctx, userID, slug, []int{5, 15, 20}); err != nil {
		return nil, err
	}
	if err := validateDraftIssueFilters(options); err != nil {
		return nil, err
	}
	builder := newWhereBuilder(userID, slug)
	builder.conditions = append(builder.conditions, "draft.created_by_id = $1", "workspace.slug = $2", "draft.deleted_at IS NULL", "workspace.deleted_at IS NULL")
	if err := addIssueFilters(builder, "draft", options.Filters, options.PresentFilters, true, options.Timezone); err != nil {
		return nil, err
	}
	pagination, err := parseOffsetPagination(options.PerPage, options.PerPageSet || options.PerPage != "", options.Cursor, options.CursorSet || options.Cursor != "")
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf(draftIssueListSQL, strings.Join(builder.conditions, " AND "), pagination.limitPlaceholder(builder), pagination.offsetPlaceholder(builder))
	var data []byte
	if err := repository.database.QueryRow(ctx, query, builder.args...).Scan(&data); err != nil {
		return nil, fmt.Errorf("list draft issues: %w", err)
	}
	return jsontext.Value(data), nil
}

func (repository *Repository) intakeIssueList(ctx context.Context, userID uuid.UUID, slug string, projectID uuid.UUID, options ListOptions) (jsontext.Value, error) {
	if err := repository.authorizeProject(ctx, userID, slug, projectID, []int{5, 15, 20}); err != nil {
		return nil, err
	}
	intakeID, guestRestricted, err := repository.intakeContext(ctx, userID, slug, projectID)
	if err != nil {
		return nil, err
	}
	if validateIntakeErr := validateIntakeIssueFilters(options); validateIntakeErr != nil {
		return nil, validateIntakeErr
	}
	builder := newWhereBuilder(intakeID, projectID)
	builder.conditions = append(builder.conditions, "intake_issue.intake_id = $1", "intake_issue.project_id = $2", "intake_issue.deleted_at IS NULL")
	if guestRestricted {
		builder.addCondition("intake_issue.created_by_id = %s", userID)
	}
	if addIssueFiltersErr := addIssueFilters(builder, "issue", options.Filters, options.PresentFilters, false, options.Timezone); addIssueFiltersErr != nil {
		return nil, addIssueFiltersErr
	}
	if addIntakeStatusesErr := addIntakeStatuses(builder, options.Filters["status"], options.StatusSet || options.Filters["status"] != ""); addIntakeStatusesErr != nil {
		return nil, addIntakeStatusesErr
	}
	pagination, err := parseOffsetPagination(options.PerPage, options.PerPageSet || options.PerPage != "", options.Cursor, options.CursorSet || options.Cursor != "")
	if err != nil {
		return nil, err
	}
	order := intakeIssueOrder(options.OrderBy)
	query := fmt.Sprintf(intakeIssueListSQL, order, strings.Join(builder.conditions, " AND "), pagination.limitPlaceholder(builder), pagination.offsetPlaceholder(builder))
	var data []byte
	if err := repository.database.QueryRow(ctx, query, builder.args...).Scan(&data); err != nil {
		return nil, fmt.Errorf("list intake issues: %w", err)
	}
	return jsontext.Value(data), nil
}

func (repository *Repository) intakeIssueDetail(ctx context.Context, userID uuid.UUID, slug string, projectID, issueID uuid.UUID) (jsontext.Value, error) {
	if err := repository.authorizeProjectOrIssueCreator(ctx, userID, slug, projectID, issueID); err != nil {
		return nil, err
	}
	intakeID, guestRestricted, err := repository.intakeContext(ctx, userID, slug, projectID)
	if errors.Is(err, ErrNotFound) {
		return nil, ErrMissingIntake
	}
	if err != nil {
		return nil, err
	}
	if guestRestricted {
		var allowed bool
		err = repository.database.QueryRow(ctx, guestIntakeIssueAccessSQL, intakeID, projectID, issueID, userID).Scan(&allowed)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		if err != nil {
			return nil, fmt.Errorf("authorize guest intake issue detail: %w", err)
		}
		if !allowed {
			return nil, ErrGuestRestricted
		}
	}
	var data []byte
	err = repository.database.QueryRow(ctx, intakeIssueDetailSQL, intakeID, projectID, issueID, userID).Scan(&data)
	return scannedJSON(data, err, "get intake issue")
}

func (repository *Repository) intakeContainerList(ctx context.Context, userID uuid.UUID, slug string, projectID uuid.UUID) (jsontext.Value, error) {
	if err := repository.authorizeProject(ctx, userID, slug, projectID, []int{15, 20}); err != nil {
		return nil, err
	}
	var data []byte
	if err := repository.database.QueryRow(ctx, intakeContainerListSQL, slug, projectID).Scan(&data); err != nil {
		return nil, fmt.Errorf("list project intake: %w", err)
	}
	return jsontext.Value(data), nil
}

func (repository *Repository) intakeContainerDetail(ctx context.Context, slug string, projectID, intakeID uuid.UUID) (jsontext.Value, error) {
	var data []byte
	err := repository.database.QueryRow(ctx, intakeContainerDetailSQL, slug, projectID, intakeID).Scan(&data)
	return scannedJSON(data, err, "get project intake")
}

func (repository *Repository) authorizeWorkspace(ctx context.Context, userID uuid.UUID, slug string, roles []int) error {
	var allowed bool
	if err := repository.database.QueryRow(ctx, workspaceAuthorizationSQL, userID, slug, roles).Scan(&allowed); err != nil {
		return fmt.Errorf("authorize workspace draft read: %w", err)
	}
	if !allowed {
		return ErrForbidden
	}
	return nil
}

func (repository *Repository) authorizeDraftIssueDetail(ctx context.Context, userID uuid.UUID, slug string, draftID uuid.UUID) error {
	var allowed bool
	if err := repository.database.QueryRow(ctx, draftIssueDetailAuthorizationSQL, userID, slug, draftID).Scan(&allowed); err != nil {
		return fmt.Errorf("authorize workspace draft detail: %w", err)
	}
	if !allowed {
		return ErrForbidden
	}
	return nil
}

func (repository *Repository) authorizeProject(ctx context.Context, userID uuid.UUID, slug string, projectID uuid.UUID, roles []int) error {
	var allowed bool
	if err := repository.database.QueryRow(ctx, projectAuthorizationSQL, userID, slug, projectID, roles).Scan(&allowed); err != nil {
		return fmt.Errorf("authorize project intake read: %w", err)
	}
	if !allowed {
		return ErrForbidden
	}
	return nil
}

func (repository *Repository) authorizeProjectOrIssueCreator(ctx context.Context, userID uuid.UUID, slug string, projectID, issueID uuid.UUID) error {
	var allowed bool
	if err := repository.database.QueryRow(ctx, projectOrCreatorAuthorizationSQL, userID, slug, projectID, issueID).Scan(&allowed); err != nil {
		return fmt.Errorf("authorize intake issue detail: %w", err)
	}
	if !allowed {
		return ErrForbidden
	}
	return nil
}

func (repository *Repository) authorizeIntakeIssue(ctx context.Context, userID uuid.UUID, slug string, projectID, issueID uuid.UUID) error {
	if err := repository.authorizeProject(ctx, userID, slug, projectID, []int{5, 15, 20}); err != nil {
		return err
	}
	var guestRestricted bool
	err := repository.database.QueryRow(ctx, descriptionIssueAccessSQL, userID, slug, projectID, issueID).Scan(&guestRestricted)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("authorize intake description version read: %w", err)
	}
	if guestRestricted {
		return ErrGuestRestricted
	}
	return nil
}

func (repository *Repository) intakeContext(ctx context.Context, userID uuid.UUID, slug string, projectID uuid.UUID) (uuid.UUID, bool, error) {
	var intakeID uuid.UUID
	var guestRestricted bool
	err := repository.database.QueryRow(ctx, intakeContextSQL, userID, slug, projectID).Scan(&intakeID, &guestRestricted)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil(), false, ErrNotFound
	}
	if err != nil {
		return uuid.Nil(), false, fmt.Errorf("load project intake: %w", err)
	}
	return intakeID, guestRestricted, nil
}

func scannedJSON(data []byte, err error, operation string) (jsontext.Value, error) {
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", operation, err)
	}
	return jsontext.Value(data), nil
}

type offsetPagination struct {
	limit int
	page  int
}

func parseOffsetPagination(perPage string, perPageSet bool, cursor string, cursorSet bool) (offsetPagination, error) {
	query := apicontract.Query{
		Params:  map[string]string{"per_page": perPage, "cursor": cursor},
		Present: map[string]bool{"per_page": perPageSet, "cursor": cursorSet},
	}
	limit, err := apicontract.PageSize(query)
	if err != nil {
		return offsetPagination{}, ErrInvalidPerPage
	}
	page, err := apicontract.CursorPage(query, limit)
	if err != nil {
		return offsetPagination{}, ErrInvalidCursor
	}
	return offsetPagination{limit: limit, page: page}, nil
}

func parseDescriptionCursor(cursor *string) (int, int, error) {
	limit, page, _, err := apicontract.CursorWindow(cursor)
	if err != nil {
		return 0, 0, ErrInvalidCursor
	}
	return limit, page, nil
}

func (pagination offsetPagination) limitPlaceholder(builder *whereBuilder) string {
	return builder.placeholder(pagination.limit)
}

func (pagination offsetPagination) offsetPlaceholder(builder *whereBuilder) string {
	return builder.placeholder(pagination.page * pagination.limit)
}

type whereBuilder struct {
	args       []any
	conditions []string
}

func newWhereBuilder(args ...any) *whereBuilder {
	return &whereBuilder{args: append([]any(nil), args...)}
}

func (builder *whereBuilder) placeholder(value any) string {
	builder.args = append(builder.args, value)
	return fmt.Sprintf("$%d", len(builder.args))
}

func (builder *whereBuilder) addCondition(format string, value any) {
	builder.conditions = append(builder.conditions, fmt.Sprintf(format, builder.placeholder(value)))
}

func listFilterPresent(options ListOptions, name string) bool {
	return options.PresentFilters[name] || options.Filters[name] != ""
}

func filterHasUUIDOrNone(value string) bool {
	for _, item := range strings.Split(value, ",") {
		if item == "None" {
			return true
		}
		if _, err := uuid.Parse(item); err == nil {
			return true
		}
	}
	return false
}

func filterHasUUID(value string) bool {
	for _, item := range strings.Split(value, ",") {
		if _, err := uuid.Parse(item); err == nil {
			return true
		}
	}
	return false
}

func validateDraftIssueFilters(options ListOptions) error {
	// Drafts have native label, assignee, cycle and module relations. The
	// issue-only relations below cannot be evaluated against a draft.
	if listFilterPresent(options, "subscriber") {
		return ErrBadRequest
	}
	if filterHasUUIDOrNone(options.Filters["logged_by"]) {
		return ErrBadRequest
	}
	if filterHasUUID(options.Filters["mentions"]) {
		return ErrBadRequest
	}
	for _, name := range []string{"intake_status", "inbox_status"} {
		if _, applies := nativeIntakeStatusFilter(options.Filters[name]); applies {
			return ErrBadRequest
		}
	}
	if listFilterPresent(options, "type") && options.Filters["type"] == "" {
		options.Filters["type"] = "all"
	}
	return nil
}

func validateIntakeIssueFilters(options ListOptions) error {
	if filterHasUUIDOrNone(options.Filters["logged_by"]) {
		return ErrBadRequest
	}
	if listFilterPresent(options, "type") && options.Filters["type"] == "" {
		options.Filters["type"] = "all"
	}
	return nil
}

func addIntakeStatuses(builder *whereBuilder, value string, set bool) error {
	if !set {
		value = "-2"
	}
	statuses, err := parseIntakeStatuses(strings.Split(value, ","))
	if err != nil {
		return err
	}
	if len(statuses) > 0 {
		builder.addCondition("intake_issue.status = ANY(%s)", statuses)
	}
	return nil
}

func parseIntakeStatuses(values []string) ([]int, error) {
	statuses := make([]int, 0, len(values))
	for _, value := range values {
		if value == "null" {
			continue
		}
		status, err := strconv.ParseInt(value, 10, 32)
		if err != nil {
			return nil, ErrBadRequest
		}
		statuses = append(statuses, int(status))
	}
	return statuses, nil
}

func addIssueFilters(builder *whereBuilder, alias string, filters map[string]string, present map[string]bool, draft bool, timezone string) error {
	if filters == nil {
		return nil
	}
	addUUIDFilter(builder, alias+".state_id", filters["state"], false)
	addStringFilter(builder, alias+".priority", filters["priority"])
	if err := addStrictEstimatePointFilter(builder, alias+".estimate_point_id", filters["estimate_point"]); err != nil {
		return err
	}
	addUUIDFilter(builder, alias+".created_by_id", filters["created_by"], true)
	addUUIDFilter(builder, alias+".project_id", filters["project"], false)
	addUUIDFilter(builder, alias+".parent_id", filters["parent"], true)
	if value := filters["name"]; value != "" {
		builder.addCondition(alias+".name ILIKE '%%' || %s || '%%' ESCAPE '\\'", escapeLikePattern(value))
	}
	now := time.Now().UTC()
	for _, field := range []string{"created_at", "updated_at", "completed_at"} {
		if err := addDateFilter(builder, alias+"."+field, filters[field], timezone, true, now); err != nil {
			return err
		}
	}
	for _, field := range []string{"start_date", "target_date"} {
		if err := addDateFilter(builder, alias+"."+field, filters[field], timezone, false, now); err != nil {
			return err
		}
	}
	if value, exists := filters["sub_issue"]; exists && value == "false" {
		builder.conditions = append(builder.conditions, alias+".parent_id IS NULL")
	}
	if filters["start_target_date"] == "true" {
		builder.conditions = append(builder.conditions, alias+".start_date IS NOT NULL", alias+".target_date IS NOT NULL")
	}
	if value := filters["type"]; value != "" {
		groups := []string{"backlog", "unstarted", "started", "completed", "cancelled"}
		switch value {
		case "backlog":
			groups = []string{"backlog"}
		case "active":
			groups = []string{"unstarted", "started"}
		}
		builder.addCondition("EXISTS (SELECT 1 FROM states filtered_state WHERE filtered_state.id = "+alias+".state_id AND filtered_state.\"group\" = ANY(%s))", groups)
	} else if value := filters["state_group"]; value != "" {
		groups := csvStrings(value)
		if len(groups) > 0 {
			builder.addCondition("EXISTS (SELECT 1 FROM states filtered_state WHERE filtered_state.id = "+alias+".state_id AND filtered_state.\"group\" = ANY(%s))", groups)
		}
	}
	if draft {
		addRelationUUIDFilter(builder, alias, "draft_issue_labels", "draft_issue_id", "label_id", filters["labels"], true, present["labels"], true)
		addRelationUUIDFilter(builder, alias, "draft_issue_assignees", "draft_issue_id", "assignee_id", filters["assignees"], true, present["assignees"], true)
		addRelationUUIDFilter(builder, alias, "draft_issue_cycles", "draft_issue_id", "cycle_id", filters["cycle"], true, present["cycle"], true)
		addRelationUUIDFilter(builder, alias, "draft_issue_modules", "draft_issue_id", "module_id", filters["module"], true, present["module"], true)
		return nil
	}
	addRelationUUIDFilter(builder, alias, "issue_labels", "issue_id", "label_id", filters["labels"], true, present["labels"], true)
	addRelationUUIDFilter(builder, alias, "issue_assignees", "issue_id", "assignee_id", filters["assignees"], true, present["assignees"], true)
	addRelationUUIDFilter(builder, alias, "issue_mentions", "issue_id", "mention_id", filters["mentions"], false, present["mentions"], false)
	addRelationUUIDFilter(builder, alias, "cycle_issues", "issue_id", "cycle_id", filters["cycle"], true, present["cycle"], true)
	addRelationUUIDFilter(builder, alias, "module_issues", "issue_id", "module_id", filters["module"], true, present["module"], true)
	addRelationUUIDFilter(builder, alias, "issue_subscribers", "issue_id", "subscriber_id", filters["subscriber"], false, present["subscriber"], true)
	statuses, hasStatusFilter := nativeIntakeStatusFilter(filters["intake_status"])
	if inboxStatuses, hasInboxStatusFilter := nativeIntakeStatusFilter(filters["inbox_status"]); hasInboxStatusFilter {
		statuses, hasStatusFilter = inboxStatuses, true
	}
	if hasStatusFilter {
		parsed, err := parseIntakeStatuses(statuses)
		if err != nil {
			return err
		}
		builder.addCondition("EXISTS (SELECT 1 FROM intake_issues filtered_intake WHERE filtered_intake.issue_id = "+alias+".id AND filtered_intake.status = ANY(%s))", parsed)
	}
	return nil
}

func nativeIntakeStatusFilter(value string) ([]string, bool) {
	statuses := make([]string, 0)
	for _, item := range strings.Split(value, ",") {
		if item == "null" {
			continue
		}
		if item == "" {
			return nil, false
		}
		statuses = append(statuses, item)
	}
	return statuses, len(statuses) > 0
}

func addUUIDFilter(builder *whereBuilder, column, value string, allowNone bool) {
	if value == "" {
		return
	}
	ids, hasNone := csvUUIDs(value)
	if len(ids) > 0 {
		condition := column + " = ANY(%s)"
		if allowNone && hasNone {
			condition = "(" + column + " IS NULL OR " + condition + ")"
		}
		builder.addCondition(condition, ids)
	} else if allowNone && hasNone {
		builder.conditions = append(builder.conditions, column+" IS NULL")
	}
}

func addStringFilter(builder *whereBuilder, column, value string) {
	values := csvStrings(value)
	if len(values) > 0 {
		builder.addCondition(column+" = ANY(%s)", values)
	}
}

func addStrictEstimatePointFilter(builder *whereBuilder, column, value string) error {
	if value == "" {
		return nil
	}
	ids := make([]uuid.UUID, 0)
	for _, item := range strings.Split(value, ",") {
		if item == "null" {
			continue
		}
		if item == "" {
			return nil
		}
		id, err := uuid.Parse(item)
		if err != nil {
			return ErrInvalidDetail
		}
		ids = append(ids, id)
	}
	if len(ids) > 0 {
		builder.addCondition(column+" = ANY(%s)", ids)
	}
	return nil
}

func addRelationUUIDFilter(builder *whereBuilder, alias, table, owner, target, value string, allowNone, present, liveOnly bool) {
	if value == "" && (!present || allowNone) {
		return
	}
	relationScope := "filtered_relation." + owner + " = " + alias + ".id"
	liveScope := relationScope
	if liveOnly {
		liveScope += " AND filtered_relation.deleted_at IS NULL"
	}
	labelRelation := true
	switch table {
	case "issue_labels":
		relationScope = issuequery.LiveIssueLabelPredicate(alias, "filtered_relation")
		liveScope = relationScope
	case "draft_issue_labels":
		relationScope = issuequery.LiveDraftIssueLabelPredicate(alias, "filtered_relation")
		liveScope = relationScope
	default:
		labelRelation = false
	}
	ids, hasNone := csvUUIDs(value)
	if len(ids) > 0 {
		condition := "EXISTS (SELECT 1 FROM " + table + " filtered_relation WHERE " + liveScope + " AND filtered_relation." + target + " = ANY(%s))"
		if allowNone && hasNone {
			condition = "(NOT EXISTS (SELECT 1 FROM " + table + " filtered_relation WHERE " + liveScope + ") OR " + condition + ")"
		}
		builder.addCondition(condition, ids)
		return
	}
	if allowNone && hasNone {
		builder.conditions = append(builder.conditions, "NOT EXISTS (SELECT 1 FROM "+table+" filtered_relation WHERE "+liveScope+")")
		return
	}
	if present && (!hasNone || !allowNone) && liveOnly && !labelRelation {
		builder.conditions = append(builder.conditions, "(NOT EXISTS (SELECT 1 FROM "+table+" filtered_relation WHERE "+relationScope+") OR EXISTS (SELECT 1 FROM "+table+" filtered_relation WHERE "+liveScope+"))")
	}
}

func addDateFilter(builder *whereBuilder, column, value, timezone string, timestamp bool, now time.Time) error {
	if value == "" {
		return nil
	}
	items := strings.Split(value, ",")
	for _, item := range items {
		if item == "" {
			return nil
		}
	}
	var before, after, contains string
	var hasBefore, hasAfter, hasContains bool
	for _, item := range items {
		parts := strings.Split(item, ";")
		if len(parts) == 1 {
			contains, hasContains = parts[0], true
			continue
		}
		boundary := parts[0]
		if days, relative := relativeDateDays(boundary); relative {
			if len(parts) != 3 || (parts[1] != "after" && parts[1] != "before") || parts[2] != "fromnow" {
				return ErrInvalidDetail
			}
			if parts[1] == "before" {
				days = -days
			}
			location := time.UTC
			if timezone != "" {
				var err error
				location, err = time.LoadLocation(timezone)
				if err != nil {
					return ErrInvalidDetail
				}
			}
			date := now.In(location).AddDate(0, 0, days)
			if date.Year() < 1 || date.Year() > 9999 {
				return ErrInvalidDetail
			}
			boundary = date.Format(time.DateOnly)
			if parts[1] == "after" {
				after, hasAfter = boundary, true
			} else {
				before, hasBefore = boundary, true
			}
			continue
		}
		parsedBoundary, err := datetime.Date(boundary)
		if err != nil {
			return ErrInvalidDetail
		}
		boundary = parsedBoundary
		if dateFilterContains(parts, "after") {
			after, hasAfter = boundary, true
		} else {
			before, hasBefore = boundary, true
		}
	}
	if hasContains {
		expression := dateFilterExpression(builder, column, timezone, timestamp)
		builder.addCondition("("+expression+")::text LIKE '%%' || %s || '%%' ESCAPE '\\'", escapeLikePattern(contains))
	}
	if hasAfter {
		expression := dateFilterExpression(builder, column, timezone, timestamp)
		builder.addCondition(expression+" >= %s::date", after)
	}
	if hasBefore {
		expression := dateFilterExpression(builder, column, timezone, timestamp)
		builder.addCondition(expression+" <= %s::date", before)
	}
	return nil
}

func asciiDecimalDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}

func dateFilterContains(parts []string, value string) bool {
	for _, part := range parts {
		if part == value {
			return true
		}
	}
	return false
}

func dateFilterExpression(builder *whereBuilder, column, timezone string, timestamp bool) string {
	if !timestamp {
		return column
	}
	if timezone == "" {
		timezone = "UTC"
	}
	return "timezone(" + builder.placeholder(timezone) + ", " + column + ")::date"
}

func relativeDateDays(value string) (int, bool) {
	multiplier := 0
	number := ""
	if prefix, ok := strings.CutSuffix(value, "_weeks"); ok {
		number, multiplier = prefix, 7
	} else if prefix, ok := strings.CutSuffix(value, "_months"); ok {
		number, multiplier = prefix, 30
	} else {
		return 0, false
	}
	if !asciiDecimalDigits(number) {
		return 0, false
	}
	duration, err := strconv.Atoi(number)
	if err != nil || duration > 9999*366/multiplier {
		return 0, false
	}
	return duration * multiplier, true
}

func csvStrings(value string) []string {
	values := make([]string, 0)
	for _, item := range strings.Split(value, ",") {
		if item == "" {
			return nil
		}
		if item != "null" {
			values = append(values, item)
		}
	}
	return values
}

func escapeLikePattern(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `%`, `\%`)
	return strings.ReplaceAll(value, `_`, `\_`)
}

func csvUUIDs(value string) ([]uuid.UUID, bool) {
	ids := make([]uuid.UUID, 0)
	hasNone := false
	for _, item := range strings.Split(value, ",") {
		if item == "None" {
			hasNone = true
			continue
		}
		if parsed, err := uuid.Parse(item); err == nil {
			ids = append(ids, parsed)
		}
	}
	return ids, hasNone
}

const draftIssueDetailAuthorizationSQL = `
SELECT EXISTS (
  SELECT 1 FROM workspace_members member
  JOIN workspaces workspace ON workspace.id = member.workspace_id
  WHERE member.member_id = $1 AND workspace.slug = $2
    AND member.is_active = true AND member.deleted_at IS NULL AND workspace.deleted_at IS NULL
    AND member.role IN (5,15,20)
    AND EXISTS (
      SELECT 1 FROM draft_issues draft
      WHERE draft.id = $3 AND draft.workspace_id = workspace.id
        AND draft.created_by_id = $1 AND draft.deleted_at IS NULL
    )
)`

func intakeIssueOrder(value string) string {
	allowed := map[string]string{
		"issue__created_at": "issue.created_at", "issue__updated_at": "issue.updated_at", "issue__sequence_id": "issue.sequence_id",
		"issue__sort_order": "issue.sort_order", "issue__target_date": "issue.target_date", "issue__start_date": "issue.start_date",
		"issue__priority": "issue.priority", "issue__state__name": "(SELECT name FROM states WHERE id = issue.state_id)",
		"created_at": "intake_issue.created_at", "updated_at": "intake_issue.updated_at", "status": "intake_issue.status",
	}
	if value == "" {
		value = "-issue__created_at"
	}
	descending := strings.HasPrefix(value, "-")
	bare := strings.TrimPrefix(value, "-")
	column, ok := allowed[bare]
	if !ok || strings.HasPrefix(bare, "-") {
		return "issue.created_at DESC"
	}
	if descending {
		return column + " DESC"
	}
	return column + " ASC"
}

const workspaceAuthorizationSQL = `
SELECT EXISTS (
  SELECT 1 FROM workspace_members member
  JOIN workspaces workspace ON workspace.id = member.workspace_id
  WHERE member.member_id = $1 AND workspace.slug = $2
    AND member.role = ANY($3) AND member.is_active = true
    AND member.deleted_at IS NULL AND workspace.deleted_at IS NULL
)`

const projectAuthorizationSQL = `
SELECT EXISTS (
  SELECT 1 FROM project_members member
  JOIN projects project ON project.id = member.project_id
  JOIN workspaces workspace ON workspace.id = member.workspace_id
  WHERE member.member_id = $1 AND workspace.slug = $2 AND project.id = $3
    AND member.is_active = true AND member.deleted_at IS NULL
    AND project.deleted_at IS NULL AND workspace.deleted_at IS NULL
    AND (member.role = ANY($4) OR (EXISTS (
      SELECT 1 FROM workspace_members workspace_member
      WHERE workspace_member.workspace_id = workspace.id AND workspace_member.member_id = $1
        AND workspace_member.role = 20 AND workspace_member.is_active = true AND workspace_member.deleted_at IS NULL
    ) AND 15 = ANY($4)))
)`

const projectOrCreatorAuthorizationSQL = `
SELECT EXISTS (
  SELECT 1 FROM workspaces workspace
  JOIN workspace_members workspace_member ON workspace_member.workspace_id = workspace.id
    AND workspace_member.member_id = $1 AND workspace_member.is_active = true AND workspace_member.deleted_at IS NULL
  WHERE workspace.slug = $2 AND workspace.deleted_at IS NULL AND (
    EXISTS (SELECT 1 FROM issues creator_issue WHERE creator_issue.id = $4 AND creator_issue.created_by_id = $1 AND creator_issue.deleted_at IS NULL)
    OR EXISTS (
      SELECT 1 FROM project_members project_member
      JOIN projects project ON project.id = project_member.project_id
      WHERE project_member.member_id = $1 AND project_member.project_id = $3
        AND project_member.workspace_id = workspace.id AND project_member.role IN (5, 15, 20)
        AND project_member.is_active = true AND project_member.deleted_at IS NULL
        AND project.deleted_at IS NULL
    )
    OR (EXISTS (
      SELECT 1 FROM project_members project_member
      WHERE project_member.member_id = $1 AND project_member.project_id = $3
        AND project_member.workspace_id = workspace.id AND project_member.is_active = true AND project_member.deleted_at IS NULL
    ) AND workspace_member.role = 20)
  )
)`

const descriptionIssueAccessSQL = `
SELECT (
  EXISTS (
    SELECT 1 FROM project_members member
    WHERE member.project_id = project.id AND member.workspace_id = workspace.id
      AND member.member_id = $1 AND member.role = 5
      AND member.is_active = true AND member.deleted_at IS NULL
  )
  AND project.guest_view_all_features = false
  AND issue.created_by_id IS DISTINCT FROM $1
)
FROM issues issue
JOIN projects project ON project.id = issue.project_id
JOIN workspaces workspace ON workspace.id = issue.workspace_id
WHERE workspace.slug = $2 AND project.id = $3 AND issue.id = $4
  AND workspace.deleted_at IS NULL AND project.deleted_at IS NULL
  AND issue.deleted_at IS NULL
LIMIT 1`

const intakeContextSQL = `
SELECT intake.id,
  EXISTS (
    SELECT 1 FROM project_members member
    WHERE member.project_id = project.id AND member.workspace_id = workspace.id AND member.member_id = $1
      AND member.role = 5 AND member.is_active = true AND member.deleted_at IS NULL
  ) AND project.guest_view_all_features = false
FROM intakes intake
JOIN projects project ON project.id = intake.project_id
JOIN workspaces workspace ON workspace.id = intake.workspace_id
WHERE workspace.slug = $2 AND project.id = $3
  AND intake.deleted_at IS NULL AND project.deleted_at IS NULL AND workspace.deleted_at IS NULL
ORDER BY intake.name
LIMIT 1`

var draftIssueProjection = `json_build_object(
  'id', draft.id, 'name', draft.name, 'state_id', draft.state_id, 'sort_order', draft.sort_order,
  'completed_at', draft.completed_at, 'estimate_point', draft.estimate_point_id, 'priority', draft.priority,
  'start_date', draft.start_date, 'target_date', draft.target_date, 'project_id', draft.project_id,
  'parent_id', ` + apicontract.VisibleParentSQL("draft", "$1::uuid") + `,
  'cycle_id', (SELECT relation.cycle_id FROM draft_issue_cycles relation WHERE relation.draft_issue_id = draft.id AND relation.deleted_at IS NULL ORDER BY relation.created_at DESC LIMIT 1),
  'module_ids', COALESCE((SELECT json_agg(DISTINCT relation.module_id) FROM draft_issue_modules relation JOIN modules module ON module.id = relation.module_id WHERE relation.draft_issue_id = draft.id AND relation.deleted_at IS NULL AND module.archived_at IS NULL), '[]'::json),
  'label_ids', COALESCE((SELECT json_agg(DISTINCT relation.label_id) FROM draft_issue_labels relation JOIN labels label ON label.id = relation.label_id WHERE ` + issuequery.LiveDraftIssueLabelPredicate("draft", "relation") + `), '[]'::json),
  'assignee_ids', COALESCE((SELECT json_agg(DISTINCT relation.assignee_id) FROM draft_issue_assignees relation WHERE relation.draft_issue_id = draft.id AND relation.deleted_at IS NULL AND EXISTS (SELECT 1 FROM project_members member WHERE member.member_id = relation.assignee_id AND member.is_active = true)), '[]'::json),
  'created_at', draft.created_at, 'updated_at', draft.updated_at, 'created_by', draft.created_by_id,
  'updated_by', draft.updated_by_id, 'type_id', draft.type_id, 'description_html', draft.description_html,
  'custom_properties', draft.custom_properties
)`

var draftIssueListSQL = `
WITH filtered AS (
  SELECT draft.*, row_number() OVER (ORDER BY draft.created_at DESC) AS ordinal
  FROM draft_issues draft
  JOIN workspaces workspace ON workspace.id = draft.workspace_id
  WHERE %s
), counted AS (SELECT count(*)::int AS total FROM filtered), page AS (
  SELECT * FROM filtered ORDER BY created_at DESC LIMIT %s OFFSET %s
)
SELECT json_build_object(
  'grouped_by', NULL, 'sub_grouped_by', NULL, 'total_count', counted.total,
  'next_cursor', concat(%[2]s::bigint, ':', (%[3]s::bigint / %[2]s::bigint) + 1, ':0'),
  'prev_cursor', concat(%[2]s::bigint, ':', (%[3]s::bigint / %[2]s::bigint) - 1, ':1'),
  'next_page_results', counted.total > %[3]s::bigint + %[2]s::bigint,
  'prev_page_results', %[3]s::bigint > 0, 'count', count(draft.id)::int,
  'total_pages', ceil(counted.total::numeric / %[2]s::bigint)::int,
  'total_results', counted.total, 'extra_stats', NULL,
  'results', COALESCE(json_agg(` + draftIssueProjection + ` ORDER BY draft.ordinal) FILTER (WHERE draft.id IS NOT NULL), '[]'::json)
)
FROM counted LEFT JOIN page draft ON true GROUP BY counted.total`

var draftIssueDetailSQL = `
SELECT ` + draftIssueProjection + `
FROM draft_issues draft
JOIN workspaces workspace ON workspace.id = draft.workspace_id
WHERE draft.id = $3 AND draft.created_by_id = $1 AND workspace.slug = $2
  AND draft.deleted_at IS NULL AND workspace.deleted_at IS NULL
LIMIT 1`

var issueLiteProjection = `json_build_object(
  'id', issue.id, 'name', issue.name, 'priority', issue.priority, 'sequence_id', issue.sequence_id,
  'project_id', issue.project_id, 'created_at', issue.created_at,
  'label_ids', COALESCE((SELECT json_agg(DISTINCT relation.label_id) FROM issue_labels relation JOIN labels label ON label.id = relation.label_id WHERE ` + issuequery.LiveIssueLabelPredicate("issue", "relation") + `), '[]'::json),
  'created_by', issue.created_by_id
)`

var intakeIssueListSQL = `
WITH filtered AS (
  SELECT intake_issue.id, row_number() OVER (ORDER BY %s) AS ordinal
  FROM intake_issues intake_issue
  JOIN issues issue ON issue.id = intake_issue.issue_id
  WHERE %s
), counted AS (SELECT count(*)::int AS total FROM filtered), page_ids AS (
  SELECT id, ordinal FROM filtered ORDER BY ordinal LIMIT %s OFFSET %s
)
SELECT json_build_object(
  'grouped_by', NULL, 'sub_grouped_by', NULL, 'total_count', counted.total,
  'next_cursor', concat(%[3]s::bigint, ':', (%[4]s::bigint / %[3]s::bigint) + 1, ':0'),
  'prev_cursor', concat(%[3]s::bigint, ':', (%[4]s::bigint / %[3]s::bigint) - 1, ':1'),
  'next_page_results', counted.total > %[4]s::bigint + %[3]s::bigint,
  'prev_page_results', %[4]s::bigint > 0, 'count', count(intake_issue.id)::int,
  'total_pages', ceil(counted.total::numeric / %[3]s::bigint)::int, 'total_results', counted.total,
  'extra_stats', NULL,
  'results', COALESCE(json_agg(json_build_object(
    'id', intake_issue.id, 'status', intake_issue.status, 'duplicate_to', intake_issue.duplicate_to_id,
    'snoozed_till', intake_issue.snoozed_till, 'source', intake_issue.source,
    'issue', ` + issueLiteProjection + `, 'created_by', intake_issue.created_by_id
  ) ORDER BY page_ids.ordinal) FILTER (WHERE intake_issue.id IS NOT NULL), '[]'::json)
)
FROM counted
LEFT JOIN page_ids ON true
LEFT JOIN intake_issues intake_issue ON intake_issue.id = page_ids.id
LEFT JOIN issues issue ON issue.id = intake_issue.issue_id
GROUP BY counted.total`

var intakeIssueDetailSQL = `
SELECT json_build_object(
  'id', intake_issue.id, 'status', intake_issue.status, 'duplicate_to', intake_issue.duplicate_to_id,
  'snoozed_till', intake_issue.snoozed_till,
  'duplicate_issue_detail', CASE WHEN duplicate_issue.id IS NULL THEN NULL ELSE json_build_object(
    'id', duplicate_issue.id, 'name', duplicate_issue.name, 'priority', duplicate_issue.priority,
    'sequence_id', duplicate_issue.sequence_id, 'project_id', duplicate_issue.project_id,
    'created_at', duplicate_issue.created_at, 'created_by', duplicate_issue.created_by_id
  ) END,
  'source', intake_issue.source,
  'issue', json_build_object(
    'id', issue.id, 'name', issue.name, 'state_id', issue.state_id, 'sort_order', issue.sort_order,
    'completed_at', issue.completed_at, 'estimate_point', issue.estimate_point_id, 'priority', issue.priority,
    'start_date', issue.start_date, 'target_date', issue.target_date, 'sequence_id', issue.sequence_id,
    'project_id', issue.project_id, 'parent_id', ` + apicontract.VisibleParentSQL("issue", "$4::uuid") + `,
    'label_ids', COALESCE((SELECT json_agg(DISTINCT relation.label_id) FROM issue_labels relation JOIN labels label ON label.id = relation.label_id WHERE ` + issuequery.LiveIssueLabelPredicate("issue", "relation") + `), '[]'::json),
    'assignee_ids', COALESCE((SELECT json_agg(DISTINCT relation.assignee_id) FROM issue_assignees relation WHERE relation.issue_id = issue.id AND relation.deleted_at IS NULL), '[]'::json),
    'created_at', issue.created_at, 'updated_at', issue.updated_at, 'created_by', issue.created_by_id,
    'updated_by', issue.updated_by_id, 'is_draft', issue.is_draft, 'archived_at', issue.archived_at,
    'description_html', issue.description_html
  )
)
FROM intake_issues intake_issue
JOIN issues issue ON issue.id = intake_issue.issue_id
LEFT JOIN issues duplicate_issue ON duplicate_issue.id = intake_issue.duplicate_to_id
WHERE intake_issue.intake_id = $1 AND intake_issue.project_id = $2 AND intake_issue.issue_id = $3
  AND intake_issue.deleted_at IS NULL
LIMIT 1`

const guestIntakeIssueAccessSQL = `
SELECT intake_issue.created_by_id IS NOT DISTINCT FROM $4
FROM intake_issues intake_issue
JOIN issues issue ON issue.id = intake_issue.issue_id
WHERE intake_issue.intake_id = $1 AND intake_issue.project_id = $2 AND intake_issue.issue_id = $3
  AND intake_issue.deleted_at IS NULL
LIMIT 1`

const intakeProjection = `json_build_object(
  'id', intake.id, 'created_at', intake.created_at, 'updated_at', intake.updated_at,
  'deleted_at', intake.deleted_at, 'created_by', intake.created_by_id, 'updated_by', intake.updated_by_id,
  'workspace', intake.workspace_id, 'project', intake.project_id, 'name', intake.name,
  'description', intake.description, 'is_default', intake.is_default, 'view_props', intake.view_props,
  'logo_props', intake.logo_props,
  'project_detail', json_build_object(
    'id', project.id, 'identifier', project.identifier, 'name', project.name,
    'cover_image', project.cover_image,
    'cover_image_url', CASE
      WHEN asset.id IS NULL THEN NULLIF(project.cover_image, '')
      WHEN asset.entity_type IN ('WORKSPACE_LOGO', 'USER_AVATAR', 'USER_COVER', 'PROJECT_COVER') THEN '/api/assets/v2/static/' || asset.id::text || '/'
      WHEN asset.entity_type = 'ISSUE_ATTACHMENT' THEN '/api/assets/v2/workspaces/' || (SELECT slug FROM workspaces asset_workspace WHERE asset_workspace.id = asset.workspace_id) || '/projects/' || COALESCE(asset.project_id::text, 'None') || '/issues/' || COALESCE(asset.issue_id::text, 'None') || '/attachments/' || asset.id::text || '/'
      WHEN asset.entity_type IN ('ISSUE_DESCRIPTION', 'COMMENT_DESCRIPTION', 'PAGE_DESCRIPTION', 'DRAFT_ISSUE_DESCRIPTION') THEN '/api/assets/v2/workspaces/' || (SELECT slug FROM workspaces asset_workspace WHERE asset_workspace.id = asset.workspace_id) || '/projects/' || COALESCE(asset.project_id::text, 'None') || '/' || asset.id::text || '/'
      ELSE NULL
    END,
    'logo_props', project.logo_props, 'description', project.description
  ),
  'pending_issue_count', (SELECT count(*)::int FROM intake_issues pending WHERE pending.intake_id = intake.id AND pending.status = -2)
)`

const intakeContainerListSQL = `
SELECT COALESCE((
  SELECT ` + intakeProjection + `
  FROM intakes intake
  JOIN projects project ON project.id = intake.project_id
  JOIN workspaces workspace ON workspace.id = intake.workspace_id
  LEFT JOIN file_assets asset ON asset.id = project.cover_image_asset_id
  WHERE workspace.slug = $1 AND project.id = $2
    AND intake.deleted_at IS NULL AND project.deleted_at IS NULL AND workspace.deleted_at IS NULL
  ORDER BY intake.name LIMIT 1
), '{}'::json)`

const intakeContainerDetailSQL = `
SELECT ` + intakeProjection + `
FROM intakes intake
JOIN projects project ON project.id = intake.project_id
JOIN workspaces workspace ON workspace.id = intake.workspace_id
LEFT JOIN file_assets asset ON asset.id = project.cover_image_asset_id
WHERE workspace.slug = $1 AND project.id = $2 AND intake.id = $3
  AND intake.deleted_at IS NULL AND project.deleted_at IS NULL AND workspace.deleted_at IS NULL
LIMIT 1`

const descriptionVersionListSQL = `
WITH filtered AS (
  SELECT version.*, row_number() OVER () AS ordinal, count(*) OVER ()::int AS total
  FROM issue_description_versions version
  JOIN issues issue ON issue.id = version.issue_id
  JOIN projects project ON project.id = version.project_id
  JOIN workspaces workspace ON workspace.id = version.workspace_id
  WHERE workspace.slug = $1 AND project.id = $2 AND issue.id = $3
    AND version.deleted_at IS NULL AND issue.deleted_at IS NULL
), page AS (SELECT * FROM filtered LIMIT $4 OFFSET $5), totals AS (
  SELECT count(*)::int AS total FROM filtered
)
SELECT json_build_object(
  'prev_cursor', concat($4::int, ':', $6::numeric - 1, ':0'),
  'cursor', concat($4::int, ':', $6::numeric, ':0'),
  'next_cursor', CASE WHEN totals.total > $5::numeric + $4::int THEN concat($4::int, ':', $6::numeric + 1, ':0') ELSE NULL END,
  'prev_page_results', $6::numeric > 0, 'next_page_results', totals.total > $5::numeric + $4::int,
  'page_count', count(page.id)::int, 'total_results', totals.total,
  'total_pages', ceil(totals.total::numeric / $4::int)::int,
  'results', COALESCE(json_agg(json_build_object(
    'id', page.id, 'workspace', page.workspace_id, 'project', page.project_id, 'issue', page.issue_id,
    'last_saved_at', page.last_saved_at, 'owned_by', page.owned_by_id, 'created_at', page.created_at,
    'updated_at', page.updated_at, 'created_by', page.created_by_id, 'updated_by', page.updated_by_id
  ) ORDER BY page.ordinal) FILTER (WHERE page.id IS NOT NULL), '[]'::json)
)
FROM totals LEFT JOIN page ON true GROUP BY totals.total`

const descriptionVersionDetailSQL = `
SELECT json_build_object(
  'id', version.id, 'workspace', version.workspace_id, 'project', version.project_id,
  'issue', version.issue_id, 'description_binary', replace(encode(version.description_binary, 'base64'), E'\n', ''),
  'description_html', version.description_html, 'description_stripped', version.description_stripped,
  'description_json', version.description_json, 'last_saved_at', version.last_saved_at,
  'owned_by', version.owned_by_id, 'created_at', version.created_at, 'updated_at', version.updated_at,
  'created_by', version.created_by_id, 'updated_by', version.updated_by_id
)
FROM issue_description_versions version
JOIN issues issue ON issue.id = version.issue_id
JOIN projects project ON project.id = version.project_id
JOIN workspaces workspace ON workspace.id = version.workspace_id
WHERE workspace.slug = $1 AND project.id = $2 AND issue.id = $3 AND version.id = $4
  AND version.deleted_at IS NULL AND issue.deleted_at IS NULL
LIMIT 1`
