package backgroundtasks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const (
	issueTaskCeleryName                   = "plane.bgtasks.issue_version_sync.issue_task"
	processLogsCeleryName                 = "plane.bgtasks.logger_task.process_logs"
	scheduleIssueDescriptionVersionCelery = "plane.bgtasks.issue_description_version_sync.schedule_issue_description_version"
	syncIssueDescriptionVersionCelery     = "plane.bgtasks.issue_description_version_sync.sync_issue_description_version"
	scheduleIssueVersionCelery            = "plane.bgtasks.issue_version_sync.schedule_issue_version"
	syncIssueVersionCelery                = "plane.bgtasks.issue_version_sync.sync_issue_version"
)

type issueVersionTaskAdmitter interface {
	Admit(context.Context, string, IssueVersionTaskInput) (string, error)
}

type issueDescriptionSyncTaskAdmitter interface {
	Admit(context.Context, string, int, int, time.Duration) (string, error)
}

type issueVersionSyncTaskAdmitter interface {
	Admit(context.Context, string, int, int, time.Duration) (string, error)
}

type apiActivityLogTaskAdmitter interface {
	AdmitInput(context.Context, APIActivityLogInput) (string, error)
}

// VersionTaskPublisher replaces the two version-history backfills, the
// per-issue version task, and process_logs with durable workflow admissions.
// All other tasks and soft deletes retain the configured fallback behavior.
type VersionTaskPublisher struct {
	fallback        TaskFallback
	issueVersions   issueVersionTaskAdmitter
	issueSync       issueVersionSyncTaskAdmitter
	descriptionSync issueDescriptionSyncTaskAdmitter
	activityLog     apiActivityLogTaskAdmitter
}

func NewVersionTaskPublisher(
	fallback TaskFallback,
	issueVersions issueVersionTaskAdmitter,
	issueSync issueVersionSyncTaskAdmitter,
	descriptionSync issueDescriptionSyncTaskAdmitter,
	activityLog apiActivityLogTaskAdmitter,
) (*VersionTaskPublisher, error) {
	if fallback == nil || issueVersions == nil || issueSync == nil || descriptionSync == nil || activityLog == nil {
		return nil, errors.New("task fallback and all durable version/log admissions are required")
	}
	return &VersionTaskPublisher{
		fallback: fallback, issueVersions: issueVersions, issueSync: issueSync,
		descriptionSync: descriptionSync, activityLog: activityLog,
	}, nil
}

func (publisher *VersionTaskPublisher) Task(ctx context.Context, name string, args []any, kwargs map[string]any) error {
	if publisher == nil || publisher.fallback == nil {
		return errors.New("version task publisher is not configured")
	}
	switch name {
	case issueTaskCeleryName:
		input, err := issueVersionInputFromTask(args, kwargs)
		if err != nil {
			return err
		}
		_, err = publisher.issueVersions.Admit(ctx, uuid.NewString(), input)
		return err
	case processLogsCeleryName:
		input, err := apiActivityLogInputFromTask(kwargs)
		if err != nil {
			return err
		}
		_, err = publisher.activityLog.AdmitInput(ctx, input)
		return err
	case scheduleIssueDescriptionVersionCelery, syncIssueDescriptionVersionCelery:
		batchSize, offset, countdown, err := versionSyncOptions(args, kwargs, name == scheduleIssueDescriptionVersionCelery)
		if err != nil {
			return err
		}
		_, err = publisher.descriptionSync.Admit(ctx, uuid.NewString(), batchSize, offset, countdown)
		return err
	case scheduleIssueVersionCelery, syncIssueVersionCelery:
		batchSize, offset, countdown, err := versionSyncOptions(args, kwargs, name == scheduleIssueVersionCelery)
		if err != nil {
			return err
		}
		_, err = publisher.issueSync.Admit(ctx, uuid.NewString(), batchSize, offset, countdown)
		return err
	default:
		return publisher.fallback.Task(ctx, name, args, kwargs)
	}
}

func (publisher *VersionTaskPublisher) SoftDelete(ctx context.Context, table, id, field string) error {
	if publisher == nil || publisher.fallback == nil {
		return errors.New("version task publisher is not configured")
	}
	return publisher.fallback.SoftDelete(ctx, table, id, field)
}

func issueVersionInputFromTask(args []any, kwargs map[string]any) (IssueVersionTaskInput, error) {
	values := make(map[string]any, len(kwargs)+3)
	for key, value := range kwargs {
		values[key] = value
	}
	for i, key := range []string{"updated_issue", "issue_id", "user_id"} {
		if i < len(args) {
			if _, exists := values[key]; !exists {
				values[key] = args[i]
			}
		}
	}
	encoded, err := json.Marshal(values)
	if err != nil {
		return IssueVersionTaskInput{}, fmt.Errorf("encode issue version task payload: %w", err)
	}
	var input IssueVersionTaskInput
	if err := json.Unmarshal(encoded, &input); err != nil {
		return IssueVersionTaskInput{}, fmt.Errorf("decode issue version task payload: %w", err)
	}
	return input, nil
}

func apiActivityLogInputFromTask(kwargs map[string]any) (APIActivityLogInput, error) {
	data, ok := kwargs["log_data"].(map[string]any)
	if !ok {
		return APIActivityLogInput{}, fmt.Errorf("%w: process_logs requires log_data object", ErrPermanentAPIActivityLog)
	}
	encoded, err := json.Marshal(data)
	if err != nil {
		return APIActivityLogInput{}, fmt.Errorf("encode process_logs payload: %w", err)
	}
	var input APIActivityLogInput
	if err := json.Unmarshal(encoded, &input); err != nil {
		return APIActivityLogInput{}, fmt.Errorf("decode process_logs payload: %w", err)
	}
	input.ID = uuid.NewString()
	return input, nil
}

func versionSyncOptions(args []any, kwargs map[string]any, schedule bool) (batchSize, offset int, countdown time.Duration, err error) {
	defaults := []any{5000, 0, 300}
	keys := []string{"batch_size", "offset", "countdown"}
	values := make([]any, len(keys))
	for i, key := range keys {
		if value, ok := kwargs[key]; ok {
			values[i] = value
		} else {
			values[i] = defaults[i]
		}
	}
	if schedule {
		if _, ok := kwargs[keys[0]]; !ok && len(args) > 0 {
			values[0] = args[0]
		}
		if _, ok := kwargs[keys[2]]; !ok && len(args) > 1 {
			values[2] = args[1]
		}
		values[1] = 0
	} else {
		for i := range values {
			if _, ok := kwargs[keys[i]]; !ok && i < len(args) {
				values[i] = args[i]
			}
		}
	}
	if batchSize, err = taskInteger(values[0], "batch_size"); err != nil {
		return 0, 0, 0, err
	}
	if offset, err = taskInteger(values[1], "offset"); err != nil {
		return 0, 0, 0, err
	}
	seconds, parseErr := taskInteger(values[2], "countdown")
	if parseErr != nil {
		return 0, 0, 0, parseErr
	}
	countdown = time.Duration(seconds) * time.Second
	return batchSize, offset, countdown, nil
}

func taskInteger(value any, field string) (int, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return 0, fmt.Errorf("encode task %s: %w", field, err)
	}
	var result int
	if err := json.Unmarshal(encoded, &result); err != nil {
		return 0, fmt.Errorf("task %s must be an integer: %w", field, err)
	}
	return result, nil
}
