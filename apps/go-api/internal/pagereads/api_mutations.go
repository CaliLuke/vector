package pagereads

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"math"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/CaliLuke/go-plane/internal/mutationreceipts"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/stickies/content"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func normalizePageAPI(in *PageAPICommand) error {
	allowed := []string{}
	switch in.Tool {
	case "page":
		switch in.Action {
		case "create":
			allowed = []string{"name", "description_html", "access", "color", "is_locked", "external_source", "external_id"}
			if in.ParentID != uuid.Nil && in.CollectionID != uuid.Nil || in.ProjectID != uuid.Nil && in.CollectionID != uuid.Nil {
				return projectmutations.ErrInvalid
			}
		case "update":
			allowed = []string{"name", "description_html"}
		case "archive":
			allowed = []string{"archive"}
		case "delete", "set_collection", "attach_to_workitem", "detach_from_workitem":
		default:
			return projectmutations.ErrInvalid
		}
	case "collection":
		switch in.Action {
		case "create":
			allowed = []string{"name", "access"}
		case "update":
			allowed = []string{"name", "sort_order"}
		case "delete":
			allowed = []string{"archive_pages"}
		case "add_member", "update_member":
			allowed = []string{"member_access"}
		case "remove_member", "add_pages", "remove_page":
		default:
			return projectmutations.ErrInvalid
		}
	default:
		return projectmutations.ErrInvalid
	}
	in.Fields = maps.Clone(in.Fields)
	if in.Fields == nil {
		in.Fields = map[string]any{}
	}
	for key, value := range in.Fields {
		if !slices.Contains(allowed, key) {
			return projectmutations.ErrInvalid
		}
		switch key {
		case "access", "member_access":
			n, ok := value.(int)
			max := 1
			if key == "member_access" {
				max = 2
			}
			if !ok || n < 0 || n > max {
				return projectmutations.ErrInvalid
			}
		case "is_locked", "archive", "archive_pages":
			if _, ok := value.(bool); !ok {
				return projectmutations.ErrInvalid
			}
		case "sort_order":
			n, ok := value.(float64)
			if !ok || math.IsNaN(n) || math.IsInf(n, 0) {
				return projectmutations.ErrInvalid
			}
		default:
			text, ok := value.(string)
			if !ok || strings.ContainsRune(text, 0) {
				return projectmutations.ErrInvalid
			}
			if key != "description_html" && utf8.RuneCountInString(text) > 255 {
				return projectmutations.ErrInvalid
			}
			if key == "name" && strings.TrimSpace(text) == "" {
				return projectmutations.ErrInvalid
			}
			if key == "description_html" {
				clean, _, err := content.ValidateHTML(text)
				if err != nil {
					return projectmutations.ErrInvalid
				}
				in.Fields[key] = clean
			}
		}
	}
	if in.ExpectedRevision != nil && *in.ExpectedRevision == "" {
		return projectmutations.ErrInvalid
	}
	if in.Action == "create" {
		if in.Fields["name"] == nil {
			return projectmutations.ErrInvalid
		}
		if in.Tool == "page" && in.Fields["description_html"] == nil {
			return projectmutations.ErrInvalid
		}
	}
	if in.Action == "update" && len(in.Fields) == 0 {
		return projectmutations.ErrInvalid
	}
	if in.Tool == "collection" && in.Action != "create" && in.CollectionID == uuid.Nil {
		return projectmutations.ErrInvalid
	}
	if in.Tool == "page" && in.Action != "create" && in.Action != "detach_from_workitem" && in.PageID == uuid.Nil {
		return projectmutations.ErrInvalid
	}
	if in.Action == "set_collection" && (in.CollectionID == uuid.Nil || in.ProjectID != uuid.Nil) {
		return projectmutations.ErrInvalid
	}
	if strings.HasSuffix(in.Action, "_workitem") && (in.ProjectID == uuid.Nil || in.ItemID == uuid.Nil) {
		return projectmutations.ErrInvalid
	}
	if (in.Action == "detach_from_workitem" || in.Action == "remove_page") && in.LinkID == uuid.Nil {
		return projectmutations.ErrInvalid
	}
	if in.Action == "add_member" && in.UserID == uuid.Nil || (in.Action == "update_member" || in.Action == "remove_member") && in.MemberID == uuid.Nil {
		return projectmutations.ErrInvalid
	}
	if (in.Action == "add_member" || in.Action == "update_member") && in.Fields["member_access"] == nil {
		return projectmutations.ErrInvalid
	}
	if len(in.PageIDs) > 1000 || in.Action == "add_pages" && len(in.PageIDs) == 0 {
		return projectmutations.ErrInvalid
	}
	for _, id := range in.PageIDs {
		if id == uuid.Nil {
			return projectmutations.ErrInvalid
		}
	}
	slices.SortFunc(in.PageIDs, func(a, b uuid.UUID) int { return strings.Compare(a.String(), b.String()) })
	in.PageIDs = slices.Compact(in.PageIDs)
	return nil
}
func authorizePageAPI(ctx context.Context, tx pgx.Tx, in PageAPICommand) error {
	if _, err := tx.Exec(ctx, `SELECT work_item_schema_lock($1::uuid)`, in.WorkspaceID); err != nil {
		return err
	}
	role, err := pageAPIWorkspace(ctx, tx, in.PageAPIQuery, true)
	if err != nil {
		return err
	}
	if in.ProjectID != uuid.Nil {
		role, err = pageAPIProject(ctx, tx, in.PageAPIQuery, true)
		if err != nil {
			return err
		}
	}
	if in.Tool == "collection" {
		if in.Action == "create" {
			if role < 15 {
				return projectmutations.ErrForbidden
			}
		} else {
			row, _, collectionScopeErr := collectionScope(ctx, tx, in.PageAPIQuery, in.CollectionID, true, in.Action == "delete")
			if collectionScopeErr != nil {
				return collectionScopeErr
			}
			if (strings.HasSuffix(in.Action, "member") || in.Action == "delete") && row["owned_by_id"] != in.ActorID.String() {
				return projectmutations.ErrForbidden
			}
		}
		if in.Action == "add_pages" {
			for _, id := range in.PageIDs {
				if _, err = pageBranchForMove(ctx, tx, in.PageAPIQuery, id); err != nil {
					return err
				}
			}
		}
		if in.Action == "remove_page" {
			var page uuid.UUID
			err = tx.QueryRow(ctx, `SELECT page_id FROM page_collection_pages WHERE id=$1 AND collection_id=$2 AND workspace_id=$3`, in.LinkID, in.CollectionID, in.WorkspaceID).Scan(&page)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return err
			}
			if err == nil {
				if _, err = pageBranchForMove(ctx, tx, in.PageAPIQuery, page); err != nil {
					return err
				}
			}
		}
		if in.Action == "add_member" {
			var member uuid.UUID
			err = tx.QueryRow(ctx, `SELECT u.id FROM users u JOIN workspace_members wm ON wm.member_id=u.id WHERE u.id=$1 AND u.is_active AND wm.workspace_id=$2 AND wm.is_active AND wm.deleted_at IS NULL AND wm.role IN(5,15,20) FOR SHARE OF u,wm`, in.UserID, in.WorkspaceID).Scan(&member)
			if errors.Is(err, pgx.ErrNoRows) {
				return projectmutations.ErrNotFound
			}
			if err != nil {
				return err
			}
		}
	} else if in.Action == "create" {
		if role < 15 && in.ParentID == uuid.Nil && in.CollectionID == uuid.Nil {
			return projectmutations.ErrForbidden
		}
		if in.ParentID != uuid.Nil {
			parent, pageScopeErr := pageScope(ctx, tx, in.PageAPIQuery, in.ParentID, true, false, false)
			if pageScopeErr != nil {
				return pageScopeErr
			}
			if parent["archived_at"] != nil || parent["is_locked"] == true {
				return projectmutations.ErrInvalid
			}
		}
		if in.CollectionID != uuid.Nil {
			if _, _, err = collectionScope(ctx, tx, in.PageAPIQuery, in.CollectionID, true, false); err != nil {
				return err
			}
		}
	} else if in.Action == "attach_to_workitem" || in.Action == "detach_from_workitem" {
		if err = pageAPIItem(ctx, tx, in.PageAPIQuery, true); err != nil {
			return err
		}
		page := in.PageID
		if in.Action == "detach_from_workitem" {
			err = tx.QueryRow(ctx, `SELECT page_id FROM work_item_pages WHERE id=$1 AND issue_id=$2 AND project_id=$3 AND workspace_id=$4`, in.LinkID, in.ItemID, in.ProjectID, in.WorkspaceID).Scan(&page)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return err
			}
			if errors.Is(err, pgx.ErrNoRows) {
				page = uuid.Nil
			}
		}
		if page != uuid.Nil {
			if _, err = pageScope(ctx, tx, in.PageAPIQuery, page, false, false, true); err != nil {
				return err
			}
		}
	} else {
		if _, err = pageScope(ctx, tx, in.PageAPIQuery, in.PageID, true, in.Action == "delete", false); err != nil {
			return err
		}
		if in.Action == "set_collection" {
			if _, _, err = collectionScope(ctx, tx, in.PageAPIQuery, in.CollectionID, true, false); err != nil {
				return err
			}
			if _, err = pageBranchForMove(ctx, tx, in.PageAPIQuery, in.PageID); err != nil {
				return err
			}
		}
	}
	if in.Revalidate != nil {
		return in.Revalidate(ctx)
	}
	return nil
}
func (r *Repository) MutatePageAPI(ctx context.Context, in PageAPICommand) (projectmutations.Result, error) {
	if err := normalizePageAPI(&in); err != nil {
		return projectmutations.Result{}, err
	}
	db, ok := r.database.(mutationreceipts.Database)
	if !ok {
		return projectmutations.Result{}, projectmutations.ErrInvalid
	}
	normalized, err := json.Marshal(struct {
		Query    PageAPIQuery
		Fields   map[string]any
		IDs      []uuid.UUID
		Revision *string
	}{in.PageAPIQuery, in.Fields, in.PageIDs, in.ExpectedRevision})
	if err != nil {
		return projectmutations.Result{}, err
	}
	var project *uuid.UUID
	if in.ProjectID != uuid.Nil {
		project = &in.ProjectID
	}
	receipt, err := mutationreceipts.Execute(ctx, db, mutationreceipts.Command{ActorID: in.ActorID, WorkspaceID: in.WorkspaceID, ProjectID: project, Tool: in.Tool, Action: in.Action, RequestID: in.RequestID, NormalizedInput: normalized}, mutationreceipts.Operation{
		Authorize: func(ctx context.Context, tx pgx.Tx) error { return authorizePageAPI(ctx, tx, in) },
		Apply: func(ctx context.Context, tx pgx.Tx) (json.RawMessage, error) {
			var rev int64
			if scanErr := tx.QueryRow(ctx, `SELECT revision FROM page_api_revisions WHERE workspace_id=$1 FOR UPDATE`, in.WorkspaceID).Scan(&rev); scanErr != nil {
				return nil, scanErr
			}
			if in.ExpectedRevision != nil && *in.ExpectedRevision != pageAPIToken(in.WorkspaceID, rev) {
				return nil, projectmutations.ErrRevision
			}
			var before any
			if in.Action != "create" {
				name, id := "pages", in.PageID
				if in.Tool == "collection" {
					name, id = "page_collections", in.CollectionID
				}
				if id != uuid.Nil {
					row, pageAPIRowErr := pageAPIRow(ctx, tx, `SELECT to_jsonb(t) FROM `+name+` t WHERE id=$1 AND workspace_id=$2 AND deleted_at IS NULL`, id, in.WorkspaceID)
					if pageAPIRowErr != nil {
						return nil, pageAPIRowErr
					}
					before = row
				}
			}
			var id uuid.UUID
			var after any
			var operationErr error
			if in.Tool == "collection" {
				id, after, operationErr = applyCollectionAPI(ctx, tx, in)
			} else {
				id, after, operationErr = r.applyPageAPI(ctx, tx, in)
			}
			if operationErr != nil {
				return nil, operationErr
			}
			old, _ := json.Marshal(before)
			current, _ := json.Marshal(after)
			if _, operationErr = tx.Exec(ctx, `INSERT INTO page_api_events(workspace_id,actor_id,resource_id,resource_kind,action,before_value,after_value) VALUES($1,$2,$3,$4,$5,$6::jsonb,$7::jsonb)`, in.WorkspaceID, in.ActorID, id, in.Tool, in.Action, old, current); operationErr != nil {
				return nil, operationErr
			}
			if operationErr = tx.QueryRow(ctx, `SELECT revision FROM page_api_revisions WHERE workspace_id=$1`, in.WorkspaceID).Scan(&rev); operationErr != nil {
				return nil, operationErr
			}
			raw, operationErr := json.Marshal(after)
			if operationErr != nil {
				return nil, operationErr
			}
			return json.Marshal(projectmutations.Result{ID: id.String(), Revision: pageAPIToken(in.WorkspaceID, rev), Record: raw})
		},
	})
	if err != nil {
		return projectmutations.Result{}, err
	}
	var result projectmutations.Result
	if err = json.Unmarshal(receipt.Value, &result); err != nil {
		return result, err
	}
	if in.RequestID != nil {
		result.RequestID, result.ReplayExpiresAt, result.Replayed = in.RequestID, receipt.ReplayExpiresAt, &receipt.Replayed
	}
	return result, nil
}
