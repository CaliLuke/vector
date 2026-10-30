// Package cyclereads contains the PostgreSQL boundary for cycle read operations.
package cyclereads

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/issuequery"
	"github.com/CaliLuke/go-plane/internal/workitemtypes"

	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrForbidden          = errors.New("active membership with an allowed role required")
	ErrNotFound           = errors.New("cycle resource not found")
	ErrCycleDatesMissing  = errors.New("cycle has no start or end date")
	ErrInvalidPagination  = errors.New("invalid pagination")
	ErrZeroPageSize       = errors.New("cycle issue page size is zero")
	ErrNegativePageSize   = errors.New("cycle issue page size is negative")
	ErrPreviousPageSlice  = errors.New("cycle issue previous page slice is invalid")
	ErrPaginationOverflow = errors.New("cycle issue pagination offset overflow")
	ErrPaginationParsing  = errors.New("error in parsing cycle issue pagination")
	ErrInvalidGrouping    = errors.New("invalid cycle issue grouping")
)

type InvalidGroupingFieldError struct {
	Field string
	Value string
}

type FilterError struct {
	Message string
	Code    string
	Errors  any
}

func (err *FilterError) Error() string {
	return err.Message
}

func (err *InvalidGroupingFieldError) Error() string {
	return fmt.Sprintf("invalid %s field: %s", err.Field, err.Value)
}

type database interface {
	Begin(context.Context) (pgx.Tx, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type taskPublisher interface {
	Task(context.Context, string, []any, map[string]any) error
}

type RecentVisitAdmitter interface {
	AdmitRecentVisit(context.Context, string, string, string, string, string, string) error
}

type Repository struct {
	database     database
	publisher    taskPublisher
	recentVisits RecentVisitAdmitter
}

type IssueListOptions struct {
	IncludeProperties bool
	snapshot          *workitemtypes.PropertyDiscoverySnapshot
	typeIDs           map[string]bool
	properties        *cyclePropertyBindings
	OrderBy           string
	GroupBy           string
	SubGroupBy        string
	Cursor            string
	CursorSet         bool
	PerPage           int
	PerPageSet        bool
	Filters           map[string]string
	ComplexFilter     string
}

func NewRepository(database database) (*Repository, error) {
	if database == nil {
		return nil, fmt.Errorf("cycle reads database is required")
	}
	return &Repository{database: database}, nil
}

func NewRepositoryWithRecentVisits(database database, publisher taskPublisher, recentVisits RecentVisitAdmitter) (*Repository, error) {
	if database == nil || publisher == nil {
		return nil, fmt.Errorf("cycle read database and task publisher are required")
	}
	return &Repository{database: database, publisher: publisher, recentVisits: recentVisits}, nil
}

func (repository *Repository) admitRecentVisit(ctx context.Context, userID uuid.UUID, slug string, projectID, cycleID uuid.UUID) error {
	if repository.recentVisits != nil {
		return repository.recentVisits.AdmitRecentVisit(ctx, uuid.New().String(), slug, "cycle", cycleID.String(), userID.String(), projectID.String())
	}
	if repository.publisher == nil {
		return nil
	}
	return repository.publisher.Task(ctx, "plane.bgtasks.recent_visited_task.recent_visited_task", []any{}, map[string]any{
		"slug": slug, "entity_name": "cycle", "entity_identifier": cycleID.String(), "user_id": userID.String(), "project_id": projectID.String(),
	})
}

func (repository *Repository) AuthorizeWorkspace(ctx context.Context, userID uuid.UUID, slug string) error {
	var allowed bool
	if err := repository.database.QueryRow(ctx, workspaceAuthorizationSQL, userID, slug).Scan(&allowed); err != nil {
		return fmt.Errorf("authorize workspace cycle read: %w", err)
	}
	if !allowed {
		return ErrForbidden
	}
	return nil
}

func (repository *Repository) AuthorizeProject(
	ctx context.Context,
	userID uuid.UUID,
	slug string,
	projectID uuid.UUID,
	allowGuest bool,
) error {
	var allowed bool
	if err := repository.database.QueryRow(ctx, projectAuthorizationSQL, userID, slug, projectID, allowGuest).Scan(&allowed); err != nil {
		return fmt.Errorf("authorize project cycle read: %w", err)
	}
	if !allowed {
		return ErrForbidden
	}
	return nil
}

func (repository *Repository) WorkspaceList(
	ctx context.Context,
	userID uuid.UUID,
	slug string,
) (json.RawMessage, error) {
	if err := repository.AuthorizeWorkspace(ctx, userID, slug); err != nil {
		return nil, err
	}
	return repository.json(ctx, workspaceCycleListSQL, "list workspace cycles", userID, slug)
}

func (repository *Repository) ArchivedList(
	ctx context.Context,
	userID uuid.UUID,
	slug string,
	projectID uuid.UUID,
) (json.RawMessage, error) {
	if err := repository.AuthorizeProject(ctx, userID, slug, projectID, false); err != nil {
		return nil, err
	}
	return repository.json(ctx, archivedCycleListSQL, "list archived cycles", userID, slug, projectID)
}

func (repository *Repository) ArchivedDetail(
	ctx context.Context,
	userID uuid.UUID,
	slug string,
	projectID uuid.UUID,
	cycleID uuid.UUID,
) (json.RawMessage, error) {
	if err := repository.AuthorizeProject(ctx, userID, slug, projectID, false); err != nil {
		return nil, err
	}
	return repository.detail(ctx, archivedCycleDetailSQL, "get archived cycle", userID, slug, projectID, cycleID)
}

func (repository *Repository) List(
	ctx context.Context,
	userID uuid.UUID,
	slug string,
	projectID uuid.UUID,
	cycleView string,
) (json.RawMessage, error) {
	if err := repository.AuthorizeProject(ctx, userID, slug, projectID, true); err != nil {
		return nil, err
	}
	current := cycleView == "current"
	return repository.json(ctx, projectCycleListSQL, "list project cycles", userID, slug, projectID, current)
}

func (repository *Repository) Analytics(
	ctx context.Context,
	userID uuid.UUID,
	slug string,
	projectID uuid.UUID,
	cycleID uuid.UUID,
	analyticsType string,
) (json.RawMessage, error) {
	if err := repository.AuthorizeProject(ctx, userID, slug, projectID, true); err != nil {
		return nil, err
	}
	var datesPresent bool
	var data []byte
	err := repository.database.QueryRow(
		ctx, cycleAnalyticsSQL, userID, slug, projectID, cycleID, analyticsType,
	).Scan(&datesPresent, &data)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get cycle analytics: %w", err)
	}
	if !datesPresent {
		return nil, ErrCycleDatesMissing
	}
	return json.RawMessage(data), nil
}

// ArchiveDetail implements the repaired /cycles/{cycle_id}/archive/ GET. The
// Django route passes cycle_id, so this method deliberately looks up that ID
// rather than an absent pk parameter.
func (repository *Repository) ArchiveDetail(
	ctx context.Context,
	userID uuid.UUID,
	slug string,
	projectID uuid.UUID,
	cycleID uuid.UUID,
) (json.RawMessage, error) {
	if err := repository.AuthorizeProject(ctx, userID, slug, projectID, false); err != nil {
		return nil, err
	}
	return repository.detail(ctx, activeCycleArchiveDetailSQL, "get cycle archive", userID, slug, projectID, cycleID)
}

func (repository *Repository) issueResults(
	ctx context.Context,
	userID uuid.UUID,
	slug string,
	projectID uuid.UUID,
	cycleID uuid.UUID,
	options IssueListOptions,
) (json.RawMessage, error) {
	if err := repository.AuthorizeProject(ctx, userID, slug, projectID, false); err != nil {
		return nil, err
	}
	if !validGroup(options.GroupBy) {
		return nil, &InvalidGroupingFieldError{Field: "group_by", Value: options.GroupBy}
	}
	if !validGroup(options.SubGroupBy) {
		return nil, &InvalidGroupingFieldError{Field: "sub_group_by", Value: options.SubGroupBy}
	}
	if options.GroupBy != "" && options.GroupBy == options.SubGroupBy {
		return nil, ErrInvalidGrouping
	}
	limit, cursorLimit, windowLimit, page, previous, err := pagination(
		options.PerPage,
		options.PerPageSet || options.PerPage != 0,
		options.Cursor,
		options.CursorSet || options.Cursor != "",
	)
	if err != nil {
		return nil, err
	}
	if options.GroupBy != "" {
		if float64(page)*cursorLimit < 0 {
			return nil, ErrPaginationParsing
		}
		return repository.groupedIssues(ctx, userID, slug, projectID, cycleID, options, limit, cursorLimit, windowLimit, page)
	}
	offset, ok := checkedIntProduct(page, limit)
	if !ok {
		return nil, ErrPaginationOverflow
	}
	if offset < 0 {
		return nil, ErrPaginationParsing
	}
	if limit == 0 {
		return nil, ErrZeroPageSize
	}
	if limit < 0 {
		return nil, ErrNegativePageSize
	}
	if previous && cursorLimit != float64(limit) {
		return nil, ErrPreviousPageSlice
	}
	result, err := repository.queryIssues(ctx, userID, slug, projectID, cycleID, options, limit, page)
	if err != nil {
		return nil, err
	}
	totalPages := 0
	if result.total > 0 {
		totalPages = int(math.Ceil(float64(result.total) / float64(limit)))
	}
	nextPage := (page+1)*limit < result.expandedTotal
	if previous && (options.GroupBy != "" || !issueMultiGroup(options.SubGroupBy)) {
		nextPage = page+1 < totalPages
	}
	responseOptions := options
	responseOptions.SubGroupBy = ""
	return encodeIssueEnvelope(responseOptions, limit, page, page > 0, nextPage, len(result.rows), totalPages, result.total, result.rows)
}

type issueQueryResult struct {
	rows          []map[string]any
	total         int
	expandedTotal int
	groups        []any
	subgroups     []any
}

func (repository *Repository) queryIssues(
	ctx context.Context,
	userID uuid.UUID,
	slug string,
	projectID uuid.UUID,
	cycleID uuid.UUID,
	options IssueListOptions,
	limit int,
	page int,
) (issueQueryResult, error) {
	query, args, err := issueListQuery(userID, slug, projectID, cycleID, options, limit, page)
	if err != nil {
		return issueQueryResult{}, err
	}
	var rowsData, groupsData, subgroupsData []byte
	var result issueQueryResult
	if err := repository.database.QueryRow(ctx, query, args...).Scan(
		&rowsData, &result.total, &groupsData, &subgroupsData, &result.expandedTotal,
	); err != nil {
		return result, fmt.Errorf("list cycle issues: %w", err)
	}
	if err := json.Unmarshal(rowsData, &result.rows); err != nil {
		return result, fmt.Errorf("decode cycle issue rows: %w", err)
	}
	if options.IncludeProperties && options.snapshot != nil {
		if projectionErr := options.snapshot.ProjectStoredRows(ctx, result.rows, workitemtypes.DiscoveryElevatedProjectVisibility); projectionErr != nil {
			return issueQueryResult{}, cycleDiscoveryError(projectionErr)
		}
	}
	if err := json.Unmarshal(groupsData, &result.groups); err != nil {
		return result, fmt.Errorf("decode cycle issue groups: %w", err)
	}
	if err := json.Unmarshal(subgroupsData, &result.subgroups); err != nil {
		return result, fmt.Errorf("decode cycle issue subgroups: %w", err)
	}
	return result, nil
}

func (repository *Repository) groupedIssues(
	ctx context.Context,
	userID uuid.UUID,
	slug string,
	projectID uuid.UUID,
	cycleID uuid.UUID,
	options IssueListOptions,
	limit int,
	cursorLimit float64,
	windowLimit float64,
	page int,
) (json.RawMessage, error) {
	query, args, err := groupedIssueListQuery(userID, slug, projectID, cycleID, options, cursorLimit, windowLimit, page)
	if err != nil {
		return nil, err
	}
	var resultsData []byte
	var total, count, maximum int
	var nextPage bool
	if err := repository.database.QueryRow(ctx, query, args...).Scan(&resultsData, &total, &count, &maximum, &nextPage); err != nil {
		return nil, fmt.Errorf("list grouped cycle issues: %w", err)
	}
	if count > 0 && limit == 0 {
		return nil, ErrZeroPageSize
	}
	totalPages := 0
	if maximum > 0 {
		totalPages = int(math.Ceil(float64(maximum) / float64(limit)))
	}
	if options.IncludeProperties && options.snapshot != nil {
		projected, projectionErr := projectGroupedRows(ctx, options, resultsData)
		if projectionErr != nil {
			return nil, cycleDiscoveryError(projectionErr)
		}
		resultsData = projected
	}
	return encodeIssueEnvelope(
		options, limit, page, page > 0, nextPage, count, totalPages, total, json.RawMessage(resultsData),
	)
}

func encodeIssueEnvelope(
	options IssueListOptions,
	limit int,
	page int,
	previousPage bool,
	nextPage bool,
	count int,
	totalPages int,
	total int,
	results any,
) (json.RawMessage, error) {
	envelope := map[string]any{
		"grouped_by": nilIfEmpty(options.GroupBy), "sub_grouped_by": nilIfEmpty(options.SubGroupBy),
		"total_count": total, "next_cursor": fmt.Sprintf("%d:%d:0", limit, page+1),
		"prev_cursor": fmt.Sprintf("%d:%d:1", limit, page-1), "next_page_results": nextPage,
		"prev_page_results": previousPage, "count": count, "total_pages": totalPages,
		"total_results": total, "extra_stats": nil, "results": results,
	}
	encoded, err := json.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("encode cycle issue page: %w", err)
	}
	return encoded, nil
}

// IssueDetail repairs the Django retrieve mismatch by selecting the active
// CycleIssue relationship with issue_id, not an unavailable DRF pk argument.
func (repository *Repository) IssueDetail(
	ctx context.Context,
	userID uuid.UUID,
	slug string,
	projectID uuid.UUID,
	cycleID uuid.UUID,
	issueID uuid.UUID,
) (json.RawMessage, error) {
	return repository.detail(
		ctx, cycleIssueDetailSQL, "get cycle issue", userID, slug, projectID, cycleID, issueID,
	)
}

func (repository *Repository) Progress(
	ctx context.Context,
	userID uuid.UUID,
	slug string,
	projectID uuid.UUID,
	cycleID uuid.UUID,
) (json.RawMessage, error) {
	if err := repository.AuthorizeProject(ctx, userID, slug, projectID, true); err != nil {
		return nil, err
	}
	return repository.detail(ctx, cycleProgressSQL, "get cycle progress", slug, projectID, cycleID)
}

func (repository *Repository) Detail(
	ctx context.Context,
	userID uuid.UUID,
	slug string,
	projectID uuid.UUID,
	cycleID uuid.UUID,
) (json.RawMessage, error) {
	if err := repository.AuthorizeProject(ctx, userID, slug, projectID, false); err != nil {
		return nil, err
	}
	data, err := repository.detail(ctx, projectCycleDetailSQL, "get cycle", userID, slug, projectID, cycleID)
	if err != nil {
		return nil, err
	}
	if err := repository.admitRecentVisit(ctx, userID, slug, projectID, cycleID); err != nil {
		return nil, fmt.Errorf("record recent cycle visit: %w", err)
	}
	return data, nil
}

