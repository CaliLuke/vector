// Package routing owns the application-operation cutover inventory.
package routing

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
)

const (
	StateDjangoOnly    = "Django only"
	StateGoImplemented = "Go implemented"
	StateGoParity      = "Go parity proven"
	StateGoActive      = "Go active"
	StateDjangoRemoved = "Django removed"
)

// PreRetirementOperations is the number of application operations the committed
// OpenAPI contract described before this migration removed any of them. It is
// the fixed universe every progress measure divides by, and the only count in
// this package that a retirement must never change.
const PreRetirementOperations = 639

var allowedStates = map[string]struct{}{
	StateDjangoOnly:    {},
	StateGoImplemented: {},
	StateGoParity:      {},
	StateGoActive:      {},
	StateDjangoRemoved: {},
}

//go:embed route-inventory.json
var embeddedInventory []byte

// Inventory is the complete migration routing policy.
type Inventory struct {
	Version       int         `json:"version"`
	Source        string      `json:"source"`
	DefaultState  string      `json:"default_state"`
	AllowedStates []string    `json:"allowed_states"`
	Operations    []Operation `json:"operations"`
}

// Operation is one source OpenAPI operation and its migration state.
type Operation struct {
	Method             string                   `json:"method"`
	Path               string                   `json:"path"`
	OperationID        string                   `json:"operation_id"`
	State              string                   `json:"state"`
	Implementation     *string                  `json:"implementation"`
	ParityEvidence     *ParityEvidence          `json:"parity_evidence,omitempty"`
	ProtocolCompanions ProtocolCompanionMethods `json:"protocol_companions,omitzero"`
	// AllowedProtocolCompanions overrides which routed companions appear in Allow.
	// An explicit empty list preserves source views that reject every companion.
	AllowedProtocolCompanions ProtocolCompanionMethods `json:"allowed_protocol_companions,omitzero"`
}

// ProtocolCompanionMethods preserves the difference between omitted and explicitly empty metadata.
type ProtocolCompanionMethods struct {
	methods []string
	present bool
}

func (methods *ProtocolCompanionMethods) UnmarshalJSON(data []byte) error {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return fmt.Errorf("protocol companions cannot be null")
	}
	var decoded []string
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	methods.methods = decoded
	methods.present = true
	return nil
}

func (methods ProtocolCompanionMethods) MarshalJSON() ([]byte, error) {
	return json.Marshal(methods.methods)
}

func (methods ProtocolCompanionMethods) IsZero() bool {
	return !methods.present
}

func (methods ProtocolCompanionMethods) Values() []string {
	return methods.methods
}

// ParityEvidence records the three independent proofs required before cutover.
type ParityEvidence struct {
	Response        []string `json:"response"`
	DatabaseChanges []string `json:"database_changes"`
	EmittedEffects  []string `json:"emitted_effects"`
}

// Load returns the inventory embedded in the Go backend.
func Load() (Inventory, error) {
	return Parse(embeddedInventory)
}

// LoadFile reads and validates an inventory file.
func LoadFile(path string) (Inventory, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Inventory{}, fmt.Errorf("read route inventory: %w", err)
	}
	return Parse(data)
}

// Parse decodes and validates one complete inventory document.
func Parse(data []byte) (Inventory, error) {
	var inventory Inventory
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&inventory); err != nil {
		return Inventory{}, fmt.Errorf("decode route inventory: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return Inventory{}, fmt.Errorf("decode route inventory: unexpected trailing JSON value")
		}
		return Inventory{}, fmt.Errorf("decode route inventory trailing data: %w", err)
	}
	if err := inventory.Validate(); err != nil {
		return Inventory{}, err
	}
	return inventory, nil
}

