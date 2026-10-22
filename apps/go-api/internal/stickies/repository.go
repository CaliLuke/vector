package stickies

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// ErrStickyNotFound intentionally has the same public meaning as a missing
// workspace membership. Callers must translate both errors to M1-001's 404.
var ErrStickyNotFound = ErrWorkspaceAccessDenied

// ErrStickyWorkspaceDeleted identifies a workspace that is still visible to
// API v1 permission checks but is unavailable to Sticky view operations.
var ErrStickyWorkspaceDeleted = errors.New("Sticky workspace is deleted")

// ErrInvalidPagination identifies a cursor page which cannot be queried.
var ErrInvalidPagination = errors.New("invalid Sticky pagination")

// ErrStickyForbidden identifies an owned Sticky which the user did not create.
var ErrStickyForbidden = errors.New("Sticky creator permission denied")

// Sticky is the database representation used by the Sticky endpoints.
type Sticky struct {
	ID                  StickyID
	CreatedAt           time.Time
	UpdatedAt           time.Time
	DeletedAt           *time.Time
	Name                *string
	Description         []byte
	DescriptionHTML     string
	DescriptionStripped *string
	DescriptionBinary   []byte
	LogoProps           []byte
	Color               *string
	BackgroundColor     *string
	CreatedByID         *UserID
	OwnerID             UserID
	UpdatedByID         *UserID
	WorkspaceID         WorkspaceID
	SortOrder           float64
}

// ListOptions controls the Django-compatible, descending-created-at listing.
type ListOptions struct {
	Query          string
	Cursor         *StickyCursor
	Limit          int
	CreatedAtOrder bool
	LiteralQuery   bool
}

// StickyCursor is Django's value:page:is_previous offset cursor.
type StickyCursor struct {
	Value      float64
	Page       int
	IsPrevious bool
}

// StickyPage is one page of owner-scoped stickies.
type StickyPage struct {
	Stickies    []Sticky
	TotalCount  int
	Page        int
	Limit       int
	HasNext     bool
	HasPrevious bool
}

// CreateStickyInput contains caller-validated content. Sanitization and HTML
// stripping deliberately remain in the content/handler boundary.
type CreateStickyInput struct {
	Name                *string
	Description         []byte
	DescriptionHTML     *string
	DescriptionStripped *string
	DescriptionBinary   []byte
	LogoProps           []byte
	Color               *string
	BackgroundColor     *string
	DeletedAt           *time.Time
	SortOrder           *float64
}

// PatchStickyInput uses nil to mean "leave this column unchanged".
type PatchStickyInput struct {
	Name                **string
	Description         *[]byte
	DescriptionHTML     *string
	DescriptionStripped **string
	DescriptionBinary   *[]byte
	LogoProps           *[]byte
	Color               **string
	BackgroundColor     **string
	CreatedByID         **UserID
	DeletedAt           **time.Time
	SortOrder           *float64
}

type stickyDatabase interface {
	transactionStarter
	QueryRow(context.Context, string, ...any) pgx.Row
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

// Repository is the PostgreSQL implementation of Sticky persistence.
type Repository struct{ database stickyDatabase }

func NewRepository(database stickyDatabase) (*Repository, error) {
	if database == nil {
		return nil, fmt.Errorf("Sticky repository database is required")
	}
	return &Repository{database: database}, nil
}

// ResolveWorkspace finds only an undeleted workspace to which the user has an
// active, undeleted membership. Absence is deliberately indistinguishable.
func (repository *Repository) ResolveWorkspace(ctx context.Context, slug string, userID UserID) (WorkspaceID, error) {
	if slug == "" || uuid.UUID(userID) == uuid.Nil() {
		return WorkspaceID{}, ErrInvalidScope
	}
	var id uuid.UUID
	err := repository.database.QueryRow(ctx, `
SELECT w.id FROM workspaces w
WHERE w.slug = $1 AND w.deleted_at IS NULL
  AND EXISTS (SELECT 1 FROM workspace_members wm WHERE wm.workspace_id = w.id
    AND wm.member_id = $2 AND wm.is_active IS TRUE AND wm.deleted_at IS NULL)
`, slug, uuid.UUID(userID)).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return WorkspaceID{}, ErrWorkspaceAccessDenied
	}
	if err != nil {
		return WorkspaceID{}, fmt.Errorf("resolve Sticky workspace: %w", err)
	}
	return WorkspaceID(id), nil
}