func (repository *Repository) json(ctx context.Context, query, operation string, args ...any) (json.RawMessage, error) {
	var data []byte
	if err := repository.database.QueryRow(ctx, query, args...).Scan(&data); err != nil {
		return nil, fmt.Errorf("%s: %w", operation, err)
	}
	return json.RawMessage(data), nil
}

func (repository *Repository) detail(ctx context.Context, query, operation string, args ...any) (json.RawMessage, error) {
	var data []byte
	err := repository.database.QueryRow(ctx, query, args...).Scan(&data)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", operation, err)
	}
	return json.RawMessage(data), nil
}

func pagination(
	perPage int,
	perPageSet bool,
	cursor string,
	cursorSet bool,
) (limit int, cursorLimit float64, windowLimit float64, page int, previous bool, err error) {
	if !perPageSet {
		perPage = 1000
	}
	if perPage > 1000 {
		return 0, 0, 0, 0, false, ErrInvalidPagination
	}
	if !cursorSet {
		return perPage, float64(perPage), float64(perPage), 0, false, nil
	}
	if cursor == "" {
		return 0, 0, 0, 0, false, ErrInvalidPagination
	}
	parts := strings.Split(cursor, ":")
	if len(parts) != 3 {
		return 0, 0, 0, 0, false, ErrInvalidPagination
	}
	cursorLimit, limitError := parseCursorValue(parts[0])
	cursorPageNumber, pageError := parsePythonInteger(parts[1])
	previousNumber, previousError := parsePythonInteger(parts[2])
	if limitError != nil || pageError != nil || previousError != nil {
		return 0, 0, 0, 0, false, ErrInvalidPagination
	}
	cursorPage, pageFits := pythonIntegerAsInt(cursorPageNumber)
	if !pageFits {
		return 0, 0, 0, 0, false, ErrPaginationOverflow
	}
	windowLimit = cursorLimit
	if windowLimit == 0 {
		windowLimit = float64(perPage)
	}
	return perPage, cursorLimit, windowLimit, cursorPage, previousNumber.Sign() != 0, nil
}

func ParsePageSize(value string) (int, bool, error) {
	integer, err := parsePythonInteger(value)
	if err != nil {
		return 0, false, err
	}
	if integer.Cmp(big.NewInt(1000)) > 0 {
		return 0, true, nil
	}
	if result, ok := pythonIntegerAsInt(integer); ok {
		return result, false, nil
	}
	return -1, false, nil
}

func parseCursorValue(value string) (float64, error) {
	value = strings.TrimFunc(value, pythonIntegerWhitespace)
	if strings.Contains(value, ".") {
		result, err := strconv.ParseFloat(value, 64)
		if err != nil && !math.IsInf(result, 0) {
			return 0, err
		}
		return result, nil
	}
	integer, err := parsePythonInteger(value)
	if err != nil {
		return 0, err
	}
	result, _ := new(big.Float).SetInt(integer).Float64()
	return result, nil
}

func parsePythonInteger(value string) (*big.Int, error) {
	runes := []rune(strings.TrimFunc(value, pythonIntegerWhitespace))
	if len(runes) == 0 {
		return nil, ErrInvalidPagination
	}
	start := 0
	var normalized strings.Builder
	if runes[0] == '+' || runes[0] == '-' {
		normalized.WriteRune(runes[0])
		start = 1
	}
	if start == len(runes) {
		return nil, ErrInvalidPagination
	}
	digits := 0
	previousDigit := false
	for index := start; index < len(runes); index++ {
		if digit, ok := pythonDecimalDigit(runes[index]); ok {
			normalized.WriteByte('0' + digit)
			digits++
			previousDigit = true
			continue
		}
		if runes[index] != '_' || !previousDigit || index+1 == len(runes) {
			return nil, ErrInvalidPagination
		}
		if _, ok := pythonDecimalDigit(runes[index+1]); !ok {
			return nil, ErrInvalidPagination
		}
		previousDigit = false
	}
	if digits > 4300 {
		return nil, ErrInvalidPagination
	}
	integer, ok := new(big.Int).SetString(normalized.String(), 10)
	if !ok {
		return nil, ErrInvalidPagination
	}
	return integer, nil
}

func pythonIntegerWhitespace(value rune) bool {
	return unicode.IsSpace(value) || value >= '\u001c' && value <= '\u001f'
}

func pythonDecimalDigit(value rune) (byte, bool) {
	if value >= '0' && value <= '9' {
		return byte(value - '0'), true
	}
	for _, digitRange := range unicode.Nd.R16 {
		unsigned := uint32(value)
		if unsigned >= uint32(digitRange.Lo) && unsigned <= uint32(digitRange.Hi) &&
			(unsigned-uint32(digitRange.Lo))%uint32(digitRange.Stride) == 0 {
			return byte(((unsigned - uint32(digitRange.Lo)) / uint32(digitRange.Stride)) % 10), true
		}
	}
	for _, digitRange := range unicode.Nd.R32 {
		unsigned := uint32(value)
		if unsigned >= digitRange.Lo && unsigned <= digitRange.Hi &&
			(unsigned-digitRange.Lo)%digitRange.Stride == 0 {
			return byte(((unsigned - digitRange.Lo) / digitRange.Stride) % 10), true
		}
	}
	return 0, false
}

func pythonIntegerAsInt(value *big.Int) (int, bool) {
	if !value.IsInt64() {
		return 0, false
	}
	result := int(value.Int64())
	return result, big.NewInt(int64(result)).Cmp(value) == 0
}

func checkedIntProduct(left, right int) (int, bool) {
	if left == 0 || right == 0 {
		return 0, true
	}
	minInt := -int(^uint(0)>>1) - 1
	if left == -1 && right == minInt || right == -1 && left == minInt {
		return 0, false
	}
	product := left * right
	return product, product/right == left
}

func nilIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}

var groupFields = map[string]string{
	"state_id":                "state_id",
	"state__group":            "state__group",
	"priority":                "priority",
	"labels__id":              "label_ids",
	"assignees__id":           "assignee_ids",
	"issue_module__module_id": "module_ids",
	"cycle_id":                "cycle_id",
	"project_id":              "project_id",
	"created_by":              "created_by",
	"target_date":             "target_date",
	"start_date":              "start_date",
}

func validGroup(group string) bool {
	if group == "" {
		return true
	}
	_, ok := groupFields[group]
	return ok
}

func issueListQuery(
	userID uuid.UUID,
	slug string,
	projectID uuid.UUID,
	cycleID uuid.UUID,
	options IssueListOptions,
	limit int,
	page int,
) (string, []any, error) {
	args := []any{userID, slug, projectID, cycleID}
	legacyJoins, predicates, err := issueFilterSource(options, &args, true)
	if err != nil {
		return "", nil, err
	}
	standaloneJoin := ""
	standalonePredicate := "true"
	if options.GroupBy == "" {
		standaloneJoin = standaloneSubgroupJoin(options.SubGroupBy)
		if standaloneJoin != "" {
			standalonePredicate = "(standalone_subgroup.issue_id IS NULL OR standalone_subgroup.deleted_at IS NULL)"
		}
	}
	order := issueOrder(options.OrderBy)
	query := fmt.Sprintf(
		issueListSQL,
		strings.Join(legacyJoins, "\n  "),
		strings.Join(predicates, "\n    AND "),
		standaloneSubgroupSelect(options),
		standaloneJoin,
		standalonePredicate,
		order,
		issueListRelationProjection(options),
		order,
		len(args)+1,
		len(args)+2,
		issueGroupUniverseSQL(options.GroupBy),
		issueGroupUniverseSQL(options.SubGroupBy),
	)
	args = append(args, limit, page*limit)
	return query, args, nil
}

func issueFilterSource(options IssueListOptions, args *[]any, includeComplex bool) ([]string, []string, error) {
	predicates := []string{
		"workspace.slug = request_scope.slug", "project.id = request_scope.project_id", "cycle_issue.cycle_id = request_scope.cycle_id",
		"cycle_issue.deleted_at IS NULL", "issue.deleted_at IS NULL", "issue.archived_at IS NULL",
		"issue.is_draft = false", "state.\"group\" IS DISTINCT FROM 'triage'", "project.deleted_at IS NULL", "project.archived_at IS NULL",
	}
	structured := options.ComplexFilter
	if !includeComplex {
		structured = ""
	}
	complexFilter, err := (issueFilterCompiler{typeIDs: options.typeIDs, properties: options.properties}).build(structured, args)
	if err != nil {
		return nil, nil, err
	}
	if complexFilter != "" {
		predicates = append(predicates, complexFilter)
	}
	complexJoins := issueComplexRelationJoins(structured)
	legacyJoins := append(complexJoins, issueLegacyRelationJoins(options.Filters, args)...)
	predicates = append(predicates, issueFilterPredicates(options.Filters, args)...)
	return legacyJoins, predicates, nil
}

func standaloneSubgroupJoin(group string) string {
	switch group {
	case "labels__id":
		return "LEFT JOIN issue_labels standalone_subgroup ON " + issueRelationPredicate("issue_labels", "standalone_subgroup", true)
	case "assignees__id":
		return "LEFT JOIN issue_assignees standalone_subgroup ON standalone_subgroup.issue_id=issue.id"
	case "issue_module__module_id":
		return "LEFT JOIN module_issues standalone_subgroup ON standalone_subgroup.issue_id=issue.id"
	default:
		return ""
	}
}

func standaloneSubgroupSelect(options IssueListOptions) string {
	if options.GroupBy != "" {
		return ""
	}
	switch options.SubGroupBy {
	case "labels__id":
		return ",standalone_subgroup.label_id standalone_subgroup_id"
	case "assignees__id":
		return ",standalone_subgroup.assignee_id standalone_subgroup_id"
	case "issue_module__module_id":
		return ",standalone_subgroup.module_id standalone_subgroup_id"
	default:
		return ""
	}
}

func issueListRelationProjection(options IssueListOptions) string {
	assignees := `'assignee_ids',COALESCE((SELECT json_agg(DISTINCT assignee.assignee_id) FROM issue_assignees assignee WHERE assignee.issue_id=issue.id AND assignee.deleted_at IS NULL),'[]'::json)`
	labels := `'label_ids',COALESCE((SELECT json_agg(DISTINCT issue_label.label_id) FROM issue_labels issue_label WHERE ` + issueRelationPredicate("issue_labels", "issue_label", true) + `),'[]'::json)`
	modules := `'module_ids',COALESCE((SELECT json_agg(DISTINCT module_issue.module_id) FROM module_issues module_issue JOIN modules projected_module ON projected_module.id=module_issue.module_id WHERE module_issue.issue_id=issue.id AND module_issue.deleted_at IS NULL AND projected_module.deleted_at IS NULL AND projected_module.archived_at IS NULL),'[]'::json)`
	if options.GroupBy == "" {
		switch options.SubGroupBy {
		case "labels__id":
			labels = `'labels__id',issue.standalone_subgroup_id`
		case "assignees__id":
			assignees = `'assignees__id',issue.standalone_subgroup_id`
		case "issue_module__module_id":
			modules = `'issue_module__module_id',issue.standalone_subgroup_id`
		}
	}
	return strings.Join([]string{assignees, labels, modules}, ",")
}

func groupedIssueListQuery(
	userID uuid.UUID,
	slug string,
	projectID uuid.UUID,
	cycleID uuid.UUID,
	options IssueListOptions,
	cursorLimit float64,
	windowLimit float64,
	page int,
) (string, []any, error) {
	args := []any{userID, slug, projectID, cycleID}
	legacyJoins, predicates, err := issueFilterSource(options, &args, true)
	if err != nil {
		return "", nil, err
	}
	for _, group := range []string{options.GroupBy, options.SubGroupBy} {
		if predicate := issueMultiGroupPredicate(group); predicate != "" {
			predicates = append(predicates, predicate)
		}
	}
	lowerBound := math.Floor(float64(page) * cursorLimit)
	upperBound := math.Ceil(float64(page)*cursorLimit + windowLimit + 1)
	args = append(args, lowerBound, upperBound)
	lowerPosition, upperPosition := len(args)-1, len(args)
	subgroupKeys := `jsonb_build_array('__none__')`
	maximumBucketCount := `(SELECT max(group_total.countable_results)::int FROM group_totals group_total)`
	payload := groupedIssuePayloadSQL
	bucketKeys := `
SELECT group_value.value group_key,'__none__'::text subgroup_key
FROM universes
CROSS JOIN LATERAL jsonb_array_elements_text(universes.group_values) group_value(value)`
	if issueMultiGroup(options.GroupBy) {
		bucketKeys = `SELECT DISTINCT group_key,'__none__'::text subgroup_key FROM paged`
		payload = multiGroupedIssuePayloadSQL
	}
	if options.SubGroupBy != "" {
		subgroupKeys = issueGroupKeysSQL(options.SubGroupBy)
		maximumBucketCount = `(SELECT max(bucket.total_results)::int FROM buckets bucket)`
		payload = subgroupedIssuePayloadSQL
		bucketKeys = `SELECT DISTINCT group_key,subgroup_key FROM expanded`
	}
	query := fmt.Sprintf(
		groupedIssueListSQL,
		strings.Join(legacyJoins, "\n  "),
		strings.Join(predicates, "\n    AND "),
		lowerPosition,
		upperPosition,
		issueOrder(options.OrderBy),
		issueGroupKeysSQL(options.GroupBy),
		subgroupKeys,
		issuePagedItemSQL(options),
		issueGroupUniverseSQL(options.GroupBy),
		bucketKeys,
		issueBucketTotalSQL(options),
		payload,
		maximumBucketCount,
	)
	return query, args, nil
}

func issueGroupKeysSQL(group string) string {
	field := groupFields[group]
	switch group {
	case "labels__id", "assignees__id":
		return `CASE WHEN jsonb_array_length(item->'` + field + `')=0 THEN '["None"]'::jsonb ELSE item->'` + field + `' END`
	case "issue_module__module_id":
		return `COALESCE((SELECT jsonb_agg(DISTINCT grouped_module.module_id) FROM module_issues grouped_module WHERE grouped_module.issue_id=matching.issue_id AND grouped_module.deleted_at IS NULL),'["None"]'::jsonb)`
	default:
		return `jsonb_build_array(COALESCE(item->>'` + field + `','None'))`
	}
}

func issueMultiGroup(group string) bool {
	switch group {
	case "labels__id", "assignees__id", "issue_module__module_id":
		return true
	default:
		return false
	}
}

func issueMultiGroupPredicate(group string) string {
	var table string
	switch group {
	case "labels__id":
		// The canonical label array supplies either live label IDs or None.
		return ""
	case "assignees__id":
		table = "issue_assignees"
	case "issue_module__module_id":
		table = "module_issues"
	default:
		return ""
	}
	return `(NOT EXISTS(SELECT 1 FROM ` + table + ` grouped_link WHERE grouped_link.issue_id=issue.id)
      OR EXISTS(SELECT 1 FROM ` + table + ` grouped_link WHERE grouped_link.issue_id=issue.id AND grouped_link.deleted_at IS NULL))`
}