// Validate rejects incomplete, duplicate, or unsafe migration records.
func (inventory Inventory) Validate() error {
	if inventory.Version != 1 {
		return fmt.Errorf("route inventory version is %d; expected 1", inventory.Version)
	}
	if inventory.Source == "" {
		return fmt.Errorf("route inventory source is required")
	}
	if inventory.DefaultState != StateDjangoOnly {
		return fmt.Errorf("route inventory default state is %q; expected %q", inventory.DefaultState, StateDjangoOnly)
	}
	if len(inventory.AllowedStates) != len(allowedStates) {
		return fmt.Errorf("route inventory allowed states are incomplete")
	}
	seenStates := make(map[string]struct{}, len(inventory.AllowedStates))
	for _, state := range inventory.AllowedStates {
		if _, ok := allowedStates[state]; !ok {
			return fmt.Errorf("route inventory has unknown allowed state %q", state)
		}
		if _, ok := seenStates[state]; ok {
			return fmt.Errorf("route inventory repeats allowed state %q", state)
		}
		seenStates[state] = struct{}{}
	}

	seen := make(map[string]struct{}, len(inventory.Operations))
	pathOperationCounts := make(map[string]int)
	for index, operation := range inventory.Operations {
		if operation.Method == "" || operation.Path == "" || operation.OperationID == "" {
			return fmt.Errorf("route inventory operation %d is incomplete", index)
		}
		key := operation.Method + " " + operation.Path
		switch operation.Method {
		case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		default:
			return fmt.Errorf("route inventory operation %s has unsupported application method %q", key, operation.Method)
		}
		if _, ok := seen[key]; ok {
			return fmt.Errorf("route inventory repeats %s", key)
		}
		seen[key] = struct{}{}
		pathOperationCounts[operation.Path]++
		if _, ok := allowedStates[operation.State]; !ok {
			return fmt.Errorf("route inventory operation %s has unknown state %q", key, operation.State)
		}
		if operation.Method == http.MethodGet && operation.State != StateDjangoOnly && operation.ProtocolCompanions.IsZero() {
			return fmt.Errorf("route inventory operation %s is missing protocol companion metadata", key)
		}
		seenCompanions := make(map[string]struct{})
		for _, companion := range operation.ProtocolCompanions.Values() {
			switch companion {
			case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions, http.MethodTrace:
			default:
				return fmt.Errorf("route inventory operation %s has unsupported protocol companion %q", key, companion)
			}
			if companion == operation.Method {
				return fmt.Errorf("route inventory operation %s repeats its application method as protocol companion %q", key, companion)
			}
			if operation.Method == http.MethodGet && companion != http.MethodHead && companion != http.MethodOptions {
				return fmt.Errorf("route inventory operation %s has unsupported GET protocol companion %q", key, companion)
			}
			if _, exists := seenCompanions[companion]; exists {
				return fmt.Errorf("route inventory operation %s repeats protocol companion %q", key, companion)
			}
			seenCompanions[companion] = struct{}{}
		}
		seenAllowed := make(map[string]bool)
		for _, companion := range operation.AllowedProtocolCompanions.Values() {
			if _, exists := seenCompanions[companion]; !exists {
				return fmt.Errorf("route inventory operation %s advertises unrouted protocol companion %q", key, companion)
			}
			if seenAllowed[companion] {
				return fmt.Errorf("route inventory operation %s repeats allowed protocol companion %q", key, companion)
			}
			seenAllowed[companion] = true
		}
		if operation.State != StateDjangoOnly && (operation.Implementation == nil || *operation.Implementation == "") {
			return fmt.Errorf("route inventory operation %s has state %q without an implementation", key, operation.State)
		}
		if operation.State == StateGoParity || operation.State == StateGoActive || operation.State == StateDjangoRemoved {
			if !completeEvidence(operation.ParityEvidence) {
				return fmt.Errorf("route inventory operation %s has state %q without complete parity evidence", key, operation.State)
			}
		}
	}
	for _, operation := range inventory.Operations {
		ownerKey := operation.Method + " " + operation.Path
		for _, companion := range operation.ProtocolCompanions.Values() {
			companionKey := companion + " " + operation.Path
			if _, exists := seen[companionKey]; exists {
				return fmt.Errorf("route inventory operation %s has protocol companion %q owned by application operation %s", ownerKey, companion, companionKey)
			}
		}
		companions := operation.ProtocolCompanions.Values()
		// A retired writer may take OPTIONS from an earlier read family once
		// the entire path is routed to Go. Broader method companions still
		// require one family to own the complete resource contract.
		splitOptionsOwner := len(companions) == 1 && companions[0] == http.MethodOptions && splitGoOwnedPath(inventory, operation.Path)
		if operation.Method != http.MethodGet && len(companions) > 0 && pathOperationCounts[operation.Path] > 1 && !inventory.pathOwnedByGoFamily(operation) && !splitOptionsOwner {
			return fmt.Errorf("route inventory operation %s cannot declare protocol companions on a mixed application path", ownerKey)
		}
	}
	return nil
}

func completeEvidence(evidence *ParityEvidence) bool {
	return evidence != nil && len(evidence.Response) > 0 && len(evidence.DatabaseChanges) > 0 && len(evidence.EmittedEffects) > 0
}

// GoActive reports whether an operation may be routed to Go.
func (operation Operation) GoActive() bool {
	return operation.State == StateGoActive || operation.State == StateDjangoRemoved
}

// ActiveOperations returns the only operations the proxy may route to Go.
func (inventory Inventory) ActiveOperations() []Operation {
	active := make([]Operation, 0)
	for _, operation := range inventory.Operations {
		if operation.GoActive() {
			active = append(active, operation)
		}
	}
	sort.Slice(active, func(i, j int) bool {
		if active[i].Path == active[j].Path {
			return active[i].Method < active[j].Method
		}
		return active[i].Path < active[j].Path
	})
	return active
}

// State returns an operation state. Missing entries always use the Django-only default.
func (inventory Inventory) State(method, path string) string {
	for _, operation := range inventory.Operations {
		if operation.Method == method && operation.Path == path {
			return operation.State
		}
	}
	return StateDjangoOnly
}

// Non-GET protocol owners are safe on shared paths once every application
// method belongs to the same active family. A remaining Django writer keeps
// protocol ownership with Django.
func (inventory Inventory) pathOwnedByGoFamily(owner Operation) bool {
	if !owner.GoActive() || owner.Implementation == nil {
		return false
	}
	for _, operation := range inventory.Operations {
		if operation.Path == owner.Path && (!operation.GoActive() || operation.Implementation == nil || *operation.Implementation != *owner.Implementation) {
			return false
		}
	}
	return true
}
