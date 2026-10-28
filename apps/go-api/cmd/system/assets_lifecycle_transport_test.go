package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	systemapi "github.com/CaliLuke/go-plane"
	activityapi "github.com/CaliLuke/go-plane/activityreads/gen/plane_internal_django_migration_api"
	assetslifecyclesvr "github.com/CaliLuke/go-plane/assetslifecycle/gen/http/plane_internal_django_migration_api/server"
	assetsapi "github.com/CaliLuke/go-plane/assetslifecycle/gen/plane_internal_django_migration_api"
	system "github.com/CaliLuke/go-plane/gen/system"
	assetdomain "github.com/CaliLuke/go-plane/internal/assets"
	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	publicprojectmetadatareadsstore "github.com/CaliLuke/go-plane/internal/publicprojectmetadatareads"
	notificationsapi "github.com/CaliLuke/go-plane/notificationreads/gen/notifications"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/google/uuid"
)

const (
	assetsTransportUserID      = "11111111-1111-4111-8111-111111111111"
	assetsTransportAssetID     = "22222222-2222-4222-8222-222222222222"
	assetsTransportWorkspaceID = "33333333-3333-4333-8333-333333333333"
)

type assetsTransportRepo struct {
	*assetdomain.PostgresRepository
	denyWorkspaceMember bool
	denyAssetParent     bool
	legacyAssets        []map[string]any
	permissionChecks    *int
	duplicateCreates    *int
	draftAssociation    func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, []uuid.UUID) error
}

type assetsTransportSession struct {
	*publicProjectMetadataThrottleStub
}

func (session *assetsTransportSession) InspectReadServiceSession(ctx context.Context, request *http.Request) (bool, bool, *djangosession.SessionRefresh, djangosession.Principal, error) {
	if _, err := request.Cookie("session-id"); err != nil {
		return false, false, nil, djangosession.Principal{}, nil
	}
	return false, false, nil, djangosession.Principal{UserID: session.userID, UserTimezone: session.userTimezone}, nil
}

func (session *assetsTransportSession) InspectReadServiceSessionWithoutTimezone(ctx context.Context, request *http.Request) (bool, bool, *djangosession.SessionRefresh, djangosession.Principal, error) {
	return session.InspectReadServiceSession(ctx, request)
}

func (repo assetsTransportRepo) BulkAssociateDraftAssets(ctx context.Context, workspace, actor, draft uuid.UUID, ids []uuid.UUID) error {
	return repo.draftAssociation(ctx, workspace, actor, draft, ids)
}