func issuePagedItemSQL(options IssueListOptions) string {
	expression := "paged.item"
	if issueMultiGroup(options.GroupBy) {
		field := groupFields[options.GroupBy]
		expression += ` || jsonb_build_object('` + field + `',CASE
  WHEN EXISTS(SELECT 1 FROM paged selected WHERE selected.issue_id=paged.issue_id AND selected.group_key='None') THEN '[]'::jsonb
  ELSE (SELECT jsonb_agg(DISTINCT selected.group_key) FROM paged selected WHERE selected.issue_id=paged.issue_id)
END)`
	}
	if issueMultiGroup(options.SubGroupBy) {
		field := groupFields[options.SubGroupBy]
		expression += ` || jsonb_build_object('` + field + `',CASE
  WHEN EXISTS(SELECT 1 FROM paged selected WHERE selected.issue_id=paged.issue_id AND selected.subgroup_key='None') THEN '[]'::jsonb
  ELSE (SELECT jsonb_agg(DISTINCT selected.subgroup_key) FROM paged selected WHERE selected.issue_id=paged.issue_id)
END)`
	}
	return expression
}

func issueBucketTotalSQL(options IssueListOptions) string {
	count := `count(DISTINCT expanded.issue_id) FILTER (WHERE expanded.countable)`
	if options.SubGroupBy != "" {
		return `(
  SELECT ` + count + `
  FROM expanded
  WHERE expanded.group_key=bucket_key.group_key AND expanded.subgroup_key=bucket_key.subgroup_key
)::int`
	}
	return `CASE WHEN EXISTS(
  SELECT 1 FROM expanded
  WHERE expanded.group_key=bucket_key.group_key AND expanded.subgroup_key=bucket_key.subgroup_key
) THEN GREATEST((
  SELECT ` + count + `
  FROM expanded
  WHERE expanded.group_key=bucket_key.group_key AND expanded.subgroup_key=bucket_key.subgroup_key
)::int,1) ELSE 0 END`
}

func issueGroupUniverseSQL(group string) string {
	switch group {
	case "":
		return "'[]'::json"
	case "state_id":
		return `(SELECT COALESCE(json_agg(state.id ORDER BY state.sequence),'[]'::json) FROM states state WHERE state.project_id=$3 AND state.is_triage=false AND state."group"<>'triage' AND state.deleted_at IS NULL)`
	case "state__group":
		return `'["backlog","unstarted","started","completed","cancelled"]'::json`
	case "priority":
		return `'["low","medium","high","urgent","none"]'::json`
	case "labels__id":
		return `(SELECT COALESCE(json_agg(value),'[]'::json) FROM (SELECT label.id::text value FROM labels label WHERE label.project_id=$3 AND label.workspace_id=(SELECT label_project.workspace_id FROM projects label_project WHERE label_project.id=$3) AND label.deleted_at IS NULL UNION ALL SELECT 'None') values)`
	case "assignees__id":
		return `(SELECT COALESCE(json_agg(project_member.member_id ORDER BY project_member.member_id),'[]'::json) FROM project_members project_member JOIN workspaces workspace ON workspace.id=project_member.workspace_id WHERE workspace.slug=$2 AND project_member.project_id=$3 AND project_member.is_active=true AND project_member.deleted_at IS NULL)`
	case "issue_module__module_id":
		return `(SELECT COALESCE(json_agg(value),'[]'::json) FROM (SELECT module.id::text value FROM modules module WHERE module.project_id=$3 AND module.deleted_at IS NULL UNION ALL SELECT 'None') values)`
	case "cycle_id":
		return `(SELECT COALESCE(json_agg(value),'[]'::json) FROM (SELECT cycle.id::text value FROM cycles cycle WHERE cycle.project_id=$3 AND cycle.deleted_at IS NULL UNION ALL SELECT 'None') values)`
	case "project_id":
		return `(SELECT COALESCE(json_agg(project.id ORDER BY project.id),'[]'::json) FROM projects project JOIN workspaces workspace ON workspace.id=project.workspace_id WHERE workspace.slug=$2 AND project.deleted_at IS NULL)`
	case "created_by":
		return `(SELECT COALESCE(json_agg(DISTINCT COALESCE(created_by_id::text,'None')),'[]'::json) FROM filtered)`
	case "target_date":
		return `(SELECT COALESCE(json_agg(DISTINCT COALESCE(target_date::text,'None')),'[]'::json) FROM filtered)`
	case "start_date":
		return `(SELECT COALESCE(json_agg(DISTINCT COALESCE(start_date::text,'None')),'[]'::json) FROM filtered)`
	default:
		return "'[]'::json"
	}
}

func issueOrder(orderBy string) string {
	descending := strings.HasPrefix(orderBy, "-")
	field := strings.TrimPrefix(orderBy, "-")
	fields := map[string]string{
		"created_at": "issue.created_at", "updated_at": "issue.updated_at", "sequence_id": "issue.sequence_id",
		"sort_order": "issue.sort_order", "target_date": "issue.target_date", "start_date": "issue.start_date",
		"completed_at": "issue.completed_at", "archived_at": "issue.archived_at",
		"priority":                   `CASE issue.priority WHEN 'urgent' THEN 0 WHEN 'high' THEN 1 WHEN 'medium' THEN 2 WHEN 'low' THEN 3 WHEN 'none' THEN 4 ELSE 5 END`,
		"state__name":                "state.name",
		"state__group":               `CASE state."group" WHEN 'backlog' THEN 0 WHEN 'unstarted' THEN 1 WHEN 'started' THEN 2 WHEN 'completed' THEN 3 WHEN 'cancelled' THEN 4 ELSE 5 END`,
		"assignees__first_name":      "(SELECT min(u.first_name) FROM issue_assignees ia JOIN users u ON u.id=ia.assignee_id WHERE ia.issue_id=issue.id)",
		"labels__name":               "(SELECT min(label.name) FROM issue_labels il JOIN labels label ON label.id=il.label_id WHERE " + issueRelationPredicate("issue_labels", "il", true) + ")",
		"issue_module__module__name": "(SELECT min(module.name) FROM module_issues mi JOIN modules module ON module.id=mi.module_id WHERE mi.issue_id=issue.id)",
	}
	expression, ok := fields[field]
	if !ok {
		expression = "issue.created_at"
		descending = true
	}
	direction := "ASC"
	switch field {
	case "priority":
		if !descending {
			direction = "DESC"
		}
	case "state__group":
		// Django reverses both the CASE values and direction for the descending
		// spelling, so both spellings produce backlog through cancelled.
		direction = "ASC"
	default:
		if descending {
			direction = "DESC"
		}
	}
	return expression + " " + direction + " NULLS LAST, issue.created_at DESC"
}

func issueFilterPredicates(filters map[string]string, args *[]any) []string {
	if len(filters) == 0 {
		return nil
	}
	predicates := make([]string, 0, len(filters))
	add := func(value any, template string) {
		*args = append(*args, value)
		predicates = append(predicates, fmt.Sprintf(template, len(*args)))
	}
	for name, value := range filters {
		if value == "" {
			continue
		}
		switch name {
		case "name":
			add(value, "issue.name ILIKE '%%' || $%d || '%%'")
		case "state", "state_id", "state_id__in":
			if values := UUIDs(value); len(values) > 0 {
				add(values, "issue.state_id = ANY($%d::uuid[])")
			}
		case "state_group", "state_group__in":
			if values := CSV(value); len(values) > 0 {
				add(values, "state.\"group\" = ANY($%d::text[])")
			}
		case "priority", "priority__in":
			if values := CSV(value); len(values) > 0 {
				add(values, "issue.priority = ANY($%d::text[])")
			}
		case "estimate_point":
			if values := UUIDs(value); len(values) > 0 {
				add(values, "issue.estimate_point_id = ANY($%d::uuid[])")
			}
		case "parent":
			if hasNoneToken(value) {
				predicates = append(predicates, "issue.parent_id IS NULL")
			}
			if values := UUIDs(value); len(values) > 0 {
				add(values, "issue.parent_id = ANY($%d::uuid[])")
			}
		case "labels", "label_id", "label_id__in":
			if hasNoneToken(value) {
				predicates = append(predicates, "NOT EXISTS (SELECT 1 FROM issue_labels no_label WHERE "+issueRelationPredicate("issue_labels", "no_label", true)+")")
			}
		case "assignees", "assignee_id", "assignee_id__in":
			if hasNoneToken(value) {
				predicates = append(predicates, "NOT EXISTS (SELECT 1 FROM issue_assignees no_assignee WHERE no_assignee.issue_id=issue.id)")
			}
		case "module", "module_id", "module_id__in":
			if hasNoneToken(value) {
				predicates = append(predicates, "NOT EXISTS (SELECT 1 FROM module_issues no_module WHERE no_module.issue_id=issue.id)")
			}
		case "cycle", "cycle_id__in":
			if hasNoneToken(value) {
				predicates = append(predicates, "NOT EXISTS (SELECT 1 FROM cycle_issues no_cycle WHERE no_cycle.issue_id=issue.id AND no_cycle.deleted_at IS NULL)")
			}
		case "created_by", "created_by_id", "created_by_id__in":
			if hasNoneToken(value) {
				predicates = append(predicates, "issue.created_by_id IS NULL")
			}
			if values := UUIDs(value); len(values) > 0 {
				add(values, "issue.created_by_id = ANY($%d::uuid[])")
			}
		case "logged_by":
			if hasNoneToken(value) {
				predicates = append(predicates, "issue.logged_by_id IS NULL")
			}
			if values := UUIDs(value); len(values) > 0 {
				add(values, "issue.logged_by_id = ANY($%d::uuid[])")
			}
		case "type":
			groups := []string{"backlog", "unstarted", "started", "completed", "cancelled"}
			switch value {
			case "backlog":
				groups = []string{"backlog"}
			case "active":
				groups = []string{"unstarted", "started"}
			}
			add(groups, "state.\"group\" = ANY($%d::text[])")
		case "sub_issue":
			if value == "false" {
				predicates = append(predicates, "issue.parent_id IS NULL")
			}
		case "is_archived":
			switch value {
			case "true", "True", "1":
				predicates = append(predicates, "issue.archived_at IS NOT NULL")
			case "false", "False", "0":
				predicates = append(predicates, "issue.archived_at IS NULL")
			}
		case "is_draft":
			switch value {
			case "true", "True", "1":
				predicates = append(predicates, "issue.is_draft = true")
			case "false", "False", "0":
				predicates = append(predicates, "issue.is_draft = false")
			}
		case "start_target_date":
			if value == "true" {
				predicates = append(predicates, "issue.start_date IS NOT NULL", "issue.target_date IS NOT NULL")
			}
		case "project", "project_id__in":
			if values := UUIDs(value); len(values) > 0 {
				add(values, "issue.project_id = ANY($%d::uuid[])")
			}
		case "subscriber", "subscriber_id", "subscriber_id__in", "mentions", "mention_id", "mention_id__in":
			// Positive relation filters are rendered as joins so Django's row
			// multiplicity is preserved.
		case "intake_status", "inbox_status":
			// Rendered as one join by issueLegacyRelationJoins. The later
			// inbox_status key replaces intake_status in the Django filter map.
		default:
			predicates = append(predicates, issueDateFilterPredicates(name, value, args)...)
		}
	}
	return predicates
}

// issueDateFilterPredicates preserves the legacy date lookup grammar. Plain
// timestamp filters use the requester's calendar; exact and range lookups retain
// their database date casts.
func issueDateFilterPredicates(name, value string, args *[]any) []string {
	switch name {
	case "created_at", "updated_at", "start_date", "target_date", "completed_at":
		column := map[string]string{
			"created_at": "issue.created_at", "updated_at": "issue.updated_at", "start_date": "issue.start_date",
			"target_date": "issue.target_date", "completed_at": "issue.completed_at",
		}[name]
		if name == "created_at" || name == "updated_at" || name == "completed_at" {
			column = cycleRequesterDateSQL(column)
		}
		return legacyDatePredicates(value, column, args)
	case "start_date__exact", "target_date__exact", "created_at__exact", "updated_at__exact":
		column := map[string]string{"start_date__exact": "issue.start_date", "target_date__exact": "issue.target_date", "created_at__exact": "issue.created_at", "updated_at__exact": "issue.updated_at"}[name]
		*args = append(*args, value)
		return []string{fmt.Sprintf(column+"::date = $%d::date", len(*args))}
	case "start_date__range", "target_date__range", "created_at__range", "updated_at__range":
		bounds := CSV(value)
		if len(bounds) == 2 {
			column := map[string]string{"start_date__range": "issue.start_date", "target_date__range": "issue.target_date", "created_at__range": "issue.created_at", "updated_at__range": "issue.updated_at"}[name]
			*args = append(*args, bounds)
			position := len(*args)
			return []string{fmt.Sprintf(column+"::date BETWEEN ($%d::text[])[1]::date AND ($%d::text[])[2]::date", position, position)}
		}
	}
	return nil
}

func issueLegacyRelationJoins(filters map[string]string, args *[]any) []string {
	type relationFilter struct {
		name       string
		table      string
		alias      string
		column     string
		softDelete bool
	}
	relations := []relationFilter{
		{"labels", "issue_labels", "legacy_label", "label_id", true},
		{"assignees", "issue_assignees", "legacy_assignee", "assignee_id", true},
		{"mentions", "issue_mentions", "legacy_mention", "mention_id", false},
		{"cycle", "cycle_issues", "legacy_cycle", "cycle_id", true},
		{"module", "module_issues", "legacy_module", "module_id", true},
		{"subscriber", "issue_subscribers", "legacy_subscriber", "subscriber_id", true},
	}
	joins := make([]string, 0, len(relations))
	for _, relation := range relations {
		values := UUIDs(filters[relation.name])
		if len(values) == 0 {
			continue
		}
		*args = append(*args, values)
		joins = append(joins, fmt.Sprintf("JOIN %s %s ON %s AND %s.%s=ANY($%d::uuid[])",
			relation.table, relation.alias, issueRelationPredicate(relation.table, relation.alias, relation.softDelete), relation.alias, relation.column, len(*args)))
	}
	statusValues := CSV(filters["intake_status"])
	if inboxValues := CSV(filters["inbox_status"]); len(inboxValues) > 0 {
		statusValues = inboxValues
	}
	if values := statusValues; len(values) > 0 {
		*args = append(*args, values)
		joins = append(joins, fmt.Sprintf(
			"JOIN intake_issues legacy_intake ON legacy_intake.issue_id=issue.id AND legacy_intake.status::text=ANY($%d::text[])",
			len(*args),
		))
	}
	return joins
}

