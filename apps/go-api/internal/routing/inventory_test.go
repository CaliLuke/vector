package routing

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInventoryMatchesDjangoContractAndKeepsRemovedOperations(t *testing.T) {
	inventory, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	contractPath := filepath.Join("..", "..", "..", "..", "docs", "go-backend-replacement", "openapi", "full-server.openapi.json")
	data, err := os.ReadFile(contractPath)
	if err != nil {
		t.Fatalf("read OpenAPI contract: %v", err)
	}
	var contract struct {
		Paths map[string]map[string]struct {
			OperationID string `json:"operationId"`
		} `json:"paths"`
	}
	if err := json.Unmarshal(data, &contract); err != nil {
		t.Fatalf("decode OpenAPI contract: %v", err)
	}

	want := make(map[string]string)
	for path, pathItem := range contract.Paths {
		for method, operation := range pathItem {
			method = strings.ToUpper(method)
			if !isHTTPMethod(method) {
				continue
			}
			want[method+" "+path] = operation.OperationID
		}
	}
	// The pre-retirement contract is the fixed universe this migration works
	// through, so this is the one genuine constant here. Every other count in
	// this file is derived, because a snapshot has to be bumped by every
	// retirement and then reads as a regression in the next family.
	if len(inventory.Operations) != PreRetirementOperations {
		t.Fatalf(
			"inventory operation count = %d, want %d",
			len(inventory.Operations),
			PreRetirementOperations,
		)
	}
	contractOperations := len(want)
	removed := 0
	for _, operation := range inventory.Operations {
		key := operation.Method + " " + operation.Path
		if operation.State == StateDjangoRemoved {
			removed++
			if _, exists := want[key]; exists {
				t.Errorf("Django-removed operation %s is still in the Django contract", key)
			}
			continue
		}
		if want[key] != operation.OperationID {
			t.Errorf("inventory %s operation ID = %q, want %q", key, operation.OperationID, want[key])
		}
		delete(want, key)
	}
	if len(want) != 0 {
		t.Fatalf("inventory is missing %d OpenAPI operations", len(want))
	}
	// The loop above proved that every operation Django still serves has an
	// inventory entry, and that no retired operation is still served. Those two
	// facts make the counts an identity: whatever Django no longer serves is
	// exactly what the inventory records as removed.
	if removed != len(inventory.Operations)-contractOperations {
		t.Fatalf(
			"Django-removed operations = %d, want %d (inventory %d minus contract %d)",
			removed,
			len(inventory.Operations)-contractOperations,
			len(inventory.Operations),
			contractOperations,
		)
	}
}

func TestInventoryTracksRetiredDjangoRoutes(t *testing.T) {
	inventory, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	states := make(map[string]int)
	for _, operation := range inventory.Operations {
		states[operation.State]++
	}
	counted := states[StateDjangoOnly] + states[StateGoActive] + states[StateDjangoRemoved]
	if counted != len(inventory.Operations) {
		t.Fatalf(
			"route states cover %d operations, want %d: %#v",
			counted,
			len(inventory.Operations),
			states,
		)
	}
	// ActiveOperations is the proxy's only source of truth for what may reach
	// Go, so it must agree with the states rather than with a recorded number.
	routed := states[StateGoActive] + states[StateDjangoRemoved]
	if active := inventory.ActiveOperations(); len(active) != routed {
		t.Fatalf("routable operations = %d, want %d: %#v", len(active), routed, states)
	}
	if state := inventory.State("GET", "/not-in-the-inventory/"); state != StateDjangoOnly {
		t.Fatalf("missing route state = %q, want %q", state, StateDjangoOnly)
	}
}
func TestActivityReadsParityLedgerCoversEveryFixture(t *testing.T) {
	inventory, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	const fixtureDirectory = "apps/go-api/internal/parity/testdata/fixtures/work_item_activity_reads"
	fixturePaths, err := filepath.Glob(filepath.Join("..", "..", "..", "..", fixtureDirectory, "*.json"))
	if err != nil {
		t.Fatalf("list activity read fixtures: %v", err)
	}
	want := make(map[string]struct{}, len(fixturePaths))
	for _, path := range fixturePaths {
		want[filepath.ToSlash(filepath.Join(fixtureDirectory, filepath.Base(path)))] = struct{}{}
	}

	got := make(map[string]struct{}, len(want))
	for _, operation := range inventory.Operations {
		if operation.Implementation == nil || (*operation.Implementation != "activity_reads" && *operation.Implementation != "user_activity_reads") || operation.ParityEvidence == nil {
			continue
		}
		for _, evidence := range [][]string{
			operation.ParityEvidence.Response,
			operation.ParityEvidence.DatabaseChanges,
			operation.ParityEvidence.EmittedEffects,
		} {
			for _, path := range evidence {
				if strings.HasPrefix(path, fixtureDirectory+"/") {
					got[path] = struct{}{}
				}
			}
		}
	}

	for path := range want {
		if _, ok := got[path]; !ok {
			t.Errorf("activity read fixture is absent from the parity ledger: %s", path)
		}
	}
	for path := range got {
		if _, ok := want[path]; !ok {
			t.Errorf("activity read parity ledger references a missing fixture: %s", path)
		}
	}
}

