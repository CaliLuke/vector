package pagereads

import (
	"context"
	"testing"

	"uuid"
)

type recentVisitCapture struct {
	args []any
}

func (capture *recentVisitCapture) Task(_ context.Context, name string, args []any, kwargs map[string]any) error {
	capture.args = []any{name, args, kwargs}
	return nil
}
func (*recentVisitCapture) SoftDelete(context.Context, string, string, string) error { return nil }

type recentVisitAdmissionCapture struct{ values []string }

func (capture *recentVisitAdmissionCapture) AdmitRecentVisit(_ context.Context, eventID, slug, entity, entityID, userID, projectID string) error {
	capture.values = []string{eventID, slug, entity, entityID, userID, projectID}
	return nil
}

func TestPageDetailRecentVisitUsesNativeAdmission(t *testing.T) {
	legacy := &recentVisitCapture{}
	native := &recentVisitAdmissionCapture{}
	repository := &Repository{publisher: legacy, recentVisits: native}
	user, project, page := uuid.New(), uuid.New(), uuid.New()
	if err := repository.admitRecentVisit(context.Background(), user, "demo", project, page); err != nil {
		t.Fatal(err)
	}
	if len(native.values) != 6 || native.values[0] == "" || native.values[1] != "demo" || native.values[2] != "page" || native.values[3] != page.String() || native.values[4] != user.String() || native.values[5] != project.String() {
		t.Fatalf("native visit arguments = %#v", native.values)
	}
	if len(legacy.args) != 0 {
		t.Fatalf("native admission also published legacy task: %#v", legacy.args)
	}
}

func TestPageRecentVisitFallsBackToLegacyTaskWhenNativeAdapterIsUnset(t *testing.T) {
	legacy := &recentVisitCapture{}
	repository := &Repository{publisher: legacy}
	user, project, page := uuid.New(), uuid.New(), uuid.New()
	if err := repository.admitRecentVisit(context.Background(), user, "demo", project, page); err != nil {
		t.Fatal(err)
	}
	if len(legacy.args) != 3 || legacy.args[0] != "plane.bgtasks.recent_visited_task.recent_visited_task" {
		t.Fatalf("legacy recent visit call = %#v", legacy.args)
	}
	kwargs := legacy.args[2].(map[string]any)
	if kwargs["slug"] != "demo" || kwargs["entity_name"] != "page" || kwargs["entity_identifier"] != page.String() || kwargs["user_id"] != user.String() || kwargs["project_id"] != project.String() {
		t.Fatalf("legacy recent visit payload = %#v", kwargs)
	}
}
