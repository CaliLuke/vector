package workitemtypes

import (
	"cmp"
	"context"
	"slices"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/formula"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
	"github.com/jackc/pgx/v5"
)

type DiscoveryPropertyChoice struct {
	PropertyChoice
	Icon   string `json:"icon,omitempty"`
	Parent string `json:"parent,omitempty"`
}

type DiscoveryProperty struct {
	Key        string                    `json:"key"`
	PropertyID string                    `json:"property_id"`
	Name       string                    `json:"name"`
	Icon       string                    `json:"icon"`
	Kind       property.Kind             `json:"kind"`
	ResultKind formula.Kind              `json:"result_kind"`
	LevelID    string                    `json:"level_id"`
	CatalogID  string                    `json:"catalog_id"`
	Multiple   bool                      `json:"multiple"`
	Groupable  bool                      `json:"groupable"`
	Operators  []string                  `json:"operators"`
	Options    []DiscoveryPropertyChoice `json:"options"`
}

type PropertyDiscoveryCatalog struct {
	Enabled  bool                `json:"enabled"`
	Revision int64               `json:"revision"`
	Fields   []DiscoveryProperty `json:"fields"`
}

// DiscoveryProperties exposes effective display/filter identities only. It has
// no usage counts, defaults, formulas, scopes or inactive/private configurations.
func (r *Repository) DiscoveryProperties(ctx context.Context, actor, slug, project string) (PropertyDiscoveryCatalog, error) {
	return WithTypeDiscoveryRead(ctx, r.db, actor, slug, project, func(tx pgx.Tx, types TypeIdentityCatalog) (PropertyDiscoveryCatalog, error) {
		out := PropertyDiscoveryCatalog{Enabled: types.Enabled, Revision: types.Revision, Fields: []DiscoveryProperty{}}
		if !types.Enabled {
			return out, nil
		}
		snapshot, err := loadPropertyDiscovery(ctx, tx, actor, slug, project, types)
		if err != nil {
			return out, err
		}
		fields, err := snapshot.discoveryFields()
		if err != nil {
			return out, err
		}
		if err = snapshot.discoveryChoices(ctx, tx, fields); err != nil {
			return out, err
		}
		for _, field := range fields {
			out.Fields = append(out.Fields, *field)
		}
		slices.SortFunc(out.Fields, func(a, b DiscoveryProperty) int {
			if order := cmp.Compare(a.Name, b.Name); order != 0 {
				return order
			}
			return cmp.Compare(a.Key, b.Key)
		})
		return out, nil
	})
}

func (snapshot *PropertyDiscoverySnapshot) discoveryFields() (map[string]*DiscoveryProperty, error) {
	fields := map[string]*DiscoveryProperty{}
	for typeID, definitions := range snapshot.definitions {
		for _, def := range definitions {
			for projectID := range snapshot.projects {
				config, owner, err := property.OptionCatalog(def, property.Scope{ProjectID: projectID, TypeID: typeID})
				if err != nil {
					return nil, err
				}
				if !config.Active {
					continue
				}
				base := DiscoveryProperty{Key: "customproperty_" + def.ID, PropertyID: def.ID, Name: def.Title, Icon: def.Icon, Kind: def.Kind, ResultKind: config.Result, Multiple: config.Multiple, Operators: propertyPredicateOperators(def.Kind, config.Result), Options: []DiscoveryPropertyChoice{}}
				base.Groupable = def.Kind == property.Boolean || !config.Multiple && (def.Kind == property.Dropdown || def.Kind == property.Member || def.Kind == property.Release || def.Kind == property.Issue)
				if def.Kind != property.Cascading {
					if def.Kind == property.Dropdown {
						for _, option := range config.Options {
							base.Options = append(base.Options, DiscoveryPropertyChoice{PropertyChoice: PropertyChoice{ID: option.ID, Name: option.Name}, Icon: option.Icon, Parent: option.Parent})
						}
					}
					if previous := fields[base.Key]; previous != nil {
						base.Groupable = base.Groupable && previous.Groupable
						base.Multiple = base.Multiple || previous.Multiple
					}
					fields[base.Key] = &base
					continue
				}
				catalogName, catalogKey := "", "base"
				if owner != "" {
					catalogKey = owner
					for _, context := range def.Contexts {
						if context.ID == owner {
							catalogName = " (" + context.Name + ")"
						}
					}
				}
				for index, level := range config.Levels {
					field := base
					field.Key += "_level_" + level.ID + "_catalog_" + catalogKey
					field.Name += " · " + level.Name + catalogName
					field.LevelID, field.CatalogID, field.Groupable = level.ID, owner, true
					field.Options = []DiscoveryPropertyChoice{}
					for _, option := range config.Options {
						if option.Level == index {
							field.Options = append(field.Options, DiscoveryPropertyChoice{PropertyChoice: PropertyChoice{ID: option.ID, Name: option.Name}, Icon: option.Icon, Parent: option.Parent})
						}
					}
					fields[field.Key] = &field
				}
			}
		}
	}
	if len(fields) > 4000 {
		return nil, reject(422, "There are too many effective property fields to load this discovery catalog.")
	}
	return fields, nil
}

func (snapshot *PropertyDiscoverySnapshot) discoveryChoices(ctx context.Context, tx pgx.Tx, fields map[string]*DiscoveryProperty) error {
	choices := map[property.Kind][]PropertyChoice{}
	for _, field := range fields {
		if field.Kind != property.Member && field.Kind != property.Release && field.Kind != property.Issue {
			continue
		}
		options, exists := choices[field.Kind]
		if !exists {
			query := `SELECT u.id::text,COALESCE(NULLIF(u.display_name,''),NULLIF(trim(u.first_name||' '||u.last_name),''),u.username) FROM users u JOIN workspace_members m ON m.member_id=u.id WHERE m.workspace_id=$1 AND m.is_active AND m.deleted_at IS NULL AND u.is_active ORDER BY u.id`
			args := []any{snapshot.workspace}
			if field.Kind == property.Release {
				projects := []string{}
				for id := range snapshot.projects {
					projects = append(projects, id)
				}
				query = `SELECT r.id::text,CASE WHEN r.project_id IS NULL THEN r.name ELSE p.name||' · '||r.name END FROM work_item_releases r LEFT JOIN projects p ON p.id=r.project_id AND p.workspace_id=r.workspace_id WHERE r.workspace_id=$1 AND (r.project_id=ANY($2::uuid[]) OR r.project_id IS NULL) AND r.deleted_at IS NULL ORDER BY p.name,r.name,r.id`
				args = append(args, projects)
			}
			if field.Kind == property.Issue {
				query = `SELECT i.id::text,p.identifier||'-'||i.sequence_id||' · '||i.name FROM issues i JOIN projects p ON p.id=i.project_id AND p.workspace_id=i.workspace_id WHERE ` + djangoapi.IssueVisibleSQL("i", "$2::uuid", "$1::uuid") + ` ORDER BY p.identifier,i.sequence_id,i.id`
				args = []any{snapshot.workspace, snapshot.actor}
			}
			var err error
			options, err = propertyChoices(ctx, tx, query, args...)
			if err != nil {
				return err
			}
			choices[field.Kind] = options
		}
		field.Options = make([]DiscoveryPropertyChoice, 0, len(options))
		for _, option := range options {
			field.Options = append(field.Options, DiscoveryPropertyChoice{PropertyChoice: option})
		}
	}
	return nil
}