var issueComplexFilterFields = map[string]struct{}{
	"assignee_id": {}, "assignee_id__exact": {}, "assignee_id__in": {},
	"cycle_id": {}, "cycle_id__exact": {}, "cycle_id__in": {},
	"module_id": {}, "module_id__exact": {}, "module_id__in": {},
	"mention_id": {}, "mention_id__exact": {}, "mention_id__in": {},
	"label_id": {}, "label_id__exact": {}, "label_id__in": {},
	"created_by_id": {}, "created_by_id__exact": {}, "created_by_id__in": {},
	"is_archived": {}, "is_archived__exact": {},
	"state_group": {}, "state_group__exact": {}, "state_group__in": {},
	"state_id": {}, "state_id__exact": {}, "state_id__in": {},
	"project_id": {}, "project_id__exact": {}, "project_id__in": {},
	"subscriber_id": {}, "subscriber_id__exact": {}, "subscriber_id__in": {},
	"created_at": {}, "created_at__exact": {}, "created_at__range": {},
	"updated_at": {}, "updated_at__exact": {}, "updated_at__range": {},
	"start_date": {}, "start_date__exact": {}, "start_date__range": {},
	"target_date": {}, "target_date__exact": {}, "target_date__range": {},
	"is_draft": {}, "is_draft__exact": {},
	"priority": {}, "priority__exact": {}, "priority__in": {},
}

func (compiler issueFilterCompiler) build(raw string, args *[]any) (string, error) {
	if raw == "" {
		return "", nil
	}
	var decoded any
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return "", newFilterError(
			"Invalid JSON for 'filter'. Expected a valid JSON object.",
			"invalid_json",
		)
	}
	switch value := decoded.(type) {
	case nil:
		return "", nil
	case bool:
		if !value {
			return "", nil
		}
	case string:
		if value == "" {
			return "", nil
		}
	case float64:
		if value == 0 {
			return "", nil
		}
	case []any:
		if len(value) == 0 {
			return "", nil
		}
	case map[string]any:
		if len(value) == 0 {
			return "", nil
		}
	}
	if err := compiler.validate(decoded, 1); err != nil {
		return "", err
	}
	clause, err := compiler.render(decoded, args, true)
	if object, ok := decoded.(map[string]any); ok {
		if _, disjunction := issueLogicalValue(object, "or"); disjunction {
			clause = "(" + clause + ")"
		}
	}
	return clause, err
}

func (compiler issueFilterCompiler) validate(node any, depth int) error {
	if depth > 5 {
		return newFilterError("Filter nesting is too deep (max 5); found depth "+strconv.Itoa(depth), "max_depth_exceeded")
	}
	object, ok := node.(map[string]any)
	if !ok {
		return newFilterError("Each filter node must be a JSON object", "invalid_filter_node")
	}
	if len(object) == 0 {
		return newFilterError("Filter objects must not be empty", "empty_filter_object")
	}
	logicalKeys := make([]string, 0, 1)
	for key := range object {
		switch strings.ToLower(key) {
		case "or", "and", "not":
			logicalKeys = append(logicalKeys, key)
		}
	}
	if len(logicalKeys) > 1 {
		return newFilterError("A filter object cannot contain multiple logical operators at the same level", "multiple_logical_operators")
	}
	if len(logicalKeys) == 1 {
		operatorKey := logicalKeys[0]
		if len(object) != 1 {
			return newFilterError("Cannot mix logical operator '"+operatorKey+"' with field keys at the same level", "mixed_operator_and_fields")
		}
		operator := strings.ToLower(operatorKey)
		value := object[operatorKey]
		if operator == "not" {
			child, ok := value.(map[string]any)
			if !ok {
				return newFilterError("'not' must be a single JSON object", "invalid_not_child")
			}
			return compiler.validate(child, depth+1)
		}
		children, ok := value.([]any)
		if !ok || len(children) == 0 {
			return newFilterError("'"+operator+"' must be a non-empty list of filter objects", "invalid_operator_children")
		}
		for _, child := range children {
			if _, ok := child.(map[string]any); !ok {
				return newFilterError("All children of '"+operator+"' must be JSON objects", "invalid_operator_child_type")
			}
			if err := compiler.validate(child, depth+1); err != nil {
				return err
			}
		}
		return nil
	}
	for field, value := range object {
		scratch := []any{}
		if _, handled, err := compiler.properties.field(field, value, &scratch); handled {
			if err != nil {
				return err
			}
			continue
		}
		if _, handled, err := compiler.typeField(field, value, &scratch); handled {
			if err != nil {
				return err
			}
			continue
		}
		if list, ok := value.([]any); ok {
			if len(list) == 0 {
				return newFilterError("List value for '"+field+"' must not be empty", "empty_list_value")
			}
			for _, item := range list {
				if !isIssueFilterScalar(item) {
					return newFilterError("List value for '"+field+"' must contain only scalar items", "non_scalar_list_item")
				}
			}
		} else if !isIssueFilterScalar(value) {
			return newFilterError("Value for '"+field+"' must be a scalar, null, or list/tuple of scalars", "invalid_value_type")
		}
		if _, ok := issueComplexFilterFields[field]; !ok {
			return newFilterError("Filtering on field '"+field+"' is not allowed", "invalid_filter_field")
		}
	}
	return nil
}

func (compiler issueFilterCompiler) render(node any, args *[]any, directRelations bool) (string, error) {
	object := node.(map[string]any)
	for _, operator := range []string{"or", "and", "not"} {
		value, exists := issueLogicalValue(object, operator)
		if !exists {
			continue
		}
		if operator == "not" {
			child, err := compiler.render(value, args, false)
			if err != nil {
				return "", err
			}
			return "NOT (" + child + ")", nil
		}
		children := value.([]any)
		if operator == "and" {
			leaves := make([]map[string]any, 0, len(children))
			allLeaves := true
			for _, child := range children {
				leaf := child.(map[string]any)
				for key := range leaf {
					switch strings.ToLower(key) {
					case "or", "and", "not":
						allLeaves = false
					}
				}
				leaves = append(leaves, leaf)
			}
			if allLeaves {
				return compiler.renderLeaves(leaves, args, directRelations)
			}
		}
		clauses := make([]string, 0, len(children))
		for _, child := range children {
			clause, err := compiler.render(child, args, directRelations)
			if err != nil {
				return "", err
			}
			if clause != "" {
				clauses = append(clauses, "("+clause+")")
			}
		}
		if len(clauses) == 0 {
			return "TRUE", nil
		}
		joiner := " AND "
		if operator == "or" {
			joiner = " OR "
		}
		return strings.Join(clauses, joiner), nil
	}
	return compiler.renderLeaves([]map[string]any{object}, args, directRelations)
}

func (compiler issueFilterCompiler) renderLeaves(leaves []map[string]any, args *[]any, directRelations bool) (string, error) {
	type relationCondition struct {
		field string
		value any
	}
	relations := make(map[string][]relationCondition)
	clauses := make([]string, 0)
	for _, object := range leaves {
		for field, value := range object {
			baseField := strings.TrimSuffix(strings.TrimSuffix(field, "__exact"), "__in")
			if _, related := issueComplexRelation(baseField); related {
				relations[baseField] = append(relations[baseField], relationCondition{field: field, value: value})
				continue
			}
			clause, err := compiler.renderField(field, value, args, directRelations)
			if err != nil {
				return "", err
			}
			if clause != "" {
				clauses = append(clauses, clause)
			}
		}
	}
	for baseField, conditions := range relations {
		relation, _ := issueComplexRelation(baseField)
		alias := "complex_relation"
		if directRelations {
			alias = "complex_join_" + baseField
		}
		predicates := make([]string, 0, len(conditions))
		for _, condition := range conditions {
			values := issueFilterValues(condition.value)
			if len(values) == 0 || (len(values) == 1 && values[0] == "") {
				continue
			}
			ids, err := issueFilterUUIDs(values, condition.field)
			if err != nil {
				return "", err
			}
			*args = append(*args, strings.Join(ids, ","))
			predicates = append(predicates, fmt.Sprintf(
				"%s.%s::text = ANY(string_to_array($%d, ','))",
				alias, relation[1], len(*args),
			))
		}
		if len(predicates) == 0 {
			continue
		}
		if directRelations {
			clauses = append(clauses, strings.Join(predicates, " AND "))
			continue
		}
		clauses = append(clauses, fmt.Sprintf(
			"EXISTS (SELECT 1 FROM %s complex_relation WHERE %s AND %s)",
			relation[0], issueRelationPredicate(relation[0], "complex_relation", true), strings.Join(predicates, " AND "),
		))
	}
	if len(clauses) == 0 {
		return "TRUE", nil
	}
	return strings.Join(clauses, " AND "), nil
}

func issueComplexRelation(field string) ([2]string, bool) {
	relation, exists := map[string][2]string{
		"assignee_id":   {"issue_assignees", "assignee_id"},
		"cycle_id":      {"cycle_issues", "cycle_id"},
		"module_id":     {"module_issues", "module_id"},
		"mention_id":    {"issue_mentions", "mention_id"},
		"label_id":      {"issue_labels", "label_id"},
		"subscriber_id": {"issue_subscribers", "subscriber_id"},
	}[field]
	return relation, exists
}
func issueComplexRelationJoins(raw string) []string {
	if raw == "" {
		return nil
	}
	var decoded any
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return nil
	}
	object, ok := decoded.(map[string]any)
	if !ok {
		return nil
	}
	type relationUse struct {
		inner bool
		left  bool
	}
	uses := make(map[string]relationUse)
	var collect func(map[string]any, bool)
	collect = func(node map[string]any, disjunctive bool) {
		for key, value := range node {
			switch strings.ToLower(key) {
			case "not":
				return
			case "or", "and":
				children, _ := value.([]any)
				for _, child := range children {
					if childObject, ok := child.(map[string]any); ok {
						collect(childObject, disjunctive || strings.EqualFold(key, "or"))
					}
				}
				return
			}
		}
		for field, value := range node {
			baseField := strings.TrimSuffix(strings.TrimSuffix(field, "__exact"), "__in")
			if _, related := issueComplexRelation(baseField); !related {
				continue
			}
			values := issueFilterValues(value)
			if len(values) == 0 || (len(values) == 1 && values[0] == "") {
				continue
			}
			use := uses[baseField]
			if disjunctive {
				use.left = true
			} else {
				use.inner = true
			}
			uses[baseField] = use
		}
	}
	collect(object, false)
	order := []string{"assignee_id", "cycle_id", "module_id", "mention_id", "label_id", "subscriber_id"}
	joins := make([]string, 0, len(uses))
	for _, baseField := range order {
		use, exists := uses[baseField]
		if !exists {
			continue
		}
		relation, _ := issueComplexRelation(baseField)
		join := "LEFT JOIN"
		if use.inner {
			join = "JOIN"
		}
		alias := "complex_join_" + baseField
		joins = append(joins, fmt.Sprintf("%s %s %s ON %s", join, relation[0], alias, issueRelationPredicate(relation[0], alias, true)))
	}
	return joins
}

func (compiler issueFilterCompiler) renderField(field string, value any, args *[]any, directRelations bool) (string, error) {
	if clause, handled, err := compiler.properties.field(field, value, args); handled {
		return clause, err
	}
	if clause, handled, err := compiler.typeField(field, value, args); handled {
		return clause, err
	}

	values := issueFilterValues(value)
	inLookup := strings.HasSuffix(field, "__in")
	baseField := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(field, "__exact"), "__in"), "__range")
	if len(values) == 0 || (len(values) == 1 && values[0] == "") {
		if inLookup {
			switch baseField {
			case "created_by_id", "state_id", "project_id", "state_group", "priority":
				return "FALSE", nil
			}
		}
		switch baseField {
		case "created_by_id":
			return "issue.created_by_id IS NULL", nil
		case "state_id":
			return "issue.state_id IS NULL", nil
		case "project_id":
			return "issue.project_id IS NULL", nil
		case "state_group":
			return `EXISTS (SELECT 1 FROM states complex_state WHERE complex_state.id = issue.state_id AND complex_state."group" = '')`, nil
		case "priority":
			return "issue.priority = ''", nil
		case "is_draft":
			return "issue.is_draft IS NULL", nil
		case "created_at":
			return cycleRequesterDateSQL("issue.created_at") + " IS NULL", nil
		case "updated_at":
			return cycleRequesterDateSQL("issue.updated_at") + " IS NULL", nil
		case "start_date":
			return "issue.start_date IS NULL", nil
		case "target_date":
			return "issue.target_date IS NULL", nil
		default:
			return "", nil
		}
	}
	switch baseField {
	case "assignee_id", "cycle_id", "module_id", "mention_id", "label_id", "subscriber_id":
		ids, err := issueFilterUUIDs(values, field)
		if err != nil {
			return "", err
		}
		*args = append(*args, strings.Join(ids, ","))
		relation := map[string][2]string{
			"assignee_id":   {"issue_assignees", "assignee_id"},
			"cycle_id":      {"cycle_issues", "cycle_id"},
			"module_id":     {"module_issues", "module_id"},
			"mention_id":    {"issue_mentions", "mention_id"},
			"label_id":      {"issue_labels", "label_id"},
			"subscriber_id": {"issue_subscribers", "subscriber_id"},
		}[baseField]
		if directRelations {
			return fmt.Sprintf(
				"complex_join_%s.%s::text = ANY(string_to_array($%d, ','))",
				baseField, relation[1], len(*args),
			), nil
		}
		return fmt.Sprintf(
			"EXISTS (SELECT 1 FROM %s complex_relation WHERE %s AND complex_relation.%s::text = ANY(string_to_array($%d, ',')))",
			relation[0], issueRelationPredicate(relation[0], "complex_relation", true), relation[1], len(*args),
		), nil
	case "created_by_id", "state_id", "project_id":
		ids, err := issueFilterUUIDs(values, field)
		if err != nil {
			return "", err
		}
		*args = append(*args, strings.Join(ids, ","))
		column := map[string]string{
			"created_by_id": "issue.created_by_id",
			"state_id":      "issue.state_id",
			"project_id":    "issue.project_id",
		}[baseField]
		return fmt.Sprintf("%s::text = ANY(string_to_array($%d, ','))", column, len(*args)), nil
	case "state_group", "priority":
		column := "issue.priority"
		if baseField == "state_group" {
			column = `(SELECT complex_state."group" FROM states complex_state WHERE complex_state.id = issue.state_id)`
		}
		*args = append(*args, values[0])
		if inLookup {
			return fmt.Sprintf("%s = ANY(string_to_array($%d, ','))", column, len(*args)), nil
		}
		return fmt.Sprintf("%s = $%d", column, len(*args)), nil
	case "is_archived", "is_draft":
		boolean, ok := issueFilterBoolean(value)
		if !ok {
			if baseField == "is_archived" {
				return "", nil
			}
			return "issue.is_draft IS NULL", nil
		}
		if baseField == "is_archived" {
			if boolean {
				return "issue.archived_at IS NOT NULL", nil
			}
			return "issue.archived_at IS NULL", nil
		}
		*args = append(*args, boolean)
		return fmt.Sprintf("issue.is_draft = $%d", len(*args)), nil
	case "created_at", "updated_at", "start_date", "target_date":
		column := map[string]string{
			"created_at":  cycleRequesterDateSQL("issue.created_at"),
			"updated_at":  cycleRequesterDateSQL("issue.updated_at"),
			"start_date":  "issue.start_date",
			"target_date": "issue.target_date",
		}[baseField]
		if strings.HasSuffix(field, "__range") {
			if len(values) == 1 {
				values = strings.Split(values[0], ",")
			}
			if len(values) != 2 {
				return "", invalidIssueFilterset(field, "Range query expects two values.")
			}
			start, startOK := normalizeIssueFilterDate(values[0])
			end, endOK := normalizeIssueFilterDate(values[1])
			if !startOK || !endOK {
				return "", invalidIssueFilterset(field, "Range query expects two values.")
			}
			*args = append(*args, start, end)
			return fmt.Sprintf("%s BETWEEN $%d::date AND $%d::date", column, len(*args)-1, len(*args)), nil
		}
		date, ok := normalizeIssueFilterDate(values[0])
		if !ok {
			return "", invalidIssueFilterset(field, "Enter a valid date.")
		}
		*args = append(*args, date)
		return fmt.Sprintf("%s = $%d::date", column, len(*args)), nil
	}
	return "", nil
}