func TestParityStatesRequireAllEvidenceKinds(t *testing.T) {
	inventory, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	operation := inventory.Operations[0]
	implementation := "example"
	operation.Implementation = &implementation
	operation.State = StateGoActive
	operation.ProtocolCompanions = companionMethods()
	operation.ParityEvidence = &ParityEvidence{Response: []string{"response"}, DatabaseChanges: []string{"database"}}
	inventory.Operations = []Operation{operation}
	if err := inventory.Validate(); err == nil || !strings.Contains(err.Error(), "without complete parity evidence") {
		t.Fatalf("Validate() error = %v, want incomplete parity evidence", err)
	}
	operation.ParityEvidence.EmittedEffects = []string{"effects"}
	inventory.Operations[0] = operation
	if err := inventory.Validate(); err != nil {
		t.Fatalf("Validate() with complete evidence: %v", err)
	}
}

func TestProtocolCompanionMetadataValidation(t *testing.T) {
	inventory, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	implementation := "example"
	base := inventory.Operations[0]
	base.Method = "GET"
	base.ProtocolCompanions = ProtocolCompanionMethods{}
	base.State = StateGoImplemented
	base.Implementation = &implementation

	unsupported := base
	unsupported.ProtocolCompanions = companionMethods("BREW")
	unsupportedGET := base
	unsupportedGET.ProtocolCompanions = companionMethods("TRACE")
	duplicate := base
	duplicate.ProtocolCompanions = companionMethods("HEAD", "HEAD")
	repeatsApplicationMethod := base
	repeatsApplicationMethod.ProtocolCompanions = companionMethods(base.Method)
	for _, testCase := range []struct {
		name        string
		operation   Operation
		wantInError string
	}{
		{name: "missing", operation: base, wantInError: "missing protocol companion metadata"},
		{name: "unsupported", operation: unsupported, wantInError: "unsupported protocol companion"},
		{name: "unsupported GET companion", operation: unsupportedGET, wantInError: "unsupported GET protocol companion"},
		{name: "duplicate", operation: duplicate, wantInError: "repeats protocol companion"},
		{name: "application method", operation: repeatsApplicationMethod, wantInError: "repeats its application method"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			inventory.Operations = []Operation{testCase.operation}
			if err := inventory.Validate(); err == nil || !strings.Contains(err.Error(), testCase.wantInError) {
				t.Fatalf("Validate() error = %v, want %q", err, testCase.wantInError)
			}
		})
	}

	nonGET := base
	nonGET.Method = "POST"
	nonGET.ProtocolCompanions = companionMethods("GET", "HEAD", "PUT", "PATCH", "DELETE", "OPTIONS", "TRACE")
	inventory.Operations = []Operation{nonGET}
	if err := inventory.Validate(); err != nil {
		t.Fatalf("Validate() with non-GET protocol baseline: %v", err)
	}
	ownedGET := base
	ownedGET.State = StateDjangoOnly
	ownedGET.Implementation = nil
	ownedGET.ProtocolCompanions = ProtocolCompanionMethods{}
	inventory.Operations = []Operation{nonGET, ownedGET}
	if err := inventory.Validate(); err == nil || !strings.Contains(err.Error(), "owned by application operation GET") {
		t.Fatalf("Validate() companion collision error = %v", err)
	}
	mixedPOST := nonGET
	mixedPOST.ProtocolCompanions = companionMethods("OPTIONS", "TRACE")
	ownedPUT := ownedGET
	ownedPUT.Method = "PUT"
	ownedPUT.OperationID = "owned-put"
	inventory.Operations = []Operation{mixedPOST, ownedPUT}
	if _, err := RenderCaddy(inventory); err == nil || !strings.Contains(err.Error(), "mixed application path") {
		t.Fatalf("RenderCaddy() mixed non-GET path error = %v", err)
	}

	// A coherent writer family owns OPTIONS when all methods cross together.
	mixedPOST.State = StateGoActive
	mixedPOST.ParityEvidence = &ParityEvidence{Response: []string{"response"}, DatabaseChanges: []string{"database"}, EmittedEffects: []string{"effects"}}
	ownedPUT.State = StateGoActive
	ownedPUT.Implementation = mixedPOST.Implementation
	ownedPUT.ParityEvidence = mixedPOST.ParityEvidence
	inventory.Operations = []Operation{mixedPOST, ownedPUT}
	if _, err := RenderCaddy(inventory); err != nil {
		t.Fatalf("complete family protocol owner rejected: %v", err)
	}
	otherFamily := "other"
	ownedPUT.Implementation = &otherFamily
	inventory.Operations = []Operation{mixedPOST, ownedPUT}
	if err := inventory.Validate(); err == nil {
		t.Fatal("cross-family protocol ownership accepted")
	}
	mixedPOST.ProtocolCompanions = companionMethods("OPTIONS")
	inventory.Operations = []Operation{mixedPOST, ownedPUT}
	if _, err := RenderCaddy(inventory); err != nil {
		t.Fatalf("OPTIONS owner on a fully Go-owned split path rejected: %v", err)
	}
	ownedPUT.State = StateDjangoOnly
	ownedPUT.Implementation = nil
	inventory.Operations = []Operation{mixedPOST, ownedPUT}
	if err := inventory.Validate(); err == nil {
		t.Fatal("OPTIONS owner accepted while another method remains in Django")
	}

	base.ProtocolCompanions = companionMethods()
	inventory.Operations = []Operation{base}
	if err := inventory.Validate(); err != nil {
		t.Fatalf("Validate() with explicit empty companions: %v", err)
	}

	base.State = StateDjangoOnly
	base.Implementation = nil
	base.ProtocolCompanions = ProtocolCompanionMethods{}
	inventory.Operations = []Operation{base}
	if err := inventory.Validate(); err != nil {
		t.Fatalf("Validate() with uncharacterized Django-only operation: %v", err)
	}
}

