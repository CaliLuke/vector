// Package deletion contains the evaluated source model policy. PostgreSQL owns
// physical cascading; this registry limits soft deletion to known Plane models.
package deletion

import (
	"embed"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

//go:embed model_policy.json
var policyFiles embed.FS

type Relation struct {
	Column       string `json:"column"`
	TargetTable  string `json:"target_table"`
	TargetColumn string `json:"target_column"`
	OnDelete     string `json:"on_delete"`
	OneToOne     bool   `json:"one_to_one"`
	SoftReverse  bool   `json:"soft_reverse"`
}
type Model struct {
	App        string     `json:"app"`
	Name       string     `json:"model"`
	Table      string     `json:"table"`
	PK         string     `json:"pk"`
	PKType     string     `json:"pk_type,omitempty"`
	SoftDelete bool       `json:"soft_delete"`
	UpdatedAt  bool       `json:"updated_at"`
	Relations  []Relation `json:"relations"`
}
type Policy struct {
	SourceRevision string  `json:"source_revision"`
	Models         []Model `json:"models"`
}

func Load() (Policy, error) {
	data, err := policyFiles.ReadFile("model_policy.json")
	if err != nil {
		return Policy{}, err
	}
	var result Policy
	if err = json.Unmarshal(data, &result); err != nil {
		return Policy{}, err
	}
	return result, nil
}
func (p Policy) Lookup(app, name string) (Model, error) {
	for _, m := range p.Models {
		if m.App == app && m.Name == strings.ToLower(name) {
			return m, nil
		}
	}
	return Model{}, fmt.Errorf("unknown deletion model %s.%s", app, name)
}

// ParseID keeps identifiers typed for PostgreSQL and rejects IDs incompatible
// with the captured model. Most models use UUIDs; project identifiers use bigint.
func (m Model) ParseID(raw string) (any, error) {
	switch m.PKType {
	case "", "uuid":
		return uuid.Parse(raw)
	case "bigint":
		return strconv.ParseInt(raw, 10, 64)
	default:
		return nil, fmt.Errorf("unsupported deletion primary key type %q", m.PKType)
	}
}
