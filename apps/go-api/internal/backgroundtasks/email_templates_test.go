package backgroundtasks

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestPlaneEmailTemplatesRenderAndStripMarkup(t *testing.T) {
	values := map[string]string{
		"email": "person@example.test", "first_name": "Taylor", "forgot_password_url": "https://plane.test/reset",
		"code": "123456", "profile_url": "https://plane.test/profile", "login_url": "https://plane.test/login",
		"project_name": "Project A", "workspace_name": "Workspace A", "project_url": "https://plane.test/project",
		"message":            "Webhook was disabled",
		"inviter_first_name": "Casey", "abs_url": "https://plane.test/invite", "invitation_url": "https://plane.test/project-invite",
		"current_site": "https://plane.test", "mirror": "", "unsubscribe": "",
	}
	for name, expected := range map[string]string{
		"auth/forgot_password.html":             "https://plane.test/reset",
		"auth/magic_signin.html":                "123456",
		"user/email_updated.html":               "person@example.test",
		"user/user_activation.html":             "https://plane.test/profile",
		"user/user_deactivation.html":           "https://plane.test/login",
		"notifications/project_addition.html":   "Project A",
		"notifications/webhook-deactivate.html": "Webhook was disabled",
		"invitations/workspace_invitation.html": "Workspace A",
		"invitations/project_invitation.html":   "Taylor",
	} {
		t.Run(name, func(t *testing.T) {
			htmlBody, textBody, err := renderPlaneEmail(name, values)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(htmlBody, expected) || strings.TrimSpace(textBody) == "" {
				t.Fatalf("rendered email omitted expected context value %q", expected)
			}
			if strings.Contains(textBody, "<html") || strings.Contains(textBody, "font-family") {
				t.Fatal("plain text email retained markup or CSS")
			}
			if strings.Contains(htmlBody, "{{") {
				t.Fatal("rendered HTML retained a Django template placeholder")
			}
		})
	}
}

type templatedEmailFallbackSpy struct {
	taskNames   []string
	softDeletes int
}

func (spy *templatedEmailFallbackSpy) Task(_ context.Context, name string, _ []any, _ map[string]any) error {
	spy.taskNames = append(spy.taskNames, name)
	return nil
}
func (spy *templatedEmailFallbackSpy) SoftDelete(context.Context, string, string, string) error {
	spy.softDeletes++
	return nil
}

type templatedEmailAdmissionSpy struct {
	events []string
	names  []string
	args   [][]any
	err    error
}

func (spy *templatedEmailAdmissionSpy) AdmitTemplatedEmailTask(_ context.Context, eventID, name string, args []any) error {
	spy.events = append(spy.events, eventID)
	spy.names = append(spy.names, name)
	spy.args = append(spy.args, args)
	return spy.err
}

func TestTemplatedEmailTaskPublisherRoutesKnownTasksAndPreservesFallback(t *testing.T) {
	fallback := &templatedEmailFallbackSpy{}
	admission := &templatedEmailAdmissionSpy{}
	publisher, err := NewTemplatedEmailTaskPublisher(fallback, admission)
	if err != nil {
		t.Fatal(err)
	}
	args := []any{"a@example.test", "123456"}
	if err := publisher.Task(context.Background(), emailUpdateMagicCodeTask, args, nil); err != nil {
		t.Fatal(err)
	}
	if err := publisher.Task(context.Background(), emailUpdateMagicCodeTask, args, nil); err != nil {
		t.Fatal(err)
	}
	if len(admission.events) != 2 || admission.events[0] == admission.events[1] {
		t.Fatalf("two legitimate identical mail operations did not get distinct event identities: %v", admission.events)
	}
	for _, eventID := range admission.events {
		if _, err := uuid.Parse(eventID); err != nil {
			t.Fatalf("event ID %q is not a UUID: %v", eventID, err)
		}
	}
	if len(fallback.taskNames) != 0 || admission.names[0] != emailUpdateMagicCodeTask || len(admission.args[0]) != 2 {
		t.Fatalf("native email admission = %#v; fallback = %#v", admission, fallback)
	}
	if err := publisher.Task(context.Background(), "plane.bgtasks.unrelated", nil, nil); err != nil {
		t.Fatal(err)
	}
	if len(fallback.taskNames) != 1 || fallback.taskNames[0] != "plane.bgtasks.unrelated" {
		t.Fatalf("fallback task names = %v", fallback.taskNames)
	}
	if err := publisher.SoftDelete(context.Background(), "users", "id", "deleted_at"); err != nil {
		t.Fatal(err)
	}
	if fallback.softDeletes != 1 {
		t.Fatalf("fallback soft deletes = %d, want 1", fallback.softDeletes)
	}
}

func TestTemplatedEmailTaskPublisherPropagatesAdmissionErrors(t *testing.T) {
	wantErr := errors.New("durable admission failed")
	fallback := &templatedEmailFallbackSpy{}
	admission := &templatedEmailAdmissionSpy{err: wantErr}
	publisher, err := NewTemplatedEmailTaskPublisher(fallback, admission)
	if err != nil {
		t.Fatal(err)
	}
	if err := publisher.Task(context.Background(), magicLinkTask, []any{"a@example.test", "key", "123456"}, nil); !errors.Is(err, wantErr) {
		t.Fatalf("native admission error = %v, want %v", err, wantErr)
	}
	if len(fallback.taskNames) != 0 {
		t.Fatalf("failed native admission unexpectedly used fallback: %v", fallback.taskNames)
	}
}

func TestPlaneEmailTemplateEscapesUntrustedValues(t *testing.T) {
	htmlBody, _, err := renderPlaneEmail("auth/magic_signin.html", map[string]string{"code": "<script>alert(1)</script>", "email": "a&b@example.test"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(htmlBody, "<script>") || !strings.Contains(htmlBody, "&lt;script&gt;") || !strings.Contains(htmlBody, "a&amp;b@example.test") {
		t.Fatalf("untrusted template values were not HTML escaped: contains script=%t html=%q", strings.Contains(htmlBody, "<script>"), htmlBody)
	}
}