func TestProtocolCompanionMetadataJSONPresence(t *testing.T) {
	inventory, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	implementation := "example"
	operation := inventory.Operations[0]
	operation.State = StateGoImplemented
	operation.Implementation = &implementation
	operation.ProtocolCompanions = companionMethods()
	inventory.Operations = []Operation{operation}

	encoded, err := json.Marshal(inventory)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"protocol_companions":[]`) {
		t.Fatalf("explicit empty companion metadata was omitted: %s", encoded)
	}
	if _, err := Parse(encoded); err != nil {
		t.Fatalf("Parse() after explicit-empty round trip: %v", err)
	}

	nullNonGET := strings.Replace(string(encoded), `"method":"GET"`, `"method":"POST"`, 1)
	nullNonGET = strings.Replace(nullNonGET, `"protocol_companions":[]`, `"protocol_companions":null`, 1)
	if _, err := Parse([]byte(nullNonGET)); err == nil || !strings.Contains(err.Error(), "protocol companions cannot be null") {
		t.Fatalf("Parse() null non-GET metadata error = %v", err)
	}

	operation.State = StateDjangoOnly
	operation.Implementation = nil
	operation.ProtocolCompanions = ProtocolCompanionMethods{}
	inventory.Operations = []Operation{operation}
	encoded, err = json.Marshal(inventory)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "protocol_companions") {
		t.Fatalf("omitted companion metadata was serialized: %s", encoded)
	}
}

func TestParseRejectsTrailingJSON(t *testing.T) {
	inventory, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(inventory)
	if err != nil {
		t.Fatal(err)
	}
	encoded = append(encoded, []byte("{}")...)
	if _, err := Parse(encoded); err == nil || !strings.Contains(err.Error(), "unexpected trailing JSON value") {
		t.Fatalf("Parse() trailing value error = %v", err)
	}
}

func TestInventoryRejectsProtocolMethodsAsApplications(t *testing.T) {
	inventory, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	operation := inventory.Operations[0]
	for _, method := range []string{"HEAD", "OPTIONS", "TRACE", "BREW"} {
		t.Run(method, func(t *testing.T) {
			operation.Method = method
			inventory.Operations = []Operation{operation}
			if err := inventory.Validate(); err == nil || !strings.Contains(err.Error(), "unsupported application method") {
				t.Fatalf("Validate() error = %v", err)
			}
		})
	}
}
func companionMethods(methods ...string) ProtocolCompanionMethods {
	if methods == nil {
		methods = []string{}
	}
	return ProtocolCompanionMethods{methods: methods, present: true}
}

func isHTTPMethod(method string) bool {
	switch method {
	case "GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD", "TRACE":
		return true
	default:
		return false
	}
}

func TestAllowedProtocolCompanionsMustBeRoutedAndUnique(t *testing.T) {
	for _, methods := range [][]string{{"TRACE"}, {"HEAD", "HEAD"}} {
		inventory, err := Load()
		if err != nil {
			t.Fatal(err)
		}
		for i := range inventory.Operations {
			op := &inventory.Operations[i]
			if op.Implementation != nil && *op.Implementation == "api_cycle_reads" {
				op.AllowedProtocolCompanions = companionMethods(methods...)
				break
			}
		}
		if err := inventory.Validate(); err == nil {
			t.Fatalf("accepted invalid allowed companions %v", methods)
		}
	}
}