func cycleRequesterDateSQL(column string) string {
	return `((
      ` + column + ` AT TIME ZONE COALESCE(
        (SELECT NULLIF(requester_user.user_timezone, '') FROM users requester_user WHERE requester_user.id = $1),
        'UTC'
      )
    )::date)`
}

func issueFilterValues(value any) []string {
	if list, ok := value.([]any); ok {
		if len(list) == 0 {
			return nil
		}
		return []string{fmt.Sprint(list[len(list)-1])}
	}
	if value == nil {
		return []string{""}
	}
	return []string{fmt.Sprint(value)}
}

func issueFilterUUIDs(values []string, field string) ([]string, error) {
	if strings.HasSuffix(field, "__in") && len(values) == 1 && strings.Contains(values[0], ",") {
		values = strings.Split(values[0], ",")
	}
	for _, value := range values {
		if _, err := uuid.Parse(value); err != nil {
			return nil, invalidIssueFilterset(field, "Enter a valid UUID.")
		}
	}
	return values, nil
}

func issueFilterBoolean(value any) (bool, bool) {
	if values, ok := value.([]any); ok {
		if len(values) == 0 {
			return false, false
		}
		return issueFilterBoolean(values[len(values)-1])
	}
	switch typed := value.(type) {
	case bool:
		return typed, true
	case string:
		switch typed {
		case "true", "True", "2":
			return true, true
		case "false", "False", "3":
			return false, true
		}
	}
	return false, false
}

func normalizeIssueFilterDate(value string) (string, bool) {
	value = strings.TrimSpace(value)
	for _, layout := range []string{
		"2006-01-02", "2006-1-2",
		"01/02/2006", "1/2/2006", "01/02/06", "1/2/06",
		"Jan 2 2006", "Jan 2, 2006",
		"2 Jan 2006", "2 Jan, 2006",
		"January 2 2006", "January 2, 2006",
		"2 January 2006", "2 January, 2006",
	} {
		if date, err := time.Parse(layout, value); err == nil {
			return date.Format("2006-01-02"), true
		}
	}
	return "", false
}

func isIssueFilterScalar(value any) bool {
	switch value.(type) {
	case nil, string, float64, bool:
		return true
	default:
		return false
	}
}

func invalidIssueFilterset(field, message string) error {
	return &FilterError{
		Message: "Invalid filter parameters",
		Code:    "invalid_filterset",
		Errors:  map[string]any{field: []string{message}},
	}
}

func newFilterError(message, code string) error {
	return &FilterError{Message: message, Code: code}
}

func legacyDatePredicates(value string, column string, args *[]any) []string {
	type dateClause struct {
		value    any
		template string
	}
	clauses := make(map[string]dateClause, 3)
	for _, query := range strings.Split(value, ",") {
		parts := strings.Split(query, ";")
		if len(parts) == 1 {
			clauses["contains"] = dateClause{
				value: parts[0], template: column + "::date::text LIKE '%%' || $%d || '%%'",
			}
			continue
		}
		durationParts := strings.SplitN(parts[0], "_", 2)
		isDuration := len(durationParts) == 2 && (durationParts[1] == "weeks" || durationParts[1] == "months")
		if isDuration {
			if len(parts) != 3 {
				continue
			}
			duration, err := strconv.Atoi(durationParts[0])
			if err != nil {
				continue
			}
			days := duration * 7
			if durationParts[1] == "months" {
				days = duration * 30
			}
			if parts[2] != "fromnow" {
				days = -days
			}
			key, operator := "lte", "<="
			if parts[1] == "after" {
				key, operator = "gte", ">="
			}
			clauses[key] = dateClause{
				value: days, template: column + "::date " + operator + " CURRENT_DATE + ($%d * interval '1 day')",
			}
			continue
		}
		key, operator := "lte", "<="
		if slicesContain(parts, "after") {
			key, operator = "gte", ">="
		}
		clauses[key] = dateClause{value: parts[0], template: column + "::date " + operator + " $%d::date"}
	}
	predicates := make([]string, 0, len(clauses))
	for _, key := range []string{"contains", "gte", "lte"} {
		clause, ok := clauses[key]
		if !ok {
			continue
		}
		*args = append(*args, clause.value)
		predicates = append(predicates, fmt.Sprintf(clause.template, len(*args)))
	}
	return predicates
}

func slicesContain(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func hasNoneToken(value string) bool {
	for _, token := range strings.Split(value, ",") {
		if token == "None" {
			return true
		}
	}
	return false
}

func CSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" && part != "null" {
			result = append(result, part)
		}
	}
	return result
}

func UUIDs(value string) []uuid.UUID {
	parts := CSV(value)
	result := make([]uuid.UUID, 0, len(parts))
	for _, part := range parts {
		if parsed, err := uuid.Parse(part); err == nil {
			result = append(result, parsed)
		}
	}
	return result
}

const workspaceAuthorizationSQL = `
SELECT EXISTS (
  SELECT 1
  FROM workspace_members workspace_member
  JOIN workspaces workspace ON workspace.id = workspace_member.workspace_id
  WHERE workspace_member.member_id = $1
    AND workspace.slug = $2
    AND workspace_member.is_active = true
    AND workspace_member.deleted_at IS NULL
    AND workspace.deleted_at IS NULL
)`

const projectAuthorizationSQL = `
SELECT EXISTS (
  SELECT 1
  FROM project_members project_member
  JOIN projects project ON project.id = project_member.project_id
  JOIN workspaces workspace ON workspace.id = project_member.workspace_id
  WHERE project_member.member_id = $1
    AND workspace.slug = $2
    AND project.id = $3
    AND project_member.is_active = true
    AND project_member.deleted_at IS NULL
    AND project.deleted_at IS NULL
    AND workspace.deleted_at IS NULL
    AND (
      project_member.role IN (15, 20)
      OR ($4::boolean AND project_member.role = 5)
      OR EXISTS (
        SELECT 1
        FROM workspace_members workspace_admin
        WHERE workspace_admin.workspace_id = workspace.id
          AND workspace_admin.member_id = $1
          AND workspace_admin.role = 20
          AND workspace_admin.is_active = true
          AND workspace_admin.deleted_at IS NULL
      )
    )
)`

const cycleStatusSQL = `CASE
  WHEN cycle.start_date <= now() AND cycle.end_date >= now() THEN 'CURRENT'
  WHEN cycle.start_date > now() THEN 'UPCOMING'
  WHEN cycle.end_date < now() THEN 'COMPLETED'
  ELSE 'DRAFT'
END`

func projectTimestampSQL(column string) string {
	local := "(" + column + " AT TIME ZONE project.timezone)"
	offset := "(" + local + " - (" + column + " AT TIME ZONE 'UTC'))"
	return `CASE WHEN ` + column + ` IS NULL THEN NULL ELSE
  to_char(` + local + `,'YYYY-MM-DD"T"HH24:MI:SS') ||
  CASE WHEN date_trunc('second',` + local + `)=` + local + ` THEN '' ELSE '.' || to_char(` + local + `,'US') END ||
  CASE WHEN extract(epoch FROM ` + offset + `)=0 THEN 'Z' ELSE
    CASE WHEN extract(epoch FROM ` + offset + `)<0 THEN '-' ELSE '+' END ||
    lpad(floor(abs(extract(epoch FROM ` + offset + `))/3600)::int::text,2,'0') || ':' ||
    lpad(floor(mod(abs(extract(epoch FROM ` + offset + `)),3600)/60)::int::text,2,'0')
  END
END`
}

func utcTimestampSQL(column string) string {
	utc := "(" + column + " AT TIME ZONE 'UTC')"
	return `CASE WHEN ` + column + ` IS NULL THEN NULL ELSE
  to_char(` + utc + `,'YYYY-MM-DD"T"HH24:MI:SS') ||
  CASE WHEN date_trunc('second',` + utc + `)=` + utc + ` THEN '' ELSE '.' || to_char(` + utc + `,'US') END || 'Z'
END`
}

func userTimestampSQL(column string) string {
	zone := `COALESCE(
    (SELECT NULLIF(requester_user.user_timezone, '') FROM users requester_user WHERE requester_user.id = $1),
    'UTC'
  )`
	local := "(" + column + " AT TIME ZONE " + zone + ")"
	offset := "(" + local + " - (" + column + " AT TIME ZONE 'UTC'))"
	return `CASE WHEN ` + column + ` IS NULL THEN NULL ELSE
  to_char(` + local + `,'YYYY-MM-DD"T"HH24:MI:SS') ||
  CASE WHEN date_trunc('second',` + local + `)=` + local + ` THEN '' ELSE '.' || to_char(` + local + `,'US') END ||
  CASE WHEN extract(epoch FROM ` + offset + `)=0 THEN 'Z' ELSE
    CASE WHEN extract(epoch FROM ` + offset + `)<0 THEN '-' ELSE '+' END ||
    lpad(floor(abs(extract(epoch FROM ` + offset + `))/3600)::int::text,2,'0') || ':' ||
    lpad(floor(mod(abs(extract(epoch FROM ` + offset + `)),3600)/60)::int::text,2,'0')
  END
END`
}

const cycleIssueCountsSQL = `
SELECT
  count(DISTINCT issue.id) FILTER (WHERE issue.archived_at IS NULL AND issue.is_draft=false AND issue.deleted_at IS NULL),
  count(DISTINCT issue.id) FILTER (WHERE state."group"='completed' AND issue.archived_at IS NULL AND issue.is_draft=false AND issue.deleted_at IS NULL),
  count(DISTINCT issue.id) FILTER (WHERE state."group"='cancelled' AND issue.archived_at IS NULL AND issue.is_draft=false AND issue.deleted_at IS NULL),
  count(DISTINCT issue.id) FILTER (WHERE state."group"='started' AND issue.archived_at IS NULL AND issue.is_draft=false AND issue.deleted_at IS NULL),
  count(DISTINCT issue.id) FILTER (WHERE state."group"='unstarted' AND issue.archived_at IS NULL AND issue.is_draft=false AND issue.deleted_at IS NULL),
  count(DISTINCT issue.id) FILTER (WHERE state."group"='backlog' AND issue.archived_at IS NULL AND issue.is_draft=false AND issue.deleted_at IS NULL)
FROM cycle_issues cycle_issue
JOIN issues issue ON issue.id=cycle_issue.issue_id
LEFT JOIN states state ON state.id=issue.state_id
WHERE cycle_issue.cycle_id=cycle.id AND cycle_issue.deleted_at IS NULL`

var workspaceCycleIssueCountsSQL = strings.ReplaceAll(
	cycleIssueCountsSQL,
	"count(DISTINCT issue.id)",
	"count(cycle_issue.id)",
)

var archivedCycleIssueCountsSQL = strings.Replace(
	cycleIssueCountsSQL,
	" AND cycle_issue.deleted_at IS NULL",
	"",
	1,
)

var workspaceCycleListSQL = `
SELECT COALESCE(json_agg(json_build_object(
  'id', cycle.id, 'workspace_id', cycle.workspace_id, 'project_id', cycle.project_id,
  'name', cycle.name, 'description', cycle.description, 'start_date', ` + userTimestampSQL("cycle.start_date") + `,
  'end_date', ` + userTimestampSQL("cycle.end_date") + `, 'owned_by_id', cycle.owned_by_id, 'view_props', cycle.view_props,
  'sort_order', cycle.sort_order, 'external_source', cycle.external_source, 'external_id', cycle.external_id,
  'progress_snapshot', cycle.progress_snapshot, 'logo_props', cycle.logo_props,
  'total_issues', counts.total, 'completed_issues', counts.completed, 'cancelled_issues', counts.cancelled,
  'started_issues', counts.started, 'unstarted_issues', counts.unstarted, 'backlog_issues', counts.backlog
) ORDER BY cycle.created_at DESC), '[]'::json)
FROM cycles cycle
JOIN projects project ON project.id=cycle.project_id
JOIN workspaces workspace ON workspace.id=cycle.workspace_id
JOIN LATERAL (` + workspaceCycleIssueCountsSQL + `) raw_counts(total,completed,cancelled,started,unstarted,backlog) ON true
CROSS JOIN LATERAL (SELECT raw_counts.total,raw_counts.completed,raw_counts.cancelled,raw_counts.started,raw_counts.unstarted,raw_counts.backlog) counts
WHERE workspace.slug=$2
  AND cycle.deleted_at IS NULL
  AND cycle.archived_at IS NULL
  AND project.deleted_at IS NULL
  AND project.archived_at IS NULL
  AND EXISTS (
    SELECT 1 FROM project_members project_member
    WHERE project_member.project_id=project.id AND project_member.member_id=$1
      AND project_member.is_active=true AND project_member.deleted_at IS NULL
  )`

var projectCycleListSQL = `
SELECT COALESCE(json_agg(item ORDER BY favorite DESC, created_at DESC), '[]'::json)
FROM (
  SELECT cycle.created_at, EXISTS (
      SELECT 1 FROM user_favorites favorite
      WHERE favorite.user_id=$1 AND favorite.entity_type='cycle' AND favorite.entity_identifier=cycle.id
        AND favorite.project_id=project.id AND favorite.workspace_id=workspace.id AND favorite.deleted_at IS NULL
    ) AS favorite,
    json_build_object(
      'id', cycle.id, 'workspace_id', cycle.workspace_id, 'project_id', cycle.project_id,
      'name', cycle.name, 'description', cycle.description,
      'start_date', ` + projectTimestampSQL("cycle.start_date") + `,
      'end_date', ` + projectTimestampSQL("cycle.end_date") + `,
      'owned_by_id', cycle.owned_by_id, 'view_props', cycle.view_props, 'sort_order', cycle.sort_order,
      'external_source', cycle.external_source, 'external_id', cycle.external_id,
      'progress_snapshot', cycle.progress_snapshot, 'logo_props', cycle.logo_props,
      'is_favorite', EXISTS (
        SELECT 1 FROM user_favorites favorite
        WHERE favorite.user_id=$1 AND favorite.entity_type='cycle' AND favorite.entity_identifier=cycle.id
          AND favorite.project_id=project.id AND favorite.workspace_id=workspace.id AND favorite.deleted_at IS NULL
      ),
      'total_issues', counts.total, 'completed_issues', counts.completed, 'cancelled_issues', counts.cancelled,
      'assignee_ids', COALESCE((
        SELECT json_agg(DISTINCT assignee.assignee_id)
        FROM cycle_issues cycle_issue JOIN issue_assignees assignee ON assignee.issue_id=cycle_issue.issue_id
        WHERE cycle_issue.cycle_id=cycle.id AND assignee.deleted_at IS NULL
      ), '[]'::json),
      'status', ` + cycleStatusSQL + `, 'version', cycle.version, 'created_by', cycle.created_by_id
    ) item
  FROM cycles cycle
  JOIN projects project ON project.id=cycle.project_id
  JOIN workspaces workspace ON workspace.id=cycle.workspace_id
  JOIN LATERAL (` + cycleIssueCountsSQL + `) raw_counts(total,completed,cancelled,started,unstarted,backlog) ON true
  CROSS JOIN LATERAL (SELECT raw_counts.total,raw_counts.completed,raw_counts.cancelled) counts
  WHERE workspace.slug=$2 AND project.id=$3 AND cycle.deleted_at IS NULL AND cycle.archived_at IS NULL
    AND project.deleted_at IS NULL AND project.archived_at IS NULL
    AND (
      NOT $4::boolean
      OR (cycle.start_date <= now() AND cycle.end_date >= now())
    )
) cycles`