// ResolveAPIWorkspace preserves API v1's permission-before-view lookup order:
// nonmembers are denied, while a deleted workspace is a missing view object.
func (repository *Repository) ResolveAPIWorkspace(ctx context.Context, slug string, userID UserID) (WorkspaceID, error) {
	if slug == "" || uuid.UUID(userID) == uuid.Nil() {
		return WorkspaceID{}, ErrInvalidScope
	}
	var id uuid.UUID
	var deleted bool
	var member bool
	err := repository.database.QueryRow(ctx, `
SELECT w.id, w.deleted_at IS NOT NULL,
  EXISTS (SELECT 1 FROM workspace_members wm WHERE wm.workspace_id = w.id
    AND wm.member_id = $2 AND wm.is_active IS TRUE AND wm.deleted_at IS NULL)
FROM workspaces w WHERE w.slug = $1
`, slug, uuid.UUID(userID)).Scan(&id, &deleted, &member)
	if errors.Is(err, pgx.ErrNoRows) {
		return WorkspaceID{}, ErrWorkspaceAccessDenied
	}
	if err != nil {
		return WorkspaceID{}, err
	}
	if !member {
		return WorkspaceID{}, ErrWorkspaceAccessDenied
	}
	if deleted {
		return WorkspaceID(id), ErrStickyWorkspaceDeleted
	}
	return WorkspaceID(id), nil
}

// UserExists reports whether a user primary key can be resolved by a DRF
// PrimaryKeyRelatedField before the model save hook overwrites audit fields.
func (repository *Repository) UserExists(ctx context.Context, userID UserID) (bool, error) {
	var exists bool
	err := repository.database.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id=$1)`, uuid.UUID(userID)).Scan(&exists)
	return exists, err
}

func (repository *Repository) List(ctx context.Context, scope Scope, options ListOptions) (StickyPage, error) {
	if !validOwnerScope(scope, OperationList, false) {
		return StickyPage{}, ErrInvalidScope
	}
	limit := options.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 1000 {
		limit = 1000
	}
	page := 0
	if options.Cursor != nil {
		page = options.Cursor.Page
	}
	if page < 0 {
		return StickyPage{}, ErrInvalidPagination
	}
	args := []any{uuid.UUID(scope.WorkspaceID), uuid.UUID(scope.UserID)}
	where := ` FROM stickies s WHERE s.workspace_id = $1 AND s.owner_id = $2 AND s.deleted_at IS NULL
