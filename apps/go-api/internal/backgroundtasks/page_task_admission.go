package backgroundtasks

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// PageTaskAdmission combines the page version and page component-log
// dispatchers behind the narrow interface used by page lifecycle producers.
type PageTaskAdmission struct {
	transactions      *PageTransactionDispatcher
	versions          *VersionHistoryDispatcher
	descriptionAssets *DescriptionAssetCopyDispatcher
}

func NewPageTaskAdmission(transactions *PageTransactionDispatcher, versions *VersionHistoryDispatcher, copies ...*DescriptionAssetCopyDispatcher) (*PageTaskAdmission, error) {
	if transactions == nil || versions == nil {
		return nil, fmt.Errorf("page transaction and version dispatchers are required")
	}
	if len(copies) > 1 {
		return nil, fmt.Errorf("at most one description asset copy dispatcher may be configured")
	}
	var descriptionAssets *DescriptionAssetCopyDispatcher
	if len(copies) == 1 {
		descriptionAssets = copies[0]
	}
	return &PageTaskAdmission{transactions: transactions, versions: versions, descriptionAssets: descriptionAssets}, nil
}

func (admission *PageTaskAdmission) AdmitPageTransaction(ctx context.Context, eventID string, newHTML, oldHTML any, pageID string) error {
	if admission == nil || admission.transactions == nil {
		return fmt.Errorf("native page transaction admission is not configured")
	}
	input := PageTransactionInput{PageID: pageID, NewDescriptionHTML: newHTML, OldDescriptionHTML: oldHTML}
	if _, err := admission.transactions.Admit(ctx, eventID, input); err != nil {
		return fmt.Errorf("admit page transaction workflow: %w", err)
	}
	return nil
}

func (admission *PageTaskAdmission) AdmitPageVersion(ctx context.Context, eventID, pageID string, existing any, userID string) error {
	if admission == nil || admission.versions == nil {
		return fmt.Errorf("native page version admission is not configured")
	}
	if _, err := uuid.Parse(eventID); err != nil {
		return fmt.Errorf("page version operation ID must be a UUID: %w", err)
	}
	snapshot, err := pageSnapshotJSON(existing)
	if err != nil {
		return err
	}
	input := PageVersionInput{PageID: pageID, ExistingInstance: snapshot, UserID: userID}
	if _, err := admission.versions.AdmitPage(ctx, eventID, input); err != nil {
		return fmt.Errorf("admit page version workflow: %w", err)
	}
	return nil
}

func pageSnapshotJSON(value any) (string, error) {
	if text, ok := value.(string); ok {
		return text, nil
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("encode page version snapshot: %w", err)
	}
	return string(encoded), nil
}

func (admission *PageTaskAdmission) AdmitPageDuplicateAssets(ctx context.Context, eventID, entityName, entityID, projectID, slug, userID string) error {
	if admission == nil || admission.descriptionAssets == nil {
		return fmt.Errorf("native page description asset copy admission is not configured")
	}
	_, err := admission.descriptionAssets.Admit(ctx, DescriptionAssetCopyInput{
		EventID: eventID, EntityName: entityName, EntityID: entityID, ProjectID: projectID, WorkspaceSlug: slug, UserID: userID,
	})
	if err != nil {
		return fmt.Errorf("admit page description asset copy workflow: %w", err)
	}
	return nil
}