var archivedCycleListSQL = `
SELECT COALESCE(json_agg(item ORDER BY favorite DESC, created_at DESC), '[]'::json)
FROM (
  SELECT cycle.created_at, EXISTS (
      SELECT 1 FROM user_favorites favorite WHERE favorite.user_id=$1 AND favorite.entity_type='cycle'
        AND favorite.entity_identifier=cycle.id AND favorite.project_id=project.id
        AND favorite.workspace_id=workspace.id AND favorite.deleted_at IS NULL
    ) favorite,
    json_build_object(
      'id',cycle.id,'workspace_id',cycle.workspace_id,'project_id',cycle.project_id,'name',cycle.name,
      'description',cycle.description,'start_date',` + utcTimestampSQL("cycle.start_date") + `,'end_date',` + utcTimestampSQL("cycle.end_date") + `,
      'owned_by_id',cycle.owned_by_id,'view_props',cycle.view_props,'sort_order',cycle.sort_order,
      'external_source',cycle.external_source,'external_id',cycle.external_id,'progress_snapshot',cycle.progress_snapshot,
      'total_issues',counts.total,'is_favorite',EXISTS (
        SELECT 1 FROM user_favorites favorite WHERE favorite.user_id=$1 AND favorite.entity_type='cycle'
          AND favorite.entity_identifier=cycle.id AND favorite.project_id=project.id
          AND favorite.workspace_id=workspace.id AND favorite.deleted_at IS NULL
      ),'cancelled_issues',counts.cancelled,'completed_issues',counts.completed,'started_issues',counts.started,
      'unstarted_issues',counts.unstarted,'backlog_issues',counts.backlog,
      'assignee_ids',COALESCE((SELECT json_agg(DISTINCT assignee.assignee_id) FROM cycle_issues ci
        JOIN issue_assignees assignee ON assignee.issue_id=ci.issue_id
        WHERE ci.cycle_id=cycle.id),'[]'::json),
      'status',` + cycleStatusSQL + `,'archived_at',` + utcTimestampSQL("cycle.archived_at") + `
    ) item
  FROM cycles cycle
  JOIN projects project ON project.id=cycle.project_id
  JOIN workspaces workspace ON workspace.id=cycle.workspace_id
  JOIN LATERAL (` + archivedCycleIssueCountsSQL + `) raw_counts(total,completed,cancelled,started,unstarted,backlog) ON true
  CROSS JOIN LATERAL (SELECT raw_counts.*) counts
  WHERE workspace.slug=$2 AND project.id=$3 AND cycle.deleted_at IS NULL AND cycle.archived_at IS NOT NULL
    AND project.deleted_at IS NULL AND project.archived_at IS NULL
) cycles`

var issueDistributionSQL = `
SELECT json_build_object(
  'assignees', COALESCE((
    SELECT json_agg(row_to_json(distribution) ORDER BY distribution.first_name, distribution.last_name)
    FROM (
      SELECT user_account.first_name,user_account.last_name,user_account.id assignee_id,
        CASE WHEN user_account.avatar_asset_id IS NOT NULL THEN '/api/assets/v2/static/'||user_account.avatar_asset_id||'/' ELSE user_account.avatar END avatar_url,
        user_account.display_name,
        count(issue.id) FILTER (WHERE issue.archived_at IS NULL AND issue.is_draft=false) total_issues,
        count(issue.id) FILTER (WHERE issue.completed_at IS NOT NULL AND issue.archived_at IS NULL AND issue.is_draft=false) completed_issues,
        count(issue.id) FILTER (WHERE issue.completed_at IS NULL AND issue.archived_at IS NULL AND issue.is_draft=false) pending_issues
      FROM cycle_issues ci JOIN issues issue ON issue.id=ci.issue_id AND issue.workspace_id=workspace.id AND issue.project_id=project.id AND project.archived_at IS NULL
      LEFT JOIN states issue_state ON issue_state.id=issue.state_id
      LEFT JOIN issue_assignees assignee ON assignee.issue_id=issue.id
      LEFT JOIN users user_account ON user_account.id=assignee.assignee_id
      WHERE ci.cycle_id=cycle.id AND ci.deleted_at IS NULL AND issue.deleted_at IS NULL
        AND issue.archived_at IS NULL AND issue.is_draft=false AND issue_state."group" IS DISTINCT FROM 'triage'
      GROUP BY user_account.id,user_account.first_name,user_account.last_name,user_account.avatar_asset_id,user_account.avatar,user_account.display_name
    ) distribution
  ),'[]'::json),
  'labels', COALESCE((
    SELECT json_agg(row_to_json(distribution) ORDER BY distribution.label_name)
    FROM (
      SELECT label.name label_name,label.color,label.id label_id,
        count(issue.id) FILTER (WHERE issue.archived_at IS NULL AND issue.is_draft=false) total_issues,
        count(issue.id) FILTER (WHERE issue.completed_at IS NOT NULL AND issue.archived_at IS NULL AND issue.is_draft=false) completed_issues,
        count(issue.id) FILTER (WHERE issue.completed_at IS NULL AND issue.archived_at IS NULL AND issue.is_draft=false) pending_issues
      FROM cycle_issues ci JOIN issues issue ON issue.id=ci.issue_id AND issue.workspace_id=workspace.id AND issue.project_id=project.id AND project.archived_at IS NULL
      LEFT JOIN states issue_state ON issue_state.id=issue.state_id
      LEFT JOIN issue_labels issue_label ON ` + issueRelationPredicate("issue_labels", "issue_label", true) + `
      LEFT JOIN labels label ON label.id=issue_label.label_id
      WHERE ci.cycle_id=cycle.id AND ci.deleted_at IS NULL AND issue.deleted_at IS NULL
        AND issue.archived_at IS NULL AND issue.is_draft=false AND issue_state."group" IS DISTINCT FROM 'triage'
      GROUP BY label.id,label.name,label.color
    ) distribution
  ),'[]'::json),
  'completion_chart', ` + issueBurndownSQL + `
)`

var analyticsIssueDistributionSQL = `
SELECT json_build_object(
  'assignees', COALESCE((
    SELECT json_agg(row_to_json(distribution) ORDER BY distribution.display_name)
    FROM (
      SELECT user_account.display_name,user_account.id assignee_id,
        CASE WHEN user_account.avatar_asset_id IS NOT NULL THEN '/api/assets/v2/static/'||user_account.avatar_asset_id||'/' ELSE user_account.avatar END avatar_url,
        count(assignee.assignee_id) total_issues,
        count(assignee.assignee_id) FILTER (WHERE issue.completed_at IS NOT NULL) completed_issues,
        count(assignee.assignee_id) FILTER (WHERE issue.completed_at IS NULL) pending_issues
      FROM cycle_issues ci JOIN issues issue ON issue.id=ci.issue_id AND issue.workspace_id=workspace.id AND issue.project_id=project.id AND project.archived_at IS NULL
      LEFT JOIN states issue_state ON issue_state.id=issue.state_id
      LEFT JOIN issue_assignees assignee ON assignee.issue_id=issue.id
      LEFT JOIN users user_account ON user_account.id=assignee.assignee_id
      WHERE ci.cycle_id=cycle.id AND ci.deleted_at IS NULL AND issue.deleted_at IS NULL
        AND issue.archived_at IS NULL AND issue.is_draft=false AND issue_state."group" IS DISTINCT FROM 'triage'
      GROUP BY user_account.id,user_account.display_name,user_account.avatar_asset_id,user_account.avatar
    ) distribution
  ),'[]'::json),
  'labels', COALESCE((
    SELECT json_agg(row_to_json(distribution) ORDER BY distribution.label_name)
    FROM (
      SELECT label.name label_name,label.color,label.id label_id,
        count(issue_label.label_id) total_issues,
        count(issue_label.label_id) FILTER (WHERE issue.completed_at IS NOT NULL) completed_issues,
        count(issue_label.label_id) FILTER (WHERE issue.completed_at IS NULL) pending_issues
      FROM cycle_issues ci JOIN issues issue ON issue.id=ci.issue_id AND issue.workspace_id=workspace.id AND issue.project_id=project.id AND project.archived_at IS NULL
      LEFT JOIN states issue_state ON issue_state.id=issue.state_id
      LEFT JOIN issue_labels issue_label ON ` + issueRelationPredicate("issue_labels", "issue_label", true) + `
      LEFT JOIN labels label ON label.id=issue_label.label_id
      WHERE ci.cycle_id=cycle.id AND ci.deleted_at IS NULL AND issue.deleted_at IS NULL
        AND issue.archived_at IS NULL AND issue.is_draft=false AND issue_state."group" IS DISTINCT FROM 'triage'
      GROUP BY label.id,label.name,label.color
    ) distribution
  ),'[]'::json),
  'completion_chart', ` + issueBurndownSQL + `
)`

var estimateDistributionSQL = `
SELECT json_build_object(
  'assignees', COALESCE((
    SELECT json_agg(row_to_json(distribution) ORDER BY distribution.display_name)
    FROM (
      SELECT user_account.display_name,user_account.id assignee_id,
        CASE WHEN user_account.avatar_asset_id IS NOT NULL THEN '/api/assets/v2/static/'||user_account.avatar_asset_id||'/' ELSE user_account.avatar END avatar_url,
        sum(estimate_point.value::float8) total_estimates,
        sum(estimate_point.value::float8) FILTER (WHERE issue.completed_at IS NOT NULL AND issue.archived_at IS NULL AND issue.is_draft=false) completed_estimates,
        sum(estimate_point.value::float8) FILTER (WHERE issue.completed_at IS NULL AND issue.archived_at IS NULL AND issue.is_draft=false) pending_estimates
      FROM cycle_issues ci JOIN issues issue ON issue.id=ci.issue_id AND issue.workspace_id=workspace.id AND issue.project_id=project.id AND project.archived_at IS NULL
      LEFT JOIN states issue_state ON issue_state.id=issue.state_id
      LEFT JOIN estimate_points estimate_point ON ` + issuequery.LiveNumericEstimatePredicate("issue", "estimate_point") + `
      LEFT JOIN issue_assignees assignee ON assignee.issue_id=issue.id
      LEFT JOIN users user_account ON user_account.id=assignee.assignee_id
      WHERE ci.cycle_id=cycle.id AND ci.deleted_at IS NULL AND issue.deleted_at IS NULL
        AND issue.archived_at IS NULL AND issue.is_draft=false AND issue_state."group" IS DISTINCT FROM 'triage'
      GROUP BY user_account.id,user_account.display_name,user_account.avatar_asset_id,user_account.avatar
    ) distribution
  ),'[]'::json),
  'labels', COALESCE((
    SELECT json_agg(row_to_json(distribution) ORDER BY distribution.label_name)
    FROM (
      SELECT label.name label_name,label.color,label.id label_id,
        sum(estimate_point.value::float8) total_estimates,
        sum(estimate_point.value::float8) FILTER (WHERE issue.completed_at IS NOT NULL AND issue.archived_at IS NULL AND issue.is_draft=false) completed_estimates,
        sum(estimate_point.value::float8) FILTER (WHERE issue.completed_at IS NULL AND issue.archived_at IS NULL AND issue.is_draft=false) pending_estimates
      FROM cycle_issues ci JOIN issues issue ON issue.id=ci.issue_id AND issue.workspace_id=workspace.id AND issue.project_id=project.id AND project.archived_at IS NULL
      LEFT JOIN states issue_state ON issue_state.id=issue.state_id
      LEFT JOIN estimate_points estimate_point ON ` + issuequery.LiveNumericEstimatePredicate("issue", "estimate_point") + `
      LEFT JOIN issue_labels issue_label ON ` + issueRelationPredicate("issue_labels", "issue_label", true) + `
      LEFT JOIN labels label ON label.id=issue_label.label_id
      WHERE ci.cycle_id=cycle.id AND ci.deleted_at IS NULL AND issue.deleted_at IS NULL
        AND issue.archived_at IS NULL AND issue.is_draft=false AND issue_state."group" IS DISTINCT FROM 'triage'
      GROUP BY label.id,label.name,label.color
    ) distribution
  ),'[]'::json),
  'completion_chart', ` + estimateBurndownSQL + `
)`

var issueBurndownSQL = `COALESCE((
  SELECT json_object_agg(day::date,
    CASE WHEN day::date > CURRENT_DATE THEN NULL ELSE
      counts.total - (SELECT count(completed_issue.id) FROM cycle_issues completed_ci
        JOIN issues completed_issue ON completed_issue.id=completed_ci.issue_id AND completed_issue.workspace_id=workspace.id AND completed_issue.project_id=project.id AND project.archived_at IS NULL
        LEFT JOIN states completed_state ON completed_state.id=completed_issue.state_id
        WHERE completed_ci.cycle_id=cycle.id AND completed_ci.deleted_at IS NULL
          AND completed_issue.deleted_at IS NULL AND completed_issue.archived_at IS NULL
          AND completed_issue.is_draft=false AND completed_state."group" IS DISTINCT FROM 'triage'
          AND ` + cycleRequesterDateSQL("completed_issue.completed_at") + ` <= day::date)
    END ORDER BY day)
  FROM generate_series(cycle.start_date::date,cycle.end_date::date,interval '1 day') day
  CROSS JOIN LATERAL (SELECT count(DISTINCT total_issue.id) total FROM cycle_issues total_ci
    JOIN issues total_issue ON total_issue.id=total_ci.issue_id
    WHERE total_ci.cycle_id=cycle.id AND total_ci.deleted_at IS NULL AND total_issue.deleted_at IS NULL
      AND total_issue.archived_at IS NULL AND total_issue.is_draft=false) counts
),'{}'::json)`

var archivedIssueBurndownSQL = strings.Replace(
	issueBurndownSQL,
	" AND total_ci.deleted_at IS NULL",
	"",
	1,
)

var archivedIssueDistributionSQL = strings.Replace(
	issueDistributionSQL,
	issueBurndownSQL,
	archivedIssueBurndownSQL,
	1,
)

