package workspaceprojectreads

import (
	"context"
	"testing"

	"uuid"
)

type recentVisitTaskCapture struct {
	name   string
	kwargs map[string]any
}

func (capture *recentVisitTaskCapture) Task(_ context.Context, name string, _ []any, kwargs map[string]any) error {
	capture.name, capture.kwargs = name, kwargs
	return nil
}

type recentVisitAdmissionCapture struct{ values []string }

func (capture *recentVisitAdmissionCapture) AdmitRecentVisit(_ context.Context, eventID, slug, entity, entityID, userID, projectID string) error {
	capture.values = []string{eventID, slug, entity, entityID, userID, projectID}
	return nil
}

func TestProjectDetailRecentVisitUsesNativeAdmission(t *testing.T) {
	legacy := &recentVisitTaskCapture{}
	native := &recentVisitAdmissionCapture{}
	repository := &Repository{publisher: legacy, recentVisits: native}
	user, project := uuid.New(), uuid.New()
	if err := repository.admitRecentVisit(context.Background(), user, "demo", project); err != nil {
		t.Fatal(err)
	}
	if len(native.values) != 6 || native.values[0] == "" || native.values[1] != "demo" || native.values[2] != "project" || native.values[3] != project.String() || native.values[4] != user.String() || native.values[5] != project.String() {
		t.Fatalf("native visit arguments = %#v", native.values)
	}
	if legacy.name != "" {
		t.Fatalf("native admission also published legacy task %q", legacy.name)
	}
}

func TestProjectRecentVisitFallsBackToLegacyTask(t *testing.T) {
	legacy := &recentVisitTaskCapture{}
	repository := &Repository{publisher: legacy}
	user, project := uuid.New(), uuid.New()
	if err := repository.admitRecentVisit(context.Background(), user, "demo", project); err != nil {
		t.Fatal(err)
	}
	if legacy.name != "plane.bgtasks.recent_visited_task.recent_visited_task" || legacy.kwargs["entity_name"] != "project" || legacy.kwargs["entity_identifier"] != project.String() || legacy.kwargs["user_id"] != user.String() || legacy.kwargs["project_id"] != project.String() || legacy.kwargs["slug"] != "demo" {
		t.Fatalf("legacy recent visit = %q %#v", legacy.name, legacy.kwargs)
	}
}