func (assetsTransportRepo) CreateUserUpload(context.Context, uuid.UUID, uuid.UUID, string, []byte, int64, string) error {
	return nil
}
func (assetsTransportRepo) AuthorizedWorkspaceID(context.Context, uuid.UUID, string) (uuid.UUID, error) {
	return uuid.MustParse(assetsTransportWorkspaceID), nil
}
func (assetsTransportRepo) WorkspaceIDBySlug(context.Context, string) (uuid.UUID, error) {
	return uuid.MustParse(assetsTransportWorkspaceID), nil
}
func (assetsTransportRepo) RelatedWorkspaceIDBySlug(context.Context, string) (uuid.UUID, error) {
	return uuid.MustParse(assetsTransportWorkspaceID), nil
}
func (repo assetsTransportRepo) WorkspaceAssetsByKey(context.Context, string, uuid.UUID, uuid.UUID) ([]map[string]any, error) {
	return repo.legacyAssets, nil
}
func (repo assetsTransportRepo) UserAssetsByKey(context.Context, uuid.UUID, string) ([]map[string]any, error) {
	return repo.legacyAssets, nil
}
func (assetsTransportRepo) UpdateAssetDeletedByKey(context.Context, string, *uuid.UUID, *uuid.UUID, uuid.UUID, bool) error {
	return nil
}
func (repo assetsTransportRepo) WorkspaceMemberBySlug(context.Context, uuid.UUID, string) (bool, error) {
	return !repo.denyWorkspaceMember, nil
}
func (repo assetsTransportRepo) WorkspacePermissionAllowed(context.Context, uuid.UUID, string) (bool, error) {
	if repo.permissionChecks != nil {
		(*repo.permissionChecks)++
	}
	return !repo.denyWorkspaceMember, nil
}
func (assetsTransportRepo) CreateWorkspaceUpload(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, string, []byte, int64, string, string, string) error {
	return nil
}
func (assetsTransportRepo) AssetByID(context.Context, uuid.UUID) (assetdomain.FileAsset, error) {
	return assetdomain.FileAsset{ID: assetsTransportAssetID, Asset: "workspace/avatar.png", EntityType: "USER_AVATAR", Attributes: map[string]any{"type": "image/png"}, IsUploaded: true}, nil
}
func (assetsTransportRepo) AssetByProjectAndID(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (assetdomain.FileAsset, error) {
	return assetdomain.FileAsset{ID: assetsTransportAssetID, Asset: "workspace/avatar.png", EntityType: "USER_AVATAR", Attributes: map[string]any{"type": "image/png"}, IsUploaded: true}, nil
}
func (assetsTransportRepo) DeleteProjectAsset(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) error {
	return nil
}
func (assetsTransportRepo) UpdateProjectAsset(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, []byte, bool) error {
	return nil
}
func (assetsTransportRepo) UserAssetByID(context.Context, uuid.UUID, uuid.UUID) (assetdomain.FileAsset, error) {
	return assetdomain.FileAsset{}, assetdomain.ErrNotFound
}
func (assetsTransportRepo) UpdateUserAsset(context.Context, uuid.UUID, uuid.UUID, []byte, bool, bool) error {
	return nil
}
func (assetsTransportRepo) UpdateLegacyUserAsset(context.Context, uuid.UUID, uuid.UUID, []byte, bool) error {
	return nil
}
func (assetsTransportRepo) DeleteUserAsset(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}
func (assetsTransportRepo) RestoreWorkspaceAsset(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) error {
	return nil
}
func (assetsTransportRepo) AssetByWorkspaceAndID(context.Context, uuid.UUID, uuid.UUID) (assetdomain.FileAsset, error) {
	return assetdomain.FileAsset{}, assetdomain.ErrNotFound
}
func (assetsTransportRepo) DeleteWorkspaceAsset(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) error {
	return nil
}
func (assetsTransportRepo) UpdateWorkspaceAssetUploaded(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, bool, bool) error {
	return nil
}
func (assetsTransportRepo) UpdateWorkspaceAsset(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, []byte, bool) (bool, error) {
	return true, nil
}
func (assetsTransportRepo) AssetByWorkspaceAndIDIncludingDeleted(context.Context, uuid.UUID, uuid.UUID) (assetdomain.FileAsset, error) {
	return assetdomain.FileAsset{}, assetdomain.ErrNotFound
}
func (assetsTransportRepo) WorkspaceAssetExists(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (bool, error) {
	return false, nil
}
func (assetsTransportRepo) AssetForDuplicate(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (assetdomain.FileAsset, error) {
	return assetdomain.FileAsset{}, assetdomain.ErrNotFound
}
func (assetsTransportRepo) ProjectBelongsToWorkspace(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return true, nil
}
func (assetsTransportRepo) WorkspaceMemberByID(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return true, nil
}
func (assetsTransportRepo) WorkspaceRole(context.Context, uuid.UUID, uuid.UUID) (int, error) {
	return 20, nil
}
func (repo assetsTransportRepo) AssetParentVisible(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (bool, error) {
	return !repo.denyAssetParent, nil
}
func (assetsTransportRepo) CreateWorkspaceEntityUpload(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, string, []byte, int64, string, string, bool) error {
	return nil
}
func (assetsTransportRepo) ProjectPermissionAllowed(context.Context, uuid.UUID, string, uuid.UUID) (bool, error) {
	return true, nil
}
func (assetsTransportRepo) ProjectMemberActive(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (bool, error) {
	return true, nil
}
func (assetsTransportRepo) LegacyRelationExists(context.Context, string, uuid.UUID) (bool, error) {
	return true, nil
}
func (repo assetsTransportRepo) CreateDuplicate(context.Context, assetdomain.FileAsset, uuid.UUID, uuid.UUID, uuid.UUID, string, string, string, string) error {
	if repo.duplicateCreates != nil {
		(*repo.duplicateCreates)++
	}
	return nil
}

func (assetsTransportRepo) MarkDuplicateUploaded(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) error {
	return nil
}
func (assetsTransportRepo) PublicProjectBoard(context.Context, string, bool) (uuid.UUID, uuid.UUID, string, bool, error) {
	return uuid.MustParse(assetsTransportWorkspaceID), uuid.MustParse("44444444-4444-4444-8444-444444444444"), "assets-transport", true, nil
}
func (assetsTransportRepo) PublicProjectAsset(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, bool, bool) (assetdomain.FileAsset, error) {
	return assetdomain.FileAsset{ID: assetsTransportAssetID, Asset: "workspace/avatar.png", EntityType: "USER_AVATAR", Attributes: map[string]any{"type": "image/png"}, IsUploaded: true}, nil
}
func (repo assetsTransportRepo) PublicOwnedProjectAsset(ctx context.Context, _ uuid.UUID, workspaceID, projectID, assetID uuid.UUID, includeDeleted bool) (assetdomain.FileAsset, error) {
	return repo.PublicProjectAsset(ctx, workspaceID, projectID, assetID, includeDeleted, false)
}
func (assetsTransportRepo) CreatePublicUpload(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, string, []byte, int64, string, string) error {
	return nil
}
func (assetsTransportRepo) UpdatePublicAsset(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, []byte) error {
	return nil
}
func (assetsTransportRepo) DeletePublicAsset(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) error {
	return nil
}
func (assetsTransportRepo) RestorePublicAsset(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) error {
	return nil
}
func (assetsTransportRepo) PublicBulkAssetRows(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, []uuid.UUID) ([]uuid.UUID, string, error) {
	return nil, "assets-transport", nil
}
func (assetsTransportRepo) BulkAssociatePublicAssetComments(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, []uuid.UUID) error {
	return nil
}

type assetsTransportKeys struct{}

func (assetsTransportKeys) Validate(_ context.Context, key string) (apikey.Principal, error) {
	if key != "assets-transport-valid" {
		return apikey.Principal{}, apikey.ErrInvalid
	}
	return apikey.Principal{UserID: uuid.MustParse(assetsTransportUserID), Timezone: "UTC"}, nil
}

type assetsTransportEffects struct{}

func (assetsTransportEffects) Task(context.Context, string, []any, map[string]any) error {
	return nil
}

func (assetsTransportEffects) Delete(context.Context, string) error {
	return nil
}

type assetsTransportThrottleCache struct {
	mu     sync.Mutex
	values map[string][]byte
	gets   int
	key    string
	getErr error
	setErr error
}

func (cache *assetsTransportThrottleCache) Get(_ context.Context, key string) ([]byte, bool, error) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	cache.gets++
	cache.key = key
	if cache.getErr != nil {
		return nil, false, cache.getErr
	}
	value, ok := cache.values[key]
	return value, ok, nil
}
func (cache *assetsTransportThrottleCache) Set(_ context.Context, key string, value []byte, _ time.Duration) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	cache.key = key
	if cache.setErr != nil {
		return cache.setErr
	}
	if cache.values == nil {
		cache.values = map[string][]byte{}
	}
	cache.values[key] = append([]byte(nil), value...)
	return nil
}

func assetsTransportHandler(t *testing.T) http.Handler {
	return assetsTransportHandlerWith(t, assetsTransportRepo{}, &assetsTransportThrottleCache{})
}

func assetsTransportHandlerWith(t *testing.T, repository assetsTransportRepo, cache *assetsTransportThrottleCache) http.Handler {
	return assetsTransportHandlerWithTimezone(t, repository, cache, "UTC")
}

func assetsTransportHandlerWithTimezone(t *testing.T, repository assetsTransportRepo, cache *assetsTransportThrottleCache, timezone string) http.Handler {
	t.Helper()
	throttle, err := apikey.NewThrottle(cache, "100/minute")
	if err != nil {
		t.Fatal(err)
	}
	duplicateThrottle, err := apikey.NewThrottle(cache, "5/minute")
	if err != nil {
		t.Fatal(err)
	}
	storage, err := assetdomain.NewStorage(assetdomain.StorageConfig{
		AccessKeyID: "test-access", SecretAccessKey: "test-secret", Bucket: "assets-transport", Region: "us-east-1",
		Endpoint: "http://minio.example.test:9000", PublicEndpoint: "http://minio.example.test:9000", PathStyle: true, SignedURLTTL: time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	domain, err := assetdomain.NewService(repository, storage, assetsTransportKeys{}, throttle, 5<<20, assetdomain.AssetEffects{
		Tasks: assetsTransportEffects{},
		Cache: assetsTransportEffects{},
	})
	if err != nil {
		t.Fatal(err)
	}
	lifecycle, err := assetdomain.NewLifecycleService(domain)
	if err != nil {
		t.Fatal(err)
	}
	session := &assetsTransportSession{publicProjectMetadataThrottleStub: &publicProjectMetadataThrottleStub{allowed: true, userID: assetsTransportUserID, userTimezone: timezone}}
	activity := &activityHTTPService{}
	logger := &publicProjectMetadataActivityStub{records: make(chan publicprojectmetadatareadsstore.APIActivityLog, 16)}
	return newHTTPHandlerWithPendingReplacements(
		newLogContext(context.Background(), io.Discard, false),
		system.NewEndpoints(systemapi.NewSystem()), false, nil, nil,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pendingReplacementEndpoints{
			assetsLifecycle:         assetsapi.NewEndpoints(lifecycle),
			assetsSessionMetadata:   lifecycle,
			assetsAPIMetadata:       lifecycle,
			assetsDuplicateThrottle: duplicateThrottle,
			notifications:           notificationsapi.NewEndpoints(&protocolNotificationsStub{}),
			notificationsMetadata:   sessionProtocolAuthorizerStub{allowed: true},
			activityReads:           activityapi.NewEndpoints(activity),
			activityReadsActivity:   logger,
			activityReadsSecret:     "assets-transport-secret",
			activityReadsBodyLimit:  5 << 20,
			readServicesSession:     session,
			readServicesCookie:      djangoSessionCookieConfig{Name: "session-id", Configured: true},
			readServicesCORS:        []string{"https://app.example.test"},
		})
}

func TestAssetsLifecycleGeneratedMountsOwnEveryContractOperation(t *testing.T) {
	root := assetsTransportRepoRoot(t)
	var manifest struct {
		OperationCount int `json:"operation_count"`
		Operations     map[string]struct {
			Method string `json:"method"`
			Path   string `json:"path"`
		} `json:"operations"`
	}
	readAssetsTransportJSON(t, filepath.Join(root, "docs/go-backend-replacement/assets-lifecycle-manifest.json"), &manifest)
	if manifest.OperationCount != 81 || len(manifest.Operations) != 81 {
		t.Fatalf("manifest has count=%d operations=%d; want 81", manifest.OperationCount, len(manifest.Operations))
	}
	server := assetslifecyclesvr.New(assetsapi.NewEndpoints(&assetdomain.LifecycleService{}), loomhttp.NewMuxer(), nil, nil, nil, nil)
	mounts := make(map[string]struct{}, len(server.Mounts))
	for _, mount := range server.Mounts {
		mounts[mount.Verb+" "+mount.Pattern] = struct{}{}
	}
	if _, ok := mounts["POST /api/assets/v2/workspaces/{slug}/{entity_id}/bulk/"]; !ok {
		t.Fatal("Vector draft association route is not mounted")
	}
	if len(mounts) != manifest.OperationCount+1 {
		t.Fatalf("generated server mounts %d distinct method/path operations, manifest has %d", len(mounts), manifest.OperationCount)
	}
	for id, operation := range manifest.Operations {
		if _, ok := mounts[operation.Method+" "+operation.Path]; !ok {
			t.Errorf("%s is not mounted by generated production service: %s %s", id, operation.Method, operation.Path)
		}
	}
}

func TestAssetsLifecycleLegacyAssetGetResponsesEncodePopulatedDataUnion(t *testing.T) {
	rows := assetsLifecycleFixtureAssetRows(t)
	cases := []struct {
		name string
		path string
	}{
		{name: "user", path: "/api/users/file-assets/avatar.png/"},
		{name: "workspace", path: "/api/workspaces/file-assets/" + assetsTransportWorkspaceID + "/logo.png/"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, tc.path, nil)
			request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid"})
			response := httptest.NewRecorder()
			assetsTransportHandlerWith(t, assetsTransportRepo{legacyAssets: rows}, &assetsTransportThrottleCache{}).ServeHTTP(response, request)
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, body = %s; want 200", response.Code, response.Body.String())
			}
			var body struct {
				Data   []map[string]any `json:"data"`
				Status bool             `json:"status"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode response body %q headers=%v: %v", response.Body.String(), response.Header(), err)
			}
			if !body.Status || len(body.Data) != 1 {
				t.Fatalf("response = %#v; want status=true and one serialized asset", body)
			}
		})
	}
}

func TestAssetsLifecycleLegacyWorkspaceGetOmitsInvisibleParentAssets(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/workspaces/file-assets/"+assetsTransportWorkspaceID+"/logo.png/", nil)
	request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid"})
	response := httptest.NewRecorder()
	assetsTransportHandlerWith(t, assetsTransportRepo{
		legacyAssets:    assetsLifecycleFixtureAssetRows(t),
		denyAssetParent: true,
	}, &assetsTransportThrottleCache{}).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s; want a legacy not-found response", response.Code, response.Body.String())
	}
	var body struct {
		Status bool `json:"status"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response body %q: %v", response.Body.String(), err)
	}
	if body.Status {
		t.Fatalf("invisible parent asset was returned: %s", response.Body.String())
	}
	if strings.Contains(response.Body.String(), "assets-fixture.png") || strings.Contains(response.Body.String(), assetsTransportAssetID) {
		t.Fatalf("response exposed the filtered asset: %s", response.Body.String())
	}
}

func TestAssetsLifecycleLegacyAssetGeneratedEncodersEmitJSONBody(t *testing.T) {
	rows := assetsLifecycleFixtureAssetRows(t)
	dataJSON, err := json.Marshal(map[string]any{"data": rows, "status": true})
	if err != nil {
		t.Fatal(err)
	}
	var userData assetsapi.GetAPIUsersFileAssetsByAssetKeyResponseData
	if err := json.Unmarshal(dataJSON, &userData); err != nil {
		t.Fatalf("decode fixture-backed user response: %v", err)
	}
	var workspaceData assetsapi.GetAPIWorkspacesFileAssetsByWorkspaceIDByAssetKeyResponseData
	if err := json.Unmarshal(dataJSON, &workspaceData); err != nil {
		t.Fatalf("decode fixture-backed workspace response: %v", err)
	}
	cases := []struct {
		name   string
		result any
	}{
		{
			name: "user",
			result: func() any {
				union := assetsapi.NewGetAPIUsersFileAssetsByAssetKeyResponseDataOrGetAPIUsersFileAssetsByAssetKeyResponseErrorGetAPIUsersFileAssetsByAssetKeyResponseData(&userData)
				return &union
			}(),
		},
		{
			name: "workspace",
			result: func() any {
				union := assetsapi.NewGetAPIWorkspacesFileAssetsByWorkspaceIDByAssetKeyResponseDataOrGetAPIWorkspacesFileAssetsByWorkspaceIDByAssetKeyResponseErrorGetAPIWorkspacesFileAssetsByWorkspaceIDByAssetKeyResponseData(&workspaceData)
				return &union
			}(),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			encodeJSON := func(_ context.Context, w http.ResponseWriter) loomhttp.Encoder {
				return loomhttp.EncodingFunc(func(value any) error { return json.NewEncoder(w).Encode(value) })
			}
			var encode func(context.Context, http.ResponseWriter, any) error
			switch tc.name {
			case "user":
				encode = assetslifecyclesvr.EncodeGetAPIUsersFileAssetsByAssetKeyResponse(encodeJSON)
			case "workspace":
				encode = assetslifecyclesvr.EncodeGetAPIWorkspacesFileAssetsByWorkspaceIDByAssetKeyResponse(encodeJSON)
			}
			if err := encode(context.Background(), response, tc.result); err != nil {
				t.Fatal(err)
			}
			var body struct {
				Data   []map[string]any `json:"data"`
				Status bool             `json:"status"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatalf("generated encoder body %q: %v", response.Body.String(), err)
			}
			if response.Code != http.StatusOK || !body.Status || len(body.Data) != 1 {
				t.Fatalf("generated response status=%d body=%s", response.Code, response.Body.String())
			}
		})
	}
}

func assetsLifecycleFixtureAssetRows(t *testing.T) []map[string]any {
	t.Helper()
	path := filepath.Join(assetsTransportRepoRoot(t), "apps", "go-api", "internal", "parity", "testdata", "fixtures", "assets_lifecycle", "legacy-workspace-file-asset-get.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for old, replacement := range map[string]string{
		"<uuid-1>":    assetsTransportWorkspaceID,
		"<uuid-2>":    assetsTransportAssetID,
		"<uuid-3>":    assetsTransportUserID,
		"<uuid-4>":    "44444444-4444-4444-8444-444444444444",
		"<uuid-5>":    "55555555-5555-4555-8555-555555555555",
		"<timestamp>": "2026-01-02T03:04:05Z",
	} {
		data = []byte(strings.ReplaceAll(string(data), old, replacement))
	}
	var fixture struct {
		Response struct {
			Body struct {
				Data []map[string]any `json:"data"`
			} `json:"body"`
		} `json:"response"`
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatalf("decode legacy workspace source fixture: %v", err)
	}
	if len(fixture.Response.Body.Data) != 1 {
		t.Fatalf("source fixture contains %d serialized assets, want one", len(fixture.Response.Body.Data))
	}
	return fixture.Response.Body.Data
}

func TestAssetsLifecycleDerivedHeadMountsOnlyEligibleGeneratedGets(t *testing.T) {
	const headPath = "/api/assets/v2/workspaces/{slug}/check/{asset_id}/"
	const invalidGetPath = "/api/assets/v2/workspaces/{slug}/"
	called := 0
	server := &assetslifecyclesvr.Server{
		Mounts: []*assetslifecyclesvr.MountPoint{
			{Method: "GetAPIAssetsV2WorkspacesBySlugCheckByAssetID", Verb: http.MethodGet, Pattern: headPath},
			{Method: "GetAPIAssetsV2WorkspacesBySlug", Verb: http.MethodGet, Pattern: invalidGetPath},
		},
		GetAPIAssetsV2WorkspacesBySlugCheckByAssetID: http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			called++
			response.Header().Set("X-Assets-Head-Test", "preserved")
			_, _ = io.WriteString(response, "get-response-body")
		}),
		GetAPIAssetsV2WorkspacesBySlug: http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			t.Error("GET-invalid collection route should not be used as a HEAD companion")
		}),
	}
	mux := loomhttp.NewMuxer()
	derived := mountAssetsLifecycleDerivedHeads(mux, server)
	if _, ok := derived[headPath]; !ok {
		t.Fatal("derived HEAD mount set omitted the eligible source-session GET")
	}
	if _, ok := derived[invalidGetPath]; ok {
		t.Fatal("derived HEAD mount set included a GET-invalid path")
	}
	routes := assetsLifecycleMountedRoutes(server)
	if _, ok := routes[assetsLifecycleMountedRoute{Method: http.MethodGet, Path: invalidGetPath}]; !ok {
		t.Fatal("generated mount map omitted the GET-invalid route")
	}
	if _, ok := routes[assetsLifecycleMountedRoute{Method: http.MethodHead, Path: headPath}]; ok {
		t.Fatal("generated operation map should describe only Loom-authored operations")
	}

	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodHead, "/api/assets/v2/workspaces/demo/check/00000000-0000-4000-8000-000000000001/", nil))
	if response.Code != http.StatusOK || response.Body.Len() != 0 || response.Header().Get("X-Assets-Head-Test") != "preserved" || called != 1 {
		t.Fatalf("derived HEAD response status=%d body=%q header=%q called=%d", response.Code, response.Body.String(), response.Header().Get("X-Assets-Head-Test"), called)
	}
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodHead, "/api/assets/v2/workspaces/demo/", nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("GET-invalid route HEAD status=%d body=%q; want unmapped path", response.Code, response.Body.String())
	}
}

func TestAssetsLifecycleNativeProtocolCoverageMatchesSourceMetadataAndMounts(t *testing.T) {
	type protocolMetadata struct {
		Allow         string `json:"allow"`
		SourceFixture string `json:"source_fixture"`
	}
	var source map[string]protocolMetadata
	if err := json.Unmarshal(assetsProtocolMetadataJSON, &source); err != nil {
		t.Fatal(err)
	}
	protocols := make(map[string]readServiceProtocol)
	for _, protocol := range assetsReadServiceProtocols() {
		protocols[protocol.pattern] = protocol
	}
	mux := loomhttp.NewMuxer()
	server := assetslifecyclesvr.New(assetsapi.NewEndpoints(&assetdomain.LifecycleService{}), mux, nil, nil, nil, nil)
	generated := assetsLifecycleMountedRoutes(server)
	derived := mountAssetsLifecycleDerivedHeads(mux, server)
	for path, metadata := range source {
		protocol, ok := protocols[path]
		if !ok {
			t.Errorf("source protocol path %s is absent from production session protocol routes", path)
			continue
		}
		if protocol.allow != metadata.Allow {
			t.Errorf("%s Allow=%q, source=%q", path, protocol.allow, metadata.Allow)
		}
		if readServiceMethodAllowed(metadata.Allow, http.MethodOptions) {
			if !strings.HasPrefix(path, "/api/v1/") && metadata.SourceFixture == "" {
				t.Errorf("native OPTIONS path %s has no Django source fixture", path)
			}
		}
		if readServiceMethodAllowed(metadata.Allow, http.MethodHead) {
			if metadata.SourceFixture == "" {
				t.Errorf("native HEAD path %s has no Django source fixture", path)
			}
			if _, ok := generated[assetsLifecycleMountedRoute{Method: http.MethodGet, Path: path}]; !ok {
				t.Errorf("native HEAD path %s has no generated GET handler", path)
			}
			if _, ok := derived[path]; !ok {
				t.Errorf("native HEAD path %s has no generated production companion", path)
			}
		}
	}
}

func TestAssetsLifecycleProductionHandlerRejectsDeclaredInvalidMethods(t *testing.T) {
	root := assetsTransportRepoRoot(t)
	var decision struct {
		IncludedOperationCount int `json:"included_operation_count"`
		Routes                 []struct {
			Path           string   `json:"path"`
			InvalidMethods []string `json:"invalid_methods"`
			TargetAllow    string   `json:"target_allow"`
		} `json:"routes"`
	}
	readAssetsTransportJSON(t, filepath.Join(root, "docs/go-backend-replacement/decisions/ASSET-001-invalid-methods.json"), &decision)
	var manifest struct {
		Operations map[string]struct {
			Method      string `json:"method"`
			Path        string `json:"path"`
			AuthProfile string `json:"auth_profile"`
		} `json:"operations"`
	}
	readAssetsTransportJSON(t, filepath.Join(root, "docs/go-backend-replacement/assets-lifecycle-manifest.json"), &manifest)
	if decision.IncludedOperationCount != 23 {
		t.Fatalf("invalid-method decision count=%d, want 23", decision.IncludedOperationCount)
	}
	handler := assetsTransportHandler(t)
	for _, route := range decision.Routes {
		path := assetsTransportConcretePath(route.Path)
		for _, method := range route.InvalidMethods {
			wantAllow := route.TargetAllow
			t.Run(method+" "+route.Path, func(t *testing.T) {
				request := httptest.NewRequest(method, path, nil)
				request.Host = "assets.example.test"
				// EntityAssetEndpoint allows anonymous requests only for GET;
				// DRF authenticates the other verbs before method dispatch.
				publicAnchorNeedsSession := strings.HasPrefix(route.Path, "/api/public/assets/v2/anchor/") && method != http.MethodGet
				if publicAnchorNeedsSession || assetsTransportNeedsSession(manifest.Operations, route.Path) {
					request.AddCookie(&http.Cookie{Name: "session-id", Value: "assets-transport-session"})
				} else if assetsTransportNeedsAPIKey(manifest.Operations, route.Path) {
					request.Header.Set("X-Api-Key", "assets-transport-valid")
				}
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, request)
				if response.Code != http.StatusMethodNotAllowed {
					t.Fatalf("status=%d body=%s, want 405", response.Code, response.Body.String())
				}
				if response.Header().Get("Allow") != wantAllow {
					t.Errorf("Allow=%q, want %q", response.Header().Get("Allow"), wantAllow)
				}
				if method == http.MethodHead {
					if response.Body.Len() != 0 {
						t.Errorf("HEAD body=%q, want empty", response.Body.String())
					}
				} else {
					var body map[string]string
					if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
						t.Fatalf("decode 405 response %q: %v", response.Body.String(), err)
					}
					wantDetail := fmt.Sprintf("Method %q not allowed.", method)
					if body["detail"] != wantDetail {
						t.Errorf("detail=%q, want %q", body["detail"], wantDetail)
					}
				}
				if response.Header().Get("X-Content-Type-Options") != "nosniff" || !strings.Contains(strings.ToLower(response.Header().Get("Vary")), "cookie") {
					t.Errorf("missing production assets protocol/security headers: %v", response.Header())
				}
			})
		}
	}
}

func TestAssetsLifecycleGeneratedStaticResponseUsesRedirectTransport(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/assets/v2/static/"+assetsTransportAssetID+"/", nil)
	request.Host = "assets.example.test"
	response := httptest.NewRecorder()
	assetsTransportHandler(t).ServeHTTP(response, request)
	if response.Code != http.StatusFound {
		t.Fatalf("static GET status=%d body=%s, want 302", response.Code, response.Body.String())
	}
	location := response.Header().Get("Location")
	if !strings.Contains(location, "minio.example.test:9000/assets-transport/workspace/avatar.png") || response.Body.Len() != 0 {
		t.Fatalf("static redirect location=%q body=%q", location, response.Body.String())
	}
}

func TestAssetsLifecycleNativeOptionsAndPublicHeadProtocol(t *testing.T) {
	handler := assetsTransportHandler(t)
	options := httptest.NewRecorder()
	handler.ServeHTTP(options, httptest.NewRequest(http.MethodOptions, "/api/assets/v2/static/"+assetsTransportAssetID+"/", nil))
	if options.Code != http.StatusOK || options.Header().Get("Allow") != "GET, HEAD, OPTIONS" {
		t.Fatalf("static OPTIONS status=%d Allow=%q body=%s", options.Code, options.Header().Get("Allow"), options.Body.String())
	}
	var metadata map[string]any
	if err := json.Unmarshal(options.Body.Bytes(), &metadata); err != nil || metadata["name"] != "Static File Asset Endpoint" {
		t.Fatalf("static OPTIONS metadata=%v err=%v", metadata, err)
	}

	protectedOptions := httptest.NewRecorder()
	handler.ServeHTTP(protectedOptions, httptest.NewRequest(http.MethodOptions, "/api/assets/v2/workspaces/assets-transport/", nil))
	if protectedOptions.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous workspace OPTIONS status=%d body=%s; want source 401", protectedOptions.Code, protectedOptions.Body.String())
	}
	protectedOptions = httptest.NewRecorder()
	protectedRequest := httptest.NewRequest(http.MethodOptions, "/api/assets/v2/workspaces/assets-transport/", nil)
	protectedRequest.AddCookie(&http.Cookie{Name: "session-id", Value: "assets-transport-session"})
	handler.ServeHTTP(protectedOptions, protectedRequest)
	if protectedOptions.Code != http.StatusOK || protectedOptions.Header().Get("Allow") != "POST, OPTIONS" {
		t.Fatalf("authenticated workspace OPTIONS status=%d Allow=%q body=%s", protectedOptions.Code, protectedOptions.Header().Get("Allow"), protectedOptions.Body.String())
	}

	head := httptest.NewRecorder()
	handler.ServeHTTP(head, httptest.NewRequest(http.MethodHead, "/api/assets/v2/static/"+assetsTransportAssetID+"/", nil))
	if head.Code != http.StatusFound || head.Body.Len() != 0 || !strings.Contains(head.Header().Get("Location"), "/assets-transport/workspace/avatar.png") {
		t.Fatalf("static anonymous HEAD status=%d body=%q Location=%q", head.Code, head.Body.String(), head.Header().Get("Location"))
	}

	publicHead := httptest.NewRecorder()
	handler.ServeHTTP(publicHead, httptest.NewRequest(http.MethodHead, "/api/public/assets/v2/anchor/example-anchor/"+assetsTransportAssetID+"/", nil))
	if publicHead.Code != http.StatusUnauthorized || publicHead.Body.Len() != 0 {
		t.Fatalf("public asset anonymous HEAD status=%d body=%q; want source 401 with an empty HEAD body", publicHead.Code, publicHead.Body.String())
	}
}

func TestAssetsLifecycleV1NativeMethodsAuthenticateAPIKeyBefore405(t *testing.T) {
	paths := []string{
		"/api/v1/assets/user-assets/",
		"/api/v1/workspaces/assets-transport/assets/",
	}
	for _, path := range paths {
		for _, method := range []string{http.MethodHead, http.MethodOptions} {
			t.Run(method+" "+path, func(t *testing.T) {
				for _, test := range []struct {
					name   string
					key    string
					status int
				}{
					{name: "missing key", status: http.StatusUnauthorized},
					{name: "invalid key", key: "assets-transport-invalid", status: http.StatusForbidden},
					{name: "valid key", key: "assets-transport-valid", status: http.StatusMethodNotAllowed},
				} {
					t.Run(test.name, func(t *testing.T) {
						request := httptest.NewRequest(method, path, nil)
						if test.key != "" {
							request.Header.Set("X-Api-Key", test.key)
						}
						response := httptest.NewRecorder()
						assetsTransportHandler(t).ServeHTTP(response, request)
						if response.Code != test.status {
							t.Fatalf("status=%d body=%s Allow=%q; want %d", response.Code, response.Body.String(), response.Header().Get("Allow"), test.status)
						}
						if test.status == http.StatusMethodNotAllowed && response.Header().Get("Allow") != "POST" {
							t.Fatalf("valid-key protocol response Allow=%q, want POST", response.Header().Get("Allow"))
						}
					})
				}
			})
		}
	}
}

func TestAssetsLifecycleGenericV1NonmemberIsDeniedBeforeThrottle(t *testing.T) {
	cache := &assetsTransportThrottleCache{}
	handler := assetsTransportHandlerWith(t, assetsTransportRepo{denyWorkspaceMember: true}, cache)
	request := httptest.NewRequest(http.MethodOptions, "/api/v1/workspaces/assets-transport/assets/", nil)
	request.Header.Set("X-Api-Key", "assets-transport-valid")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("nonmember OPTIONS status=%d body=%s; want 403", response.Code, response.Body.String())
	}
	cache.mu.Lock()
	gets := cache.gets
	cache.mu.Unlock()
	if gets != 0 {
		t.Fatalf("nonmember OPTIONS reached API-key throttle cache %d times; want zero", gets)
	}
}

func TestAssetsLifecycleDuplicateAssetThrottlePrecedesPermissionAndDispatch(t *testing.T) {
	cache := &assetsTransportThrottleCache{}
	permissionChecks := 0
	duplicateCreates := 0
	handler := assetsTransportHandlerWith(t, assetsTransportRepo{
		denyWorkspaceMember: true,
		permissionChecks:    &permissionChecks,
		duplicateCreates:    &duplicateCreates,
	}, cache)
	path := "/api/assets/v2/workspaces/assets-transport/duplicate-assets/" + strings.ToUpper(assetsTransportAssetID) + "/"

	unauthorized := httptest.NewRequest(http.MethodOptions, path, nil)
	unauthorizedResponse := httptest.NewRecorder()
	handler.ServeHTTP(unauthorizedResponse, unauthorized)
	if unauthorizedResponse.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous OPTIONS status=%d body=%s, want 401", unauthorizedResponse.Code, unauthorizedResponse.Body.String())
	}
	if got := cacheStats(cache).gets; got != 0 {
		t.Fatalf("anonymous request reached throttle cache %d times, want zero", got)
	}

	for index, method := range []string{http.MethodPost, http.MethodOptions, http.MethodPatch, http.MethodPost, http.MethodOptions} {
		request := httptest.NewRequest(method, path, nil)
		request.AddCookie(&http.Cookie{Name: "session-id", Value: "assets-transport-session"})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		wantStatus := http.StatusForbidden
		if method == http.MethodPatch {
			wantStatus = http.StatusMethodNotAllowed
		} else if method == http.MethodOptions {
			wantStatus = http.StatusOK
		}
		if response.Code != wantStatus {
			t.Fatalf("request %d %s status=%d body=%s, want %d", index+1, method, response.Code, response.Body.String(), wantStatus)
		}
	}
	if got := cacheStats(cache).gets; got != 5 {
		t.Fatalf("authenticated duplicate endpoint methods reached throttle cache %d times, want five", got)
	}
	cache.mu.Lock()
	key := cache.key
	cache.mu.Unlock()
	if key != ":1:throttle_asset_"+assetsTransportAssetID {
		t.Fatalf("asset bucket key=%q, want canonical UUID-scoped key", key)
	}
	if permissionChecks != 2 {
		t.Fatalf("workspace permission checks=%d, want only the two dispatched POST requests", permissionChecks)
	}
	if duplicateCreates != 0 {
		t.Fatalf("denied requests created %d duplicate assets", duplicateCreates)
	}

	blocked := httptest.NewRequest(http.MethodPatch, path, nil)
	blocked.AddCookie(&http.Cookie{Name: "session-id", Value: "assets-transport-session"})
	blockedResponse := httptest.NewRecorder()
	handler.ServeHTTP(blockedResponse, blocked)
	if blockedResponse.Code != http.StatusTooManyRequests || blockedResponse.Body.String() != `{"error_code":5900,"error_message":"RATE_LIMIT_EXCEEDED"}` {
		t.Fatalf("sixth request status=%d body=%s, want source 429", blockedResponse.Code, blockedResponse.Body.String())
	}
	if blockedResponse.Header().Get("Retry-After") == "" {
		t.Fatalf("throttled response has no Retry-After header: %v", blockedResponse.Header())
	}
	if permissionChecks != 2 || duplicateCreates != 0 {
		t.Fatalf("throttled request reached permission/effect: permission checks=%d duplicate creates=%d", permissionChecks, duplicateCreates)
	}

	beforePreflight := cacheStats(cache).gets
	preflight := httptest.NewRequest(http.MethodOptions, path, nil)
	preflight.Header.Set("Origin", "https://app.example.test")
	preflight.Header.Set("Access-Control-Request-Method", http.MethodPost)
	preflight.AddCookie(&http.Cookie{Name: "session-id", Value: "assets-transport-session"})
	preflightResponse := httptest.NewRecorder()
	handler.ServeHTTP(preflightResponse, preflight)
	if preflightResponse.Code >= http.StatusBadRequest || cacheStats(cache).gets != beforePreflight {
		t.Fatalf("CORS preflight status=%d body=%s throttle gets=%d, want successful bypass at %d", preflightResponse.Code, preflightResponse.Body.String(), cacheStats(cache).gets, beforePreflight)
	}
}

func cacheStats(cache *assetsTransportThrottleCache) struct{ gets int } {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	return struct{ gets int }{gets: cache.gets}
}

func TestAssetsLifecycleDuplicateAssetThrottleCountsBeforeTimezoneAndMapsCacheFailure(t *testing.T) {
	path := "/api/assets/v2/workspaces/assets-transport/duplicate-assets/" + assetsTransportAssetID + "/"
	cache := &assetsTransportThrottleCache{}
	handler := assetsTransportHandlerWithTimezone(t, assetsTransportRepo{}, cache, "Not/AZone")
	for attempt := 1; attempt <= 5; attempt++ {
		request := httptest.NewRequest(http.MethodPost, path, nil)
		request.AddCookie(&http.Cookie{Name: "session-id", Value: "assets-transport-session"})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest || response.Body.String() != `{"error":"The required key does not exist."}` {
			t.Fatalf("allowed-bucket attempt %d status=%d body=%s, want source invalid-timezone 400", attempt, response.Code, response.Body.String())
		}
	}
	blocked := httptest.NewRequest(http.MethodPost, path, nil)
	blocked.AddCookie(&http.Cookie{Name: "session-id", Value: "assets-transport-session"})
	blockedResponse := httptest.NewRecorder()
	handler.ServeHTTP(blockedResponse, blocked)
	if blockedResponse.Code != http.StatusTooManyRequests {
		t.Fatalf("exhausted-bucket invalid-timezone status=%d body=%s, want throttle 429 before timezone", blockedResponse.Code, blockedResponse.Body.String())
	}

	failedCache := &assetsTransportThrottleCache{getErr: errors.New("cache unavailable")}
	failedHandler := assetsTransportHandlerWith(t, assetsTransportRepo{}, failedCache)
	request := httptest.NewRequest(http.MethodPost, path, nil)
	request.AddCookie(&http.Cookie{Name: "session-id", Value: "assets-transport-session"})
	response := httptest.NewRecorder()
	failedHandler.ServeHTTP(response, request)
	if response.Code != http.StatusInternalServerError || !strings.Contains(response.Body.String(), "server error") {
		t.Fatalf("cache failure status=%d body=%s, want internal 500", response.Code, response.Body.String())
	}
}

func TestAssetsLifecycleProductionHandlerKeepsMalformedAndUnownedPathsOut(t *testing.T) {
	handler := assetsTransportHandler(t)
	for _, test := range []struct {
		path string
		want int
	}{
		{path: "/api/assets/v2/static//", want: http.StatusBadRequest},
		{path: "/api/assets/v2/static/not-a-uuid/extra/", want: http.StatusNotFound},
		{path: "/api/not-migrated/", want: http.StatusNotFound},
	} {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, test.path, nil)
		if test.path == "/api/assets/v2/static//" {
			request.AddCookie(&http.Cookie{Name: "session-id", Value: "assets-transport-session"})
		}
		handler.ServeHTTP(response, request)
		if response.Code != test.want {
			t.Errorf("GET %s status=%d body=%s, want %d", test.path, response.Code, response.Body.String(), test.want)
		}
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/demo/assets/"+assetsTransportAssetID+"/", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("protected API-key route without credential status=%d body=%s, want 401", response.Code, response.Body.String())
	}
}

func TestAssetsLifecyclePublicAnchorKeepsAuthenticationPrecedence(t *testing.T) {
	for _, requestCase := range []struct{ method, path string }{
		{http.MethodPatch, "/api/public/assets/v2/anchor/example-anchor/"},
		{http.MethodDelete, "/api/public/assets/v2/anchor/example-anchor/"},
	} {
		t.Run(requestCase.method+" "+requestCase.path, func(t *testing.T) {
			request := httptest.NewRequest(requestCase.method, requestCase.path, nil)
			request.Host = "assets.example.test"
			response := httptest.NewRecorder()
			assetsTransportHandler(t).ServeHTTP(response, request)
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("anonymous status=%d body=%s, want 401", response.Code, response.Body.String())
			}
		})
	}
}

func assetsTransportNeedsAPIKey(operations map[string]struct {
	Method      string `json:"method"`
	Path        string `json:"path"`
	AuthProfile string `json:"auth_profile"`
}, path string) bool {
	for _, operation := range operations {
		if operation.Path == path && strings.Contains(operation.AuthProfile, "api-key") {
			return true
		}
	}
	return false
}

func assetsTransportNeedsSession(operations map[string]struct {
	Method      string `json:"method"`
	Path        string `json:"path"`
	AuthProfile string `json:"auth_profile"`
}, path string) bool {
	for _, operation := range operations {
		if operation.Path == path && operation.AuthProfile == "session" {
			return true
		}
	}
	return false
}

func assetsTransportConcretePath(path string) string {
	values := map[string]string{
		"slug": "assets-transport", "project_id": "44444444-4444-4444-8444-444444444444", "issue_id": "55555555-5555-4555-8555-555555555555",
		"asset_id": assetsTransportAssetID, "pk": assetsTransportAssetID, "anchor": "assets-transport-board", "asset_key": "assets-transport-key",
		"workspace_id": assetsTransportWorkspaceID, "entity_id": "66666666-6666-4666-8666-666666666666",
	}
	for key, value := range values {
		path = strings.ReplaceAll(path, "{"+key+"}", value)
	}
	return path
}

func assetsTransportRepoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate assets lifecycle transport test")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../../../../"))
}

func readAssetsTransportJSON(t *testing.T, path string, destination any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, destination); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
}