var estimateBurndownSQL = `COALESCE((
  SELECT json_object_agg(day::date,
    CASE WHEN day::date > CURRENT_DATE THEN NULL ELSE
      counts.total - COALESCE((SELECT sum(completed_point.value::float8) FROM cycle_issues completed_ci
        JOIN issues completed_issue ON completed_issue.id=completed_ci.issue_id AND completed_issue.workspace_id=workspace.id AND completed_issue.project_id=project.id AND project.archived_at IS NULL
        LEFT JOIN states completed_state ON completed_state.id=completed_issue.state_id
        JOIN estimate_points completed_point ON ` + issuequery.LiveNumericEstimatePredicate("completed_issue", "completed_point") + `
        WHERE completed_ci.cycle_id=cycle.id AND completed_ci.deleted_at IS NULL
          AND completed_issue.deleted_at IS NULL AND completed_issue.archived_at IS NULL
          AND completed_issue.is_draft=false AND completed_state."group" IS DISTINCT FROM 'triage'
          AND ` + cycleRequesterDateSQL("completed_issue.completed_at") + ` <= day::date),0)
    END ORDER BY day)
  FROM generate_series(cycle.start_date::date,cycle.end_date::date,interval '1 day') day
  CROSS JOIN LATERAL (SELECT COALESCE(sum(total_point.value::float8),0) total FROM cycle_issues total_ci
    JOIN issues total_issue ON total_issue.id=total_ci.issue_id AND total_issue.workspace_id=workspace.id AND total_issue.project_id=project.id AND project.archived_at IS NULL
    LEFT JOIN states total_state ON total_state.id=total_issue.state_id
    JOIN estimate_points total_point ON ` + issuequery.LiveNumericEstimatePredicate("total_issue", "total_point") + `
    WHERE total_ci.cycle_id=cycle.id AND total_ci.deleted_at IS NULL AND total_issue.deleted_at IS NULL
      AND total_issue.archived_at IS NULL AND total_issue.is_draft=false AND total_state."group" IS DISTINCT FROM 'triage') counts
),'{}'::json)`

var archivedCycleDetailSQL = `
SELECT json_build_object(
  'id',cycle.id,'workspace_id',cycle.workspace_id,'project_id',cycle.project_id,'name',cycle.name,
  'description',cycle.description,'start_date',` + utcTimestampSQL("cycle.start_date") + `,'end_date',` + utcTimestampSQL("cycle.end_date") + `,'owned_by_id',cycle.owned_by_id,
  'view_props',cycle.view_props,'sort_order',cycle.sort_order,'external_source',cycle.external_source,
  'external_id',cycle.external_id,'progress_snapshot',cycle.progress_snapshot,
  'sub_issues',(SELECT count(*) FROM cycle_issues ci JOIN issues issue ON issue.id=ci.issue_id AND issue.workspace_id=workspace.id AND issue.project_id=project.id
    LEFT JOIN states issue_state ON issue_state.id=issue.state_id
    WHERE ci.cycle_id=cycle.id AND ci.deleted_at IS NULL AND issue.deleted_at IS NULL AND issue.parent_id IS NOT NULL
      AND issue.archived_at IS NULL AND issue.is_draft=false AND issue_state."group" IS DISTINCT FROM 'triage'),
  'logo_props',cycle.logo_props,
  'completed_estimate_points',COALESCE((SELECT sum(point.value::float8) FROM cycle_issues ci JOIN issues issue ON issue.id=ci.issue_id
    JOIN states state ON state.id=issue.state_id JOIN estimate_points point ON ` + issuequery.LiveNumericEstimatePredicate("issue", "point") + `
    WHERE ci.cycle_id=cycle.id AND ci.deleted_at IS NULL AND issue.deleted_at IS NULL
      AND issue.archived_at IS NULL AND issue.is_draft=false AND state."group"='completed'),0),
  'total_estimate_points',COALESCE((SELECT sum(point.value::float8) FROM cycle_issues ci JOIN issues issue ON issue.id=ci.issue_id
    LEFT JOIN states state ON state.id=issue.state_id
    JOIN estimate_points point ON ` + issuequery.LiveNumericEstimatePredicate("issue", "point") + `
    WHERE ci.cycle_id=cycle.id AND ci.deleted_at IS NULL AND issue.deleted_at IS NULL
      AND issue.archived_at IS NULL AND issue.is_draft=false AND state."group" IS DISTINCT FROM 'triage'),0),
  'is_favorite',EXISTS(SELECT 1 FROM user_favorites favorite WHERE favorite.user_id=$1 AND favorite.entity_type='cycle'
    AND favorite.entity_identifier=cycle.id AND favorite.project_id=project.id AND favorite.workspace_id=workspace.id AND favorite.deleted_at IS NULL),
  'total_issues',counts.total,'cancelled_issues',counts.cancelled,'completed_issues',counts.completed,
  'started_issues',counts.started,'unstarted_issues',counts.unstarted,'backlog_issues',counts.backlog,
  'assignee_ids',COALESCE((SELECT json_agg(DISTINCT assignee.assignee_id) FROM cycle_issues ci
    JOIN issue_assignees assignee ON assignee.issue_id=ci.issue_id
    WHERE ci.cycle_id=cycle.id),'[]'::json),
  'status',` + cycleStatusSQL + `,'created_by',cycle.created_by_id,'archived_at',` + utcTimestampSQL("cycle.archived_at") + `,
  'estimate_distribution',CASE WHEN EXISTS(SELECT 1 FROM estimates estimate WHERE estimate.id=project.estimate_id AND estimate.type='points')
    THEN (` + estimateDistributionSQL + `) ELSE '{}'::json END,
  'distribution',(` + archivedIssueDistributionSQL + `)
)
FROM cycles cycle
JOIN projects project ON project.id=cycle.project_id
JOIN workspaces workspace ON workspace.id=cycle.workspace_id
JOIN LATERAL (` + archivedCycleIssueCountsSQL + `) raw_counts(total,completed,cancelled,started,unstarted,backlog) ON true
CROSS JOIN LATERAL (SELECT raw_counts.*) counts
WHERE workspace.slug=$2 AND project.id=$3 AND cycle.id=$4
  AND cycle.deleted_at IS NULL AND cycle.archived_at IS NOT NULL
  AND project.deleted_at IS NULL AND project.archived_at IS NULL`

var activeCycleArchiveDetailSQL = strings.Replace(
	strings.Replace(
		strings.Replace(
			strings.Replace(
				archivedCycleDetailSQL,
				"cycle.archived_at IS NOT NULL",
				"cycle.archived_at IS NULL",
				1,
			),
			archivedCycleIssueCountsSQL,
			cycleIssueCountsSQL,
			1,
		),
		archivedIssueDistributionSQL,
		issueDistributionSQL,
		1,
	),
	"WHERE ci.cycle_id=cycle.id),'[]'::json)",
	"WHERE ci.cycle_id=cycle.id AND assignee.deleted_at IS NULL),'[]'::json)",
	1,
)

var cycleAnalyticsSQL = `
SELECT cycle.start_date IS NOT NULL AND cycle.end_date IS NOT NULL,
  CASE WHEN cycle.progress_snapshot <> '{}'::jsonb THEN json_build_object(
    'labels',COALESCE(cycle.progress_snapshot->'distribution'->'labels','[]'::jsonb),
    'assignees',COALESCE(cycle.progress_snapshot->'distribution'->'assignees','[]'::jsonb),
    'completion_chart',COALESCE(cycle.progress_snapshot->'distribution'->'completion_chart','{}'::jsonb)
  )
  WHEN $5='points' AND EXISTS(SELECT 1 FROM estimates estimate WHERE estimate.id=project.estimate_id AND estimate.type='points')
    THEN (` + estimateDistributionSQL + `)
  WHEN $5='issues' THEN (` + analyticsIssueDistributionSQL + `)
  ELSE json_build_object('labels','[]'::json,'assignees','[]'::json,'completion_chart','{}'::json) END
FROM cycles cycle
JOIN projects project ON project.id=cycle.project_id
JOIN workspaces workspace ON workspace.id=cycle.workspace_id
WHERE workspace.slug=$2 AND project.id=$3 AND cycle.id=$4
  AND cycle.deleted_at IS NULL AND project.deleted_at IS NULL`

const groupedIssuePayloadSQL = `
SELECT COALESCE(jsonb_object_agg(
  bucket.group_key,
  jsonb_build_object('results',bucket.results,'total_results',bucket.total_results)
),'{}'::jsonb)
FROM buckets bucket`

const multiGroupedIssuePayloadSQL = `
SELECT jsonb_object_agg(bucket.group_key,jsonb_build_object(
  'total_results',bucket.total_results,
  'results',selected.results
))
FROM buckets bucket
CROSS JOIN LATERAL (
  SELECT COALESCE(jsonb_agg(deduplicated.item ORDER BY deduplicated.position),'[]'::jsonb) results
  FROM (
    SELECT DISTINCT ON (processed.issue_id) processed.item,processed.position
    FROM processed
    WHERE processed.group_key=bucket.group_key
    ORDER BY processed.issue_id,processed.position
  ) deduplicated
) selected`

const subgroupedIssuePayloadSQL = `
SELECT COALESCE(jsonb_object_agg(
  group_value.value,
  jsonb_build_object(
    'results',COALESCE((
      SELECT jsonb_object_agg(
        bucket.subgroup_key,
        jsonb_build_object('results',bucket.results,'total_results',bucket.total_results)
      )
      FROM buckets bucket
      WHERE bucket.group_key=group_value.value
    ),'{}'::jsonb),
    'total_results',COALESCE((
      SELECT group_total.total_results
      FROM group_totals group_total
      WHERE group_total.group_key=group_value.value
    ),0)
  )
),'{}'::jsonb)
FROM universes
CROSS JOIN LATERAL jsonb_array_elements_text(universes.group_values) group_value(value)`

const issueRequestScopeSQL = `
WITH request_scope AS (
  SELECT $1::uuid user_id, $2::text slug, $3::uuid project_id, $4::uuid cycle_id
)`

const issueFilterFromSQL = `  FROM issues issue
  CROSS JOIN request_scope
  JOIN cycle_issues cycle_issue ON cycle_issue.issue_id=issue.id
  %s
  JOIN projects project ON project.id=issue.project_id
  JOIN workspaces workspace ON workspace.id=issue.workspace_id
  LEFT JOIN states state ON state.id=issue.state_id
`

var groupedIssueListSQL = issueRequestScopeSQL + `, filtered AS (
  SELECT issue.*,(SELECT annotated_cycle.cycle_id FROM cycle_issues annotated_cycle JOIN cycles live_cycle ON live_cycle.id=annotated_cycle.cycle_id AND live_cycle.deleted_at IS NULL AND live_cycle.project_id=issue.project_id AND live_cycle.workspace_id=issue.workspace_id WHERE annotated_cycle.issue_id=issue.id AND annotated_cycle.deleted_at IS NULL ORDER BY annotated_cycle.created_at DESC LIMIT 1) cycle_id
` + issueFilterFromSQL + `
  WHERE %s
), params AS (
  SELECT $%d::numeric lower_bound,$%d::numeric upper_bound
), matching AS (
  SELECT issue.id issue_id,
    (NOT EXISTS(
      SELECT 1 FROM intake_issues count_intake
      WHERE count_intake.issue_id=issue.id
    ) OR EXISTS(
      SELECT 1 FROM intake_issues count_intake
      WHERE count_intake.issue_id=issue.id
        AND count_intake.status IN (-1,1,2)
    )) countable,
    row_number() OVER (ORDER BY %s) position,
    jsonb_build_object(
      'id',issue.id,'name',issue.name,'type_id',issue.type_id,'state_id',issue.state_id,'sort_order',issue.sort_order,
      'completed_at',` + utcTimestampSQL("issue.completed_at") + `,'estimate_point',issue.estimate_point_id,'priority',issue.priority,
      'start_date',issue.start_date,'target_date',issue.target_date,'sequence_id',issue.sequence_id,
      'project_id',issue.project_id,'parent_id',` + djangoapi.VisibleParentSQL("issue", "$1::uuid") + `,'cycle_id',issue.cycle_id,
      'sub_issues_count',(SELECT count(*) FROM issues child
        LEFT JOIN states child_state ON child_state.id=child.state_id
        JOIN projects child_project ON child_project.id=child.project_id
        WHERE child.parent_id=issue.id AND ` + djangoapi.IssueVisibleSQL("child", "$1::uuid", "issue.workspace_id") + ` AND child.deleted_at IS NULL AND child.archived_at IS NULL
          AND child.is_draft=false AND child_state."group" IS DISTINCT FROM 'triage' AND child_project.archived_at IS NULL),
      'created_at',` + utcTimestampSQL("issue.created_at") + `,'updated_at',` + utcTimestampSQL("issue.updated_at") + `,
      'created_by',issue.created_by_id,'updated_by',issue.updated_by_id,
      'attachment_count',(SELECT count(*) FROM file_assets asset WHERE asset.issue_id=issue.id AND asset.entity_type='ISSUE_ATTACHMENT' AND asset.deleted_at IS NULL),
      'link_count',(SELECT count(*) FROM issue_links link WHERE link.issue_id=issue.id AND link.deleted_at IS NULL),
      'is_draft',issue.is_draft,'archived_at',issue.archived_at,'state__group',state."group",
      'assignee_ids',COALESCE((SELECT jsonb_agg(DISTINCT assignee.assignee_id) FROM issue_assignees assignee WHERE assignee.issue_id=issue.id AND assignee.deleted_at IS NULL),'[]'::jsonb),
      'label_ids',COALESCE((SELECT jsonb_agg(DISTINCT issue_label.label_id) FROM issue_labels issue_label WHERE ` + issueRelationPredicate("issue_labels", "issue_label", true) + `),'[]'::jsonb),
      'module_ids',COALESCE((SELECT jsonb_agg(DISTINCT module_issue.module_id) FROM module_issues module_issue JOIN modules projected_module ON projected_module.id=module_issue.module_id WHERE module_issue.issue_id=issue.id AND module_issue.deleted_at IS NULL AND projected_module.deleted_at IS NULL AND projected_module.archived_at IS NULL),'[]'::jsonb)
    ) item
  FROM filtered issue
  LEFT JOIN states state ON state.id=issue.state_id
), expanded AS (
  SELECT matching.issue_id,matching.countable,matching.item,matching.position,group_value.value group_key,subgroup_value.value subgroup_key
  FROM matching
  CROSS JOIN LATERAL jsonb_array_elements_text(%s) group_value(value)
  CROSS JOIN LATERAL jsonb_array_elements_text(%s) subgroup_value(value)
), ranked AS (
  SELECT expanded.*,
    row_number() OVER (PARTITION BY group_key,subgroup_key ORDER BY position) bucket_position
  FROM expanded
), paged AS (
  SELECT ranked.*
  FROM ranked CROSS JOIN params
  WHERE ranked.bucket_position>params.lower_bound
    AND ranked.bucket_position<params.upper_bound
), processed AS (
  SELECT paged.issue_id,paged.position,paged.group_key,paged.subgroup_key,%s item
  FROM paged
), universes AS (
  SELECT COALESCE(jsonb_agg(group_key ORDER BY group_key),'[]'::jsonb) group_values
  FROM (
    SELECT jsonb_array_elements_text((%s)::jsonb) group_key
    UNION
    SELECT group_key FROM expanded
  ) visible_groups
), bucket_keys AS (
  %s
), buckets AS (
  SELECT bucket_key.group_key,bucket_key.subgroup_key,
    COALESCE((
      SELECT jsonb_agg(processed.item ORDER BY processed.position)
      FROM processed
      WHERE processed.group_key=bucket_key.group_key AND processed.subgroup_key=bucket_key.subgroup_key
    ),'[]'::jsonb) results,
    %s total_results
  FROM bucket_keys bucket_key
), group_totals AS (
  SELECT expanded.group_key,
    count(DISTINCT expanded.issue_id) FILTER (WHERE expanded.countable)::int countable_results,
    GREATEST(count(DISTINCT expanded.issue_id) FILTER (WHERE expanded.countable)::int,1) total_results
  FROM expanded
  GROUP BY expanded.group_key
)
SELECT CASE WHEN EXISTS(SELECT 1 FROM paged) THEN (` + `%s` + `) ELSE '{}'::jsonb END,
  (SELECT count(*)::int FROM expanded),
  (SELECT count(*)::int FROM paged),
  CASE WHEN EXISTS(SELECT 1 FROM paged)
    THEN COALESCE(%s,0)
    ELSE 0 END,
  COALESCE((
    SELECT EXISTS(
      SELECT 1 FROM ranked CROSS JOIN params
      WHERE ranked.bucket_position>=params.upper_bound
    )
  ),false)`

