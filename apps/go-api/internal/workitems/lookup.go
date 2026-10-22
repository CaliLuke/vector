package workitems

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"
)

func (r *Repository) rows(ctx context.Context, query string, args ...any) ([]map[string]any, error) {
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var raw []byte
		if err = rows.Scan(&raw); err != nil {
			return nil, err
		}
		value, err := decodeObject(raw)
		if err != nil {
			return nil, err
		}
		out = append(out, value)
	}
	return out, rows.Err()
}

func (r *Repository) identifier(ctx context.Context, q Request) (Result, error) {
	projectName, sequence, ok := splitIdentifier(q.Params["identifier"])
	if !ok {
		return Result{}, failure(404, "error", "Page not found.")
	}
	if !q.API {
		digits := strings.TrimPrefix(sequence, "-")
		valid := digits != ""
		for _, c := range digits {
			if !unicode.IsDigit(c) {
				valid = false
			}
		}
		if !valid {
			return Result{}, failure(400, "error", "Invalid issue identifier")
		}
	}
	n, err := pythoncompat.DecimalInteger(sequence)
	if err != nil {
		if !q.API {
			return Result{}, failure(400, "error", "Invalid issue identifier")
		}
		return Result{}, err
	}
	if q.API {
		// API lookup traverses the project FK inside Issue.get(). A deleted
		// project still participates, and duplicate identifiers can match twice.
		rows, e := r.rows(ctx, `SELECT `+djangoapi.IssueRowSQL("i")+` FROM issues i JOIN projects p ON p.id=i.project_id JOIN workspaces w ON w.id=i.workspace_id WHERE w.slug=$1 AND p.identifier=$2 AND i.sequence_id::numeric=$3::numeric AND `+liveIssue, q.Slug, projectName, n.String())
		if e != nil {
			return Result{}, e
		}
		if len(rows) == 0 {
			return Result{}, failure(404, "error", "The requested resource does not exist.")
		}
		if len(rows) > 1 {
			return Result{}, errors.New("multiple work items match identifier")
		}
		v, e := djangoapi.SerializeIssue(ctx, r.db, rows[0], q.Params["fields"], q.Params["expand"])
		return Result{200, v}, e
	}
	comparison := "p.identifier=$2"
	if !q.API {
		comparison = "UPPER(p.identifier)=UPPER($2)"
	}
	projects, err := r.rows(ctx, `SELECT to_jsonb(p) FROM projects p JOIN workspaces w ON w.id=p.workspace_id WHERE w.slug=$1 AND `+comparison+` AND p.deleted_at IS NULL ORDER BY p.created_at DESC`, q.Slug, projectName)
	if err != nil {
		return Result{}, err
	}
	if len(projects) == 0 {
		return Result{}, requiredObject()
	}
	if len(projects) > 1 {
		return Result{}, errors.New("multiple projects match identifier")
	}
	project := projects[0]
	q.ProjectID = project["id"].(string)
	if !q.API {
		var member bool
		err = r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM project_members pm JOIN workspaces w ON w.id=pm.workspace_id WHERE w.slug=$1 AND pm.project_id=$2 AND pm.member_id=$3 AND pm.is_active AND pm.deleted_at IS NULL)`, q.Slug, q.ProjectID, q.ActorID).Scan(&member)
		if err != nil {
			return Result{}, err
		}
		if !member {
			return Result{}, failure(403, "error", "You are not allowed to view this issue")
		}
	}
	predicate := "i.deleted_at IS NULL"
	if q.API {
		predicate = liveIssue
	}
	values, err := r.rows(ctx, `SELECT `+djangoapi.IssueRowSQL("i")+` FROM issues i JOIN workspaces w ON w.id=i.workspace_id WHERE w.slug=$1 AND i.project_id=$2 AND i.sequence_id::numeric=$3::numeric AND `+predicate+` ORDER BY i.created_at DESC`, q.Slug, q.ProjectID, n.String())
	if err != nil {
		return Result{}, err
	}
	if len(values) == 0 {
		if q.API {
			return Result{}, failure(404, "error", "The requested resource does not exist.")
		}
		return Result{}, requiredObject()
	}
	row := values[0]
	q.IssueID = row["id"].(string)
	if q.API {
		v, e := djangoapi.SerializeIssue(ctx, r.db, row, q.Params["fields"], q.Params["expand"])
		return Result{200, v}, e
	}
	if err = r.guestIssue(ctx, q, row, project); err != nil {
		return Result{}, err
	}
	if err = r.recentVisit(ctx, q, "issue", q.IssueID); err != nil {
		return Result{}, err
	}
	v, err := r.detail(ctx, row, q, true, true)
	return Result{200, v}, err
}

func escapeSearch(raw string) string {
	return strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(raw)
}

// Python's Unicode word boundary includes letters, digits and underscores.
func searchSequences(query string) []string {
	chars := []rune(query)
	out := []string{}
	word := func(c rune) bool { return unicode.IsLetter(c) || unicode.IsNumber(c) || c == '_' }
	for i := 0; i < len(chars); {
		if !unicode.IsDigit(chars[i]) {
			i++
			continue
		}
		start := i
		for i < len(chars) && unicode.IsDigit(chars[i]) {
			i++
		}
		if (start == 0 || !word(chars[start-1])) && (i == len(chars) || !word(chars[i])) {
			if n, e := pythoncompat.DecimalInteger(string(chars[start:i])); e == nil {
				out = append(out, n.String())
			}
		}
	}
	return out
}
func (r *Repository) search(ctx context.Context, q Request) (Result, error) {
	query := q.Params["search"]
	if q.API && query == "" {
		return Result{200, map[string]any{"issues": []any{}}}, nil
	}
	args := []any{q.Slug, q.ActorID}
	where := []string{liveIssue, `w.slug=$1`, `EXISTS(SELECT 1 FROM project_members pm WHERE pm.project_id=i.project_id AND pm.member_id=$2 AND pm.is_active)`}
	add := func(value any, format string) {
		args = append(args, value)
		where = append(where, fmt.Sprintf(format, len(args)))
	}
	workspace := q.Params["workspace_search"]
	if workspace == "" {
		workspace = "false"
	}
	project := q.ProjectID
	if q.API {
		project = q.Params["project_id"]
	}
	if workspace == "false" && project != "" {
		id, e := djangoinput.UUID(project)
		if e != nil {
			return Result{}, failure(400, "error", "Please provide valid detail")
		}
		add(id.String(), `i.project_id=$%d`)
	}
	if query != "" {
		args = append(args, "%"+escapeSearch(query)+"%")
		pos := len(args)
		clauses := []string{fmt.Sprintf(`i.name ILIKE $%d`, pos), fmt.Sprintf(`p.identifier ILIKE $%d`, pos)}
		if q.API || utf8.RuneCountInString(query) <= 20 {
			seq := searchSequences(query)
			if len(seq) > 0 {
				args = append(args, seq)
				clauses = append(clauses, fmt.Sprintf(`i.sequence_id::text=ANY($%d::text[])`, len(args)))
			}
		} else {
			clauses = append(clauses, fmt.Sprintf(`i.sequence_id::text ILIKE $%d`, pos))
		}
		where = append(where, "("+strings.Join(clauses, " OR ")+")")
	}
	if !q.API {
		if id := q.Params["issue_id"]; id != "" && (q.Params["parent"] == "true" || q.Params["issue_relation"] == "true" || q.Params["sub_issue"] == "true") {
			parsed, e := djangoinput.UUID(id)
			if e != nil {
				return Result{}, failure(400, "error", "Please provide valid detail")
			}
			anchors, e := r.rows(ctx, `SELECT to_jsonb(i) FROM issues i WHERE i.id=$1 AND `+liveIssue, parsed.String())
			if e != nil {
				return Result{}, e
			}
			if len(anchors) == 0 && q.Params["sub_issue"] == "true" {
				return Result{}, errors.New("search parent missing")
			}
			if len(anchors) > 0 {
				a := anchors[0]
				add(parsed.String(), `i.id<>$%d`)
				if q.Params["parent"] == "true" {
					if a["parent_id"] != nil {
						add(a["parent_id"], `i.id<>$%d`)
					}
					add(parsed.String(), `(i.parent_id IS NULL OR i.parent_id<>$%d)`)
				}
				if q.Params["sub_issue"] == "true" {
					where = append(where, "i.parent_id IS NULL")
					if a["parent_id"] != nil {
						add(a["parent_id"], `i.id<>$%d`)
					}
				}
				if q.Params["issue_relation"] == "true" {
					add(parsed.String(), `NOT EXISTS(SELECT 1 FROM issue_relations rel WHERE rel.deleted_at IS NULL AND ((rel.issue_id=$%[1]d AND rel.related_issue_id=i.id) OR (rel.related_issue_id=$%[1]d AND rel.issue_id=i.id)))`)
				}
			}
		}
		if q.Params["cycle"] == "true" {
			where = append(where, `NOT EXISTS(SELECT 1 FROM cycle_issues ci WHERE ci.issue_id=i.id AND ci.deleted_at IS NULL)`)
		}
		if module := q.Params["module"]; module != "" {
			id, e := djangoinput.UUID(module)
			if e != nil {
				return Result{}, failure(400, "error", "Please provide valid detail")
			}
			add(id.String(), `NOT EXISTS(SELECT 1 FROM module_issues mi WHERE mi.issue_id=i.id AND mi.module_id=$%d AND mi.deleted_at IS NULL)`)
		}
		if q.Params["target_date"] == "none" {
			where = append(where, `i.target_date IS NULL`)
		}
		args = append(args, q.ProjectID)
		where = append(where, fmt.Sprintf(`(i.created_by_id=$2 OR NOT EXISTS(SELECT 1 FROM project_members pm WHERE pm.project_id=$%d AND pm.member_id=$2 AND pm.is_active AND pm.role=5 AND pm.deleted_at IS NULL))`, len(args)))
	}
	limit := 100
	if q.API {
		limit = 10
		if q.Present["limit"] {
			n, e := pythoncompat.DecimalInteger(q.Params["limit"])
			if e != nil || !n.IsInt64() || n.Sign() < 0 {
				return Result{}, errors.New("invalid search limit")
			}
			limit = int(n.Int64())
		}
	}
	fields := `'name',i.name,'id',i.id,'sequence_id',i.sequence_id,'project__identifier',p.identifier,'project_id',i.project_id,'workspace__slug',w.slug`
	if !q.API {
		fields += `,'start_date',i.start_date,'project__name',p.name,'state__name',s.name,'state__group',s."group",'state__color',s.color`
	}
	args = append(args, limit)
	rows, err := r.rows(ctx, `SELECT jsonb_build_object(`+fields+`) FROM issues i JOIN projects p ON p.id=i.project_id JOIN workspaces w ON w.id=i.workspace_id LEFT JOIN states s ON s.id=i.state_id WHERE `+strings.Join(where, " AND ")+fmt.Sprintf(` ORDER BY i.created_at DESC LIMIT $%d`, len(args)), args...)
	if q.API {
		return Result{200, map[string]any{"issues": rows}}, err
	}
	return Result{200, rows}, err
}