AND EXISTS (SELECT 1 FROM workspace_members wm WHERE wm.workspace_id = s.workspace_id AND wm.member_id = $2 AND wm.is_active IS TRUE AND wm.deleted_at IS NULL)`
	if options.Query != "" {
		args = append(args, options.Query)
		if options.LiteralQuery {
			args[len(args)-1] = escapeStickySearch(options.Query)
		}
		where += fmt.Sprintf(" AND s.description_stripped ILIKE '%%' || $%d || '%%'", len(args))
	}
	var total int
	if err := repository.database.QueryRow(ctx, "SELECT COUNT(*)"+where, args...).Scan(&total); err != nil {
		return StickyPage{}, fmt.Errorf("count Stickies: %w", err)
	}
	args = append(args, limit+1, page*limit)
	order := "s.sort_order DESC"
	if options.CreatedAtOrder {
		order = "s.created_at DESC"
	}
	query := "SELECT " + stickyColumns + where + fmt.Sprintf(" ORDER BY %s LIMIT $%d OFFSET $%d", order, len(args)-1, len(args))
	rows, err := repository.database.Query(ctx, query, args...)
	if err != nil {
		return StickyPage{}, fmt.Errorf("list Stickies: %w", err)
	}
	defer rows.Close()
	result := StickyPage{Stickies: make([]Sticky, 0, limit), TotalCount: total, Page: page, Limit: limit, HasPrevious: page > 0}
	for rows.Next() {
		var sticky Sticky
		if err := scanSticky(rows, &sticky); err != nil {
			return StickyPage{}, err
		}
		result.Stickies = append(result.Stickies, sticky)
	}
	if err := rows.Err(); err != nil {
		return StickyPage{}, fmt.Errorf("read Stickies: %w", err)
	}
	if len(result.Stickies) > limit {
		result.Stickies = result.Stickies[:limit]
		result.HasNext = true
	}
	return result, nil
}

func escapeStickySearch(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `%`, `\%`)
	return strings.ReplaceAll(value, `_`, `\_`)
}

func (repository *Repository) Retrieve(ctx context.Context, stickyID StickyID, scope Scope) (Sticky, error) {
	if uuid.UUID(stickyID) == uuid.Nil() || !validOwnerScope(scope, OperationRetrieve, false) {
		return Sticky{}, ErrInvalidScope
	}
	return repository.one(ctx, stickyID, scope, false)
}

// Create locks its workspace before calculating sort_order, matching
// CreateWithAllocatedSort and preventing concurrent duplicate allocations.
func (repository *Repository) Create(ctx context.Context, stickyID StickyID, scope Scope, input CreateStickyInput) (Sticky, error) {
	if uuid.UUID(stickyID) == uuid.Nil() || !validCreateScope(scope) {
		return Sticky{}, ErrInvalidScope
	}
	tx, err := repository.database.Begin(ctx)
	if err != nil {
		return Sticky{}, fmt.Errorf("%w: begin transaction: %w", ErrStickyCreateFailed, err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	var workspace uuid.UUID
	err = tx.QueryRow(ctx, `SELECT id FROM workspaces WHERE id=$1 AND deleted_at IS NULL FOR UPDATE`, uuid.UUID(scope.WorkspaceID)).Scan(&workspace)
	if errors.Is(err, pgx.ErrNoRows) {
		return Sticky{}, ErrWorkspaceAccessDenied
	}
	if err != nil {
		return Sticky{}, fmt.Errorf("%w: lock workspace: %w", ErrStickyCreateFailed, err)
	}
	var active bool
	if err = tx.QueryRow(ctx, activeMembershipQuery, workspace, uuid.UUID(scope.UserID)).Scan(&active); err != nil {
		return Sticky{}, fmt.Errorf("%w: read workspace membership: %w", ErrStickyCreateFailed, err)
	}
	if !active {
		return Sticky{}, ErrWorkspaceAccessDenied
	}
	var largest pgtype.Float8
	if err = tx.QueryRow(ctx, `SELECT MAX(sort_order) FROM stickies WHERE workspace_id=$1 AND deleted_at IS NULL`, workspace).Scan(&largest); err != nil {
		return Sticky{}, fmt.Errorf("%w: read largest sort order: %w", ErrStickyCreateFailed, err)
	}
	description := input.Description
	if description == nil {
		description = []byte("{}")
	}
	html := "<p></p>"
	if input.DescriptionHTML != nil {
		html = *input.DescriptionHTML
	}
	stripped := input.DescriptionStripped
	logo := input.LogoProps
	if logo == nil {
		logo = []byte("{}")
	}
	order := float64(initialSortOrder)
	if largest.Valid {
		order = nextSortOrder(largest)
	} else if input.SortOrder != nil {
		order = *input.SortOrder
	}
	var inserted Sticky
	err = scanSticky(tx.QueryRow(ctx, `INSERT INTO stickies (created_at,updated_at,deleted_at,id,name,description,description_html,description_stripped,description_binary,logo_props,color,background_color,created_by_id,owner_id,updated_by_id,workspace_id,sort_order) VALUES (CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$11,NULL,$12,$13) RETURNING `+stickyColumns, input.DeletedAt, uuid.UUID(stickyID), input.Name, description, html, stripped, input.DescriptionBinary, logo, input.Color, input.BackgroundColor, uuid.UUID(scope.UserID), workspace, order), &inserted)
	if err != nil {
		return Sticky{}, fmt.Errorf("%w: insert Sticky: %w", ErrStickyCreateFailed, err)
	}
	if err = tx.Commit(ctx); err != nil {
		return Sticky{}, fmt.Errorf("%w: commit transaction: %w", ErrStickyCreateFailed, err)
	}
	return inserted, nil
}

func (repository *Repository) Patch(ctx context.Context, stickyID StickyID, scope Scope, input PatchStickyInput) (Sticky, error) {
	return repository.patch(ctx, stickyID, scope, input, true)
}

// PatchOwned applies an API v1 Sticky update scoped by workspace and owner.
func (repository *Repository) PatchOwned(ctx context.Context, stickyID StickyID, scope Scope, input PatchStickyInput) (Sticky, error) {
	return repository.patch(ctx, stickyID, scope, input, false)
}

func (repository *Repository) patch(ctx context.Context, stickyID StickyID, scope Scope, input PatchStickyInput, requireCreator bool) (Sticky, error) {
	if uuid.UUID(stickyID) == uuid.Nil() || !validOwnerScope(scope, OperationUpdate, requireCreator) {
		return Sticky{}, ErrInvalidScope
	}
	sets, args := []string{"updated_at = CURRENT_TIMESTAMP", "updated_by_id = $1"}, []any{uuid.UUID(scope.UserID)}
	add := func(column string, value any) {
		args = append(args, value)
		sets = append(sets, fmt.Sprintf("%s = $%d", column, len(args)))
	}
	if input.Name != nil {
		add("name", *input.Name)
	}
	if input.Description != nil {
		add("description", *input.Description)
	}
	if input.DescriptionHTML != nil {
		add("description_html", *input.DescriptionHTML)
	}
	if input.DescriptionStripped != nil {
		add("description_stripped", *input.DescriptionStripped)
	}
	if input.DescriptionBinary != nil {
		add("description_binary", *input.DescriptionBinary)
	}
	if input.LogoProps != nil {
		add("logo_props", *input.LogoProps)
	}
	if input.Color != nil {
		add("color", *input.Color)
	}
	if input.BackgroundColor != nil {
		add("background_color", *input.BackgroundColor)
	}
	if input.CreatedByID != nil {
		add("created_by_id", *input.CreatedByID)
	}
	if input.DeletedAt != nil {
		add("deleted_at", *input.DeletedAt)
	}
	if input.SortOrder != nil {
		add("sort_order", *input.SortOrder)
	}
	idPosition := len(args) + 1
	workspacePosition := idPosition + 1
	userPosition := workspacePosition + 1
	args = append(args, uuid.UUID(stickyID), uuid.UUID(scope.WorkspaceID), uuid.UUID(scope.UserID))
	query := "UPDATE stickies SET " + join(sets, ", ") + fmt.Sprintf(" WHERE id=$%d AND workspace_id=$%d AND owner_id=$%d", idPosition, workspacePosition, userPosition)
	if requireCreator {
		query += fmt.Sprintf(" AND created_by_id=$%d", userPosition)
	}
	query += " AND deleted_at IS NULL RETURNING " + stickyColumns
	var sticky Sticky
	err := scanSticky(repository.database.QueryRow(ctx, query, args...), &sticky)
	if errors.Is(err, pgx.ErrNoRows) {
		if requireCreator {
			if _, ownerErr := repository.one(ctx, stickyID, Scope{Operation: OperationRetrieve, WorkspaceID: scope.WorkspaceID, UserID: scope.UserID, Ownership: RequireOwner}, false); ownerErr == nil {
				return Sticky{}, ErrStickyForbidden
			}
		}
		return Sticky{}, ErrStickyNotFound
	}
	if err != nil {
		return Sticky{}, fmt.Errorf("patch Sticky: %w", err)
	}
	return sticky, nil
}

func (repository *Repository) Delete(ctx context.Context, stickyID StickyID, scope Scope) error {
	return repository.delete(ctx, stickyID, scope, true)
}

// DeleteOwned applies the API v1 owner/workspace scope without a creator check.
func (repository *Repository) DeleteOwned(ctx context.Context, stickyID StickyID, scope Scope) error {
	return repository.delete(ctx, stickyID, scope, false)
}

func (repository *Repository) delete(ctx context.Context, stickyID StickyID, scope Scope, requireCreator bool) error {
	if uuid.UUID(stickyID) == uuid.Nil() || !validOwnerScope(scope, OperationDelete, requireCreator) {
		return ErrInvalidScope
	}
	tx, err := repository.database.Begin(ctx)
	if err != nil {
		return fmt.Errorf("delete Sticky: begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	query := `UPDATE stickies SET deleted_at=CURRENT_TIMESTAMP,updated_at=CURRENT_TIMESTAMP,updated_by_id=$1 WHERE id=$2 AND workspace_id=$3 AND owner_id=$1`
	if requireCreator {
		query += " AND created_by_id=$1"
	}
	query += " AND deleted_at IS NULL"
	command, err := tx.Exec(ctx, query, uuid.UUID(scope.UserID), uuid.UUID(stickyID), uuid.UUID(scope.WorkspaceID))
	if err != nil {
		return fmt.Errorf("delete Sticky: %w", err)
	}
	if command.RowsAffected() != 1 {
		if requireCreator {
			if _, ownerErr := repository.one(ctx, stickyID, Scope{Operation: OperationRetrieve, WorkspaceID: scope.WorkspaceID, UserID: scope.UserID, Ownership: RequireOwner}, false); ownerErr == nil {
				return ErrStickyForbidden
			}
		}
		return ErrStickyNotFound
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("delete Sticky: commit transaction: %w", err)
	}
	return nil
}

func (repository *Repository) one(ctx context.Context, id StickyID, scope Scope, creator bool) (Sticky, error) {
	query := `SELECT ` + stickyColumns + ` FROM stickies s WHERE s.id=$1 AND s.workspace_id=$2 AND s.owner_id=$3 AND s.deleted_at IS NULL AND EXISTS (SELECT 1 FROM workspace_members wm WHERE wm.workspace_id=s.workspace_id AND wm.member_id=$3 AND wm.is_active IS TRUE AND wm.deleted_at IS NULL)`
	if creator {
		query += " AND s.created_by_id=$3"
	}
	var s Sticky
	err := scanSticky(repository.database.QueryRow(ctx, query, uuid.UUID(id), uuid.UUID(scope.WorkspaceID), uuid.UUID(scope.UserID)), &s)
	if errors.Is(err, pgx.ErrNoRows) {
		return Sticky{}, ErrStickyNotFound
	}
	if err != nil {
		return Sticky{}, fmt.Errorf("retrieve Sticky: %w", err)
	}
	return s, nil
}

const stickyColumns = "id, created_at, updated_at, deleted_at, name, description, description_html, description_stripped, description_binary, logo_props, color, background_color, created_by_id, owner_id, updated_by_id, workspace_id, sort_order"

type scanner interface{ Scan(...any) error }

func scanSticky(row scanner, s *Sticky) error {
	var id, owner, workspace uuid.UUID
	var createdBy, updatedBy *uuid.UUID
	err := row.Scan(&id, &s.CreatedAt, &s.UpdatedAt, &s.DeletedAt, &s.Name, &s.Description, &s.DescriptionHTML, &s.DescriptionStripped, &s.DescriptionBinary, &s.LogoProps, &s.Color, &s.BackgroundColor, &createdBy, &owner, &updatedBy, &workspace, &s.SortOrder)
	if err != nil {
		return err
	}
	s.ID = StickyID(id)
	if createdBy != nil {
		value := UserID(*createdBy)
		s.CreatedByID = &value
	}
	s.OwnerID = UserID(owner)
	if updatedBy != nil {
		value := UserID(*updatedBy)
		s.UpdatedByID = &value
	}
	s.WorkspaceID = WorkspaceID(workspace)
	return nil
}
func validOwnerScope(scope Scope, operation Operation, creator bool) bool {
	return scope.Operation == operation && uuid.UUID(scope.WorkspaceID) != uuid.Nil() && uuid.UUID(scope.UserID) != uuid.Nil() && scope.Ownership == RequireOwner && scope.RequireCreator == creator
}
func join(parts []string, separator string) string {
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for _, part := range parts[1:] {
		result += separator + part
	}
	return result
}