var issueListSQL = issueRequestScopeSQL + `, base AS (
  SELECT issue.*,(SELECT annotated_cycle.cycle_id FROM cycle_issues annotated_cycle JOIN cycles live_cycle ON live_cycle.id=annotated_cycle.cycle_id AND live_cycle.deleted_at IS NULL AND live_cycle.project_id=issue.project_id AND live_cycle.workspace_id=issue.workspace_id WHERE annotated_cycle.issue_id=issue.id AND annotated_cycle.deleted_at IS NULL ORDER BY annotated_cycle.created_at DESC LIMIT 1) cycle_id
` + issueFilterFromSQL + `
  WHERE %s
), base_total AS (
  SELECT count(DISTINCT id)::int total FROM base
), filtered AS (
  SELECT issue.*%s
  FROM base issue
  %s
  WHERE %s
), matching AS (
  SELECT issue.id, issue.created_at, row_number() OVER (ORDER BY %s) position, json_build_object(
    'id',issue.id,'name',issue.name,'type_id',issue.type_id,'state_id',issue.state_id,'sort_order',issue.sort_order,
    'completed_at',` + utcTimestampSQL("issue.completed_at") + `,'estimate_point',issue.estimate_point_id,'priority',issue.priority,
    'start_date',issue.start_date,'target_date',issue.target_date,'sequence_id',issue.sequence_id,
    'project_id',issue.project_id,'parent_id',` + djangoapi.VisibleParentSQL("issue", "$1::uuid") + `,'cycle_id',issue.cycle_id,
    'sub_issues_count',(SELECT count(*) FROM issues child
      LEFT JOIN states child_state ON child_state.id=child.state_id
      JOIN projects child_project ON child_project.id=child.project_id
      WHERE child.parent_id=issue.id AND ` + djangoapi.IssueVisibleSQL("child", "$1::uuid", "issue.workspace_id") + ` AND child.deleted_at IS NULL AND child.archived_at IS NULL
        AND child.is_draft=false AND child_state."group" IS DISTINCT FROM 'triage' AND child_project.archived_at IS NULL),
    'created_at',` + utcTimestampSQL("issue.created_at") + `,'updated_at',` + utcTimestampSQL("issue.updated_at") + `,'created_by',issue.created_by_id,'updated_by',issue.updated_by_id,
    'attachment_count',(SELECT count(*) FROM file_assets asset WHERE asset.issue_id=issue.id AND asset.entity_type='ISSUE_ATTACHMENT' AND asset.deleted_at IS NULL),
    'link_count',(SELECT count(*) FROM issue_links link WHERE link.issue_id=issue.id AND link.deleted_at IS NULL),
    'is_draft',issue.is_draft,'archived_at',issue.archived_at,'state__group',state."group",
    %s
  ) item
  FROM filtered issue
  LEFT JOIN states state ON state.id=issue.state_id
  ORDER BY %s
  LIMIT $%d OFFSET $%d
)
SELECT COALESCE(json_agg(item ORDER BY position),'[]'::json),
  (SELECT total FROM base_total), %s, %s,
  (SELECT count(*) FROM filtered)
FROM matching`

var cycleIssueDetailSQL = `
SELECT json_build_object(
  'id',cycle_issue.id,'workspace',cycle_issue.workspace_id,'project',cycle_issue.project_id,
  'cycle',cycle_issue.cycle_id,'issue',cycle_issue.issue_id,
  'created_at',` + userTimestampSQL("cycle_issue.created_at") + `,
  'updated_at',` + userTimestampSQL("cycle_issue.updated_at") + `,
  'created_by',cycle_issue.created_by_id,'updated_by',cycle_issue.updated_by_id,
  'deleted_at',` + userTimestampSQL("cycle_issue.deleted_at") + `,
  'sub_issues_count',(SELECT count(*) FROM issues child
    LEFT JOIN states child_state ON child_state.id=child.state_id
    JOIN projects child_project ON child_project.id=child.project_id
    WHERE child.parent_id=issue.id AND ` + djangoapi.IssueVisibleSQL("child", "$1::uuid", "issue.workspace_id") + ` AND child.deleted_at IS NULL AND child.archived_at IS NULL
      AND child.is_draft=false AND child_state."group" IS DISTINCT FROM 'triage' AND child_project.archived_at IS NULL),
  'issue_detail',(to_jsonb(issue)
    - 'workspace_id' - 'project_id' - 'state_id' - 'parent_id' - 'estimate_point_id' - 'type_id'
    - 'created_by_id' - 'updated_by_id'
    || jsonb_build_object(
      'workspace',issue.workspace_id,'project',issue.project_id,'state',issue.state_id,'parent',` + djangoapi.VisibleParentSQL("issue", "$1::uuid") + `,
      'estimate_point',issue.estimate_point_id,'type',issue.type_id,
      'created_by',issue.created_by_id,'updated_by',issue.updated_by_id,
      'description_binary',CASE WHEN issue.description_binary IS NULL THEN NULL
        ELSE replace(encode(issue.description_binary,'base64'),E'\n','') END,
      'created_at',` + userTimestampSQL("issue.created_at") + `,
      'updated_at',` + userTimestampSQL("issue.updated_at") + `,
      'deleted_at',` + userTimestampSQL("issue.deleted_at") + `,
      'completed_at',` + userTimestampSQL("issue.completed_at") + `,
      'archived_at',issue.archived_at,
      'state_detail',CASE WHEN state.id IS NULL THEN NULL ELSE jsonb_build_object(
        'id',state.id,'name',state.name,'color',state.color,'group',state."group"
      ) END,
      'project_detail',jsonb_build_object(
        'id',project.id,'identifier',project.identifier,'name',project.name,'cover_image',project.cover_image,
        'cover_image_url',CASE WHEN project.cover_image_asset_id IS NOT NULL
          THEN '/api/assets/v2/static/'||project.cover_image_asset_id::text||'/'
          ELSE NULLIF(project.cover_image,'') END,
        'logo_props',project.logo_props,'description',project.description
      ),
      'assignee_details',COALESCE((
        SELECT jsonb_agg(jsonb_build_object(
          'id',user_account.id,'first_name',user_account.first_name,'last_name',user_account.last_name,
          'avatar',user_account.avatar,
          'avatar_url',CASE WHEN user_account.avatar_asset_id IS NOT NULL
            THEN '/api/assets/v2/static/'||user_account.avatar_asset_id::text||'/'
            ELSE NULLIF(user_account.avatar,'') END,
          'is_bot',user_account.is_bot,'display_name',user_account.display_name
        ) ORDER BY user_account.created_at DESC)
        FROM issue_assignees assignee
        JOIN users user_account ON user_account.id=assignee.assignee_id
        WHERE assignee.issue_id=issue.id
      ),'[]'::jsonb),
      'label_details',COALESCE((
        SELECT jsonb_agg(jsonb_build_object('id',label.id,'name',label.name,'color',label.color) ORDER BY label.created_at DESC)
        FROM issue_labels issue_label
        JOIN labels label ON label.id=issue_label.label_id
        WHERE ` + issueRelationPredicate("issue_labels", "issue_label", true) + `
      ),'[]'::jsonb)
    )
  )
)
FROM cycle_issues cycle_issue
JOIN issues issue ON issue.id=cycle_issue.issue_id
  AND issue.project_id=cycle_issue.project_id AND issue.workspace_id=cycle_issue.workspace_id
JOIN projects project ON project.id=cycle_issue.project_id
JOIN workspaces workspace ON workspace.id=cycle_issue.workspace_id
LEFT JOIN states state ON state.id=issue.state_id
WHERE workspace.slug=$2 AND project.id=$3 AND cycle_issue.cycle_id=$4 AND cycle_issue.issue_id=$5
  AND cycle_issue.deleted_at IS NULL
  AND project.deleted_at IS NULL AND project.archived_at IS NULL
  AND EXISTS (
    SELECT 1 FROM project_members detail_member
    WHERE detail_member.project_id=project.id AND detail_member.workspace_id=workspace.id
      AND detail_member.member_id=$1 AND detail_member.is_active=true AND detail_member.deleted_at IS NULL
  )`

var cycleProgressSQL = `
SELECT json_build_object(
  'backlog_estimate_points',COALESCE(sum(point.value::float8) FILTER(WHERE state."group"='backlog'),0),
  'unstarted_estimate_points',COALESCE(sum(point.value::float8) FILTER(WHERE state."group"='unstarted'),0),
  'started_estimate_points',COALESCE(sum(point.value::float8) FILTER(WHERE state."group"='started'),0),
  'cancelled_estimate_points',COALESCE(sum(point.value::float8) FILTER(WHERE state."group"='cancelled'),0),
  'completed_estimate_points',COALESCE(sum(point.value::float8) FILTER(WHERE state."group"='completed'),0),
  'total_estimate_points',COALESCE(sum(point.value::float8) FILTER(WHERE state."group" IS DISTINCT FROM 'triage'),0),
  'backlog_issues',CASE WHEN cycle.progress_snapshot <> '{}'::jsonb THEN COALESCE((cycle.progress_snapshot->>'backlog_issues')::int,0) ELSE count(issue.id) FILTER(WHERE state."group"='backlog') END,
  'unstarted_issues',CASE WHEN cycle.progress_snapshot <> '{}'::jsonb THEN COALESCE((cycle.progress_snapshot->>'unstarted_issues')::int,0) ELSE count(issue.id) FILTER(WHERE state."group"='unstarted') END,
  'started_issues',CASE WHEN cycle.progress_snapshot <> '{}'::jsonb THEN COALESCE((cycle.progress_snapshot->>'started_issues')::int,0) ELSE count(issue.id) FILTER(WHERE state."group"='started') END,
  'cancelled_issues',CASE WHEN cycle.progress_snapshot <> '{}'::jsonb THEN COALESCE((cycle.progress_snapshot->>'cancelled_issues')::int,0) ELSE count(issue.id) FILTER(WHERE state."group"='cancelled') END,
  'completed_issues',CASE WHEN cycle.progress_snapshot <> '{}'::jsonb THEN COALESCE((cycle.progress_snapshot->>'completed_issues')::int,0) ELSE count(issue.id) FILTER(WHERE state."group"='completed') END,
  'total_issues',CASE WHEN cycle.progress_snapshot <> '{}'::jsonb THEN COALESCE((cycle.progress_snapshot->>'total_issues')::int,0) ELSE count(issue.id) FILTER(WHERE state."group" IS DISTINCT FROM 'triage') END
)
FROM cycles cycle
JOIN projects project ON project.id=cycle.project_id
JOIN workspaces workspace ON workspace.id=cycle.workspace_id
LEFT JOIN cycle_issues cycle_issue ON cycle_issue.cycle_id=cycle.id AND cycle_issue.deleted_at IS NULL
LEFT JOIN issues issue ON issue.id=cycle_issue.issue_id AND issue.project_id=project.id AND issue.workspace_id=workspace.id
  AND project.archived_at IS NULL
  AND issue.deleted_at IS NULL
  AND issue.archived_at IS NULL AND issue.is_draft=false
LEFT JOIN states state ON state.id=issue.state_id
LEFT JOIN estimate_points point ON ` + issuequery.LiveNumericEstimatePredicate("issue", "point") + `
WHERE workspace.slug=$1 AND project.id=$2 AND cycle.id=$3 AND cycle.deleted_at IS NULL
GROUP BY cycle.id`

var projectCycleDetailSQL = `
SELECT json_build_object(
  'id',cycle.id,'workspace_id',cycle.workspace_id,'project_id',cycle.project_id,'name',cycle.name,
  'description',cycle.description,'start_date',` + projectTimestampSQL("cycle.start_date") + `,
  'end_date',` + projectTimestampSQL("cycle.end_date") + `,'owned_by_id',cycle.owned_by_id,
  'view_props',cycle.view_props,'sort_order',cycle.sort_order,'external_source',cycle.external_source,
  'external_id',cycle.external_id,'progress_snapshot',cycle.progress_snapshot,
  'sub_issues',(SELECT count(*) FROM cycle_issues ci JOIN issues child ON child.id=ci.issue_id
    LEFT JOIN states child_state ON child_state.id=child.state_id
    WHERE ci.cycle_id=cycle.id AND ci.deleted_at IS NULL AND child.deleted_at IS NULL
      AND child.project_id=project.id AND child.parent_id IS NOT NULL AND child.archived_at IS NULL
      AND child.is_draft=false AND child_state."group" IS DISTINCT FROM 'triage'),
  'logo_props',cycle.logo_props,'version',cycle.version,
  'is_favorite',EXISTS(SELECT 1 FROM user_favorites favorite WHERE favorite.user_id=$1 AND favorite.entity_type='cycle'
    AND favorite.entity_identifier=cycle.id AND favorite.project_id=project.id AND favorite.workspace_id=workspace.id AND favorite.deleted_at IS NULL),
  'total_issues',counts.total,'completed_issues',counts.completed,
  'assignee_ids',COALESCE((SELECT json_agg(DISTINCT assignee.assignee_id) FROM cycle_issues ci JOIN issue_assignees assignee ON assignee.issue_id=ci.issue_id
    WHERE ci.cycle_id=cycle.id AND assignee.deleted_at IS NULL),'[]'::json),
  'status',` + cycleStatusSQL + `,'created_by',cycle.created_by_id
)
FROM cycles cycle
JOIN projects project ON project.id=cycle.project_id
JOIN workspaces workspace ON workspace.id=cycle.workspace_id
JOIN LATERAL (` + cycleIssueCountsSQL + `) raw_counts(total,completed,cancelled,started,unstarted,backlog) ON true
CROSS JOIN LATERAL (SELECT raw_counts.total,raw_counts.completed) counts
WHERE workspace.slug=$2 AND project.id=$3 AND cycle.id=$4
  AND cycle.deleted_at IS NULL AND cycle.archived_at IS NULL
  AND project.deleted_at IS NULL AND project.archived_at IS NULL`
