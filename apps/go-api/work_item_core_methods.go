package systemapi

import (
	"context"
	"io"

	api "github.com/CaliLuke/go-plane/work_item_core/gen/work_item_core"
)

// Each authored adapter preserves the generated method signature and operation ID.
var _ api.Service = (*workItemCoreService)(nil)

func (s *workItemCoreService) GetAPIV1WorkspacesBySlugIssuesSearch(ctx context.Context, payload *api.GetAPIV1WorkspacesBySlugIssuesSearchPayload) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "get_api_v1_workspaces_by_slug_issues_search", payload, nil)
}

func (s *workItemCoreService) GetAPIV1WorkspacesBySlugIssuesProjectIdentifierIssueIdentifier(ctx context.Context, payload *api.GetAPIV1WorkspacesBySlugIssuesProjectIdentifierIssueIdentifierPayload) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "get_api_v1_workspaces_by_slug_issues__project_identifier___issue_identifier", payload, nil)
}

func (s *workItemCoreService) PostAPIV1WorkspacesBySlugProjectsByProjectIDCyclesByCycleIDCycleIssues(ctx context.Context, payload *api.PostAPIV1WorkspacesBySlugProjectsByProjectIDCyclesByCycleIDCycleIssuesPayload, requestBody io.ReadCloser) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "post_api_v1_workspaces_by_slug_projects_by_project_id_cycles_by_cycle_id_cycle_issues", payload, requestBody)
}

func (s *workItemCoreService) DeleteAPIV1WorkspacesBySlugProjectsByProjectIDCyclesByCycleIDCycleIssuesByIssueID(ctx context.Context, payload *api.DeleteAPIV1WorkspacesBySlugProjectsByProjectIDCyclesByCycleIDCycleIssuesByIssueIDPayload) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "delete_api_v1_workspaces_by_slug_projects_by_project_id_cycles_by_cycle_id_cycle_issues_by_issue_id", payload, nil)
}

func (s *workItemCoreService) PostAPIV1WorkspacesBySlugProjectsByProjectIDCyclesByCycleIDTransferIssues(ctx context.Context, payload *api.PostAPIV1WorkspacesBySlugProjectsByProjectIDCyclesByCycleIDTransferIssuesPayload, requestBody io.ReadCloser) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "post_api_v1_workspaces_by_slug_projects_by_project_id_cycles_by_cycle_id_transfer_issues", payload, requestBody)
}

func (s *workItemCoreService) GetAPIV1WorkspacesBySlugProjectsByProjectIDIntakeIssues(ctx context.Context, payload *api.GetAPIV1WorkspacesBySlugProjectsByProjectIDIntakeIssuesPayload) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "get_api_v1_workspaces_by_slug_projects_by_project_id_intake_issues", payload, nil)
}

func (s *workItemCoreService) PostAPIV1WorkspacesBySlugProjectsByProjectIDIntakeIssues(ctx context.Context, payload *api.PostAPIV1WorkspacesBySlugProjectsByProjectIDIntakeIssuesPayload, requestBody io.ReadCloser) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "post_api_v1_workspaces_by_slug_projects_by_project_id_intake_issues", payload, requestBody)
}

func (s *workItemCoreService) DeleteAPIV1WorkspacesBySlugProjectsByProjectIDIntakeIssuesByIssueID(ctx context.Context, payload *api.DeleteAPIV1WorkspacesBySlugProjectsByProjectIDIntakeIssuesByIssueIDPayload) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "delete_api_v1_workspaces_by_slug_projects_by_project_id_intake_issues_by_issue_id", payload, nil)
}

func (s *workItemCoreService) GetAPIV1WorkspacesBySlugProjectsByProjectIDIntakeIssuesByIssueID(ctx context.Context, payload *api.GetAPIV1WorkspacesBySlugProjectsByProjectIDIntakeIssuesByIssueIDPayload) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "get_api_v1_workspaces_by_slug_projects_by_project_id_intake_issues_by_issue_id", payload, nil)
}

func (s *workItemCoreService) PatchAPIV1WorkspacesBySlugProjectsByProjectIDIntakeIssuesByIssueID(ctx context.Context, payload *api.PatchAPIV1WorkspacesBySlugProjectsByProjectIDIntakeIssuesByIssueIDPayload, requestBody io.ReadCloser) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "patch_api_v1_workspaces_by_slug_projects_by_project_id_intake_issues_by_issue_id", payload, requestBody)
}

func (s *workItemCoreService) GetAPIV1WorkspacesBySlugProjectsByProjectIDIssues(ctx context.Context, payload *api.GetAPIV1WorkspacesBySlugProjectsByProjectIDIssuesPayload) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "get_api_v1_workspaces_by_slug_projects_by_project_id_issues", payload, nil)
}

func (s *workItemCoreService) PostAPIV1WorkspacesBySlugProjectsByProjectIDIssues(ctx context.Context, payload *api.PostAPIV1WorkspacesBySlugProjectsByProjectIDIssuesPayload, requestBody io.ReadCloser) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "post_api_v1_workspaces_by_slug_projects_by_project_id_issues", payload, requestBody)
}

func (s *workItemCoreService) DeleteAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByPk(ctx context.Context, payload *api.DeleteAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByPkPayload) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "delete_api_v1_workspaces_by_slug_projects_by_project_id_issues_by_pk", payload, nil)
}

func (s *workItemCoreService) GetAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByPk(ctx context.Context, payload *api.GetAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByPkPayload) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "get_api_v1_workspaces_by_slug_projects_by_project_id_issues_by_pk", payload, nil)
}

func (s *workItemCoreService) PatchAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByPk(ctx context.Context, payload *api.PatchAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByPkPayload, requestBody io.ReadCloser) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "patch_api_v1_workspaces_by_slug_projects_by_project_id_issues_by_pk", payload, requestBody)
}

func (s *workItemCoreService) PostAPIV1WorkspacesBySlugProjectsByProjectIDModulesByModuleIDModuleIssues(ctx context.Context, payload *api.PostAPIV1WorkspacesBySlugProjectsByProjectIDModulesByModuleIDModuleIssuesPayload, requestBody io.ReadCloser) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "post_api_v1_workspaces_by_slug_projects_by_project_id_modules_by_module_id_module_issues", payload, requestBody)
}

func (s *workItemCoreService) DeleteAPIV1WorkspacesBySlugProjectsByProjectIDModulesByModuleIDModuleIssuesByIssueID(ctx context.Context, payload *api.DeleteAPIV1WorkspacesBySlugProjectsByProjectIDModulesByModuleIDModuleIssuesByIssueIDPayload) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "delete_api_v1_workspaces_by_slug_projects_by_project_id_modules_by_module_id_module_issues_by_issue_id", payload, nil)
}

func (s *workItemCoreService) GetAPIV1WorkspacesBySlugProjectsByProjectIDWorkItems(ctx context.Context, payload *api.GetAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsPayload) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "get_api_v1_workspaces_by_slug_projects_by_project_id_work_items", payload, nil)
}

func (s *workItemCoreService) PostAPIV1WorkspacesBySlugProjectsByProjectIDWorkItems(ctx context.Context, payload *api.PostAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsPayload, requestBody io.ReadCloser) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "post_api_v1_workspaces_by_slug_projects_by_project_id_work_items", payload, requestBody)
}

func (s *workItemCoreService) GetAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDRelations(ctx context.Context, payload *api.GetAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDRelationsPayload) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "get_api_v1_workspaces_by_slug_projects_by_project_id_work_items_by_issue_id_relations", payload, nil)
}

func (s *workItemCoreService) PostAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDRelations(ctx context.Context, payload *api.PostAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDRelationsPayload, requestBody io.ReadCloser) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "post_api_v1_workspaces_by_slug_projects_by_project_id_work_items_by_issue_id_relations", payload, requestBody)
}

func (s *workItemCoreService) DeleteAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByPk(ctx context.Context, payload *api.DeleteAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByPkPayload) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "delete_api_v1_workspaces_by_slug_projects_by_project_id_work_items_by_pk", payload, nil)
}

func (s *workItemCoreService) GetAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByPk(ctx context.Context, payload *api.GetAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByPkPayload) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "get_api_v1_workspaces_by_slug_projects_by_project_id_work_items_by_pk", payload, nil)
}

func (s *workItemCoreService) PatchAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByPk(ctx context.Context, payload *api.PatchAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByPkPayload, requestBody io.ReadCloser) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "patch_api_v1_workspaces_by_slug_projects_by_project_id_work_items_by_pk", payload, requestBody)
}

func (s *workItemCoreService) GetAPIV1WorkspacesBySlugWorkItemsSearch(ctx context.Context, payload *api.GetAPIV1WorkspacesBySlugWorkItemsSearchPayload) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "get_api_v1_workspaces_by_slug_work_items_search", payload, nil)
}

func (s *workItemCoreService) GetAPIV1WorkspacesBySlugWorkItemsProjectIdentifierIssueIdentifier(ctx context.Context, payload *api.GetAPIV1WorkspacesBySlugWorkItemsProjectIdentifierIssueIdentifierPayload) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "get_api_v1_workspaces_by_slug_work_items__project_identifier___issue_identifier", payload, nil)
}

func (s *workItemCoreService) PostAPIWorkspacesBySlugDraftIssues(ctx context.Context, payload *api.PostAPIWorkspacesBySlugDraftIssuesPayload, requestBody io.ReadCloser) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "post_api_workspaces_by_slug_draft_issues", payload, requestBody)
}

func (s *workItemCoreService) DeleteAPIWorkspacesBySlugDraftIssuesByPk(ctx context.Context, payload *api.DeleteAPIWorkspacesBySlugDraftIssuesByPkPayload) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "delete_api_workspaces_by_slug_draft_issues_by_pk", payload, nil)
}

func (s *workItemCoreService) PatchAPIWorkspacesBySlugDraftIssuesByPk(ctx context.Context, payload *api.PatchAPIWorkspacesBySlugDraftIssuesByPkPayload, requestBody io.ReadCloser) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "patch_api_workspaces_by_slug_draft_issues_by_pk", payload, requestBody)
}

func (s *workItemCoreService) PostAPIWorkspacesBySlugDraftToIssueByDraftID(ctx context.Context, payload *api.PostAPIWorkspacesBySlugDraftToIssueByDraftIDPayload, requestBody io.ReadCloser) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "post_api_workspaces_by_slug_draft_to_issue_by_draft_id", payload, requestBody)
}

func (s *workItemCoreService) GetAPIWorkspacesBySlugIssues(ctx context.Context, payload *api.GetAPIWorkspacesBySlugIssuesPayload) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "get_api_workspaces_by_slug_issues", payload, nil)
}

func (s *workItemCoreService) GetAPIWorkspacesBySlugProjectsByProjectIDArchivedIssues(ctx context.Context, payload *api.GetAPIWorkspacesBySlugProjectsByProjectIDArchivedIssuesPayload) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "get_api_workspaces_by_slug_projects_by_project_id_archived_issues", payload, nil)
}

func (s *workItemCoreService) PostAPIWorkspacesBySlugProjectsByProjectIDBulkArchiveIssues(ctx context.Context, payload *api.PostAPIWorkspacesBySlugProjectsByProjectIDBulkArchiveIssuesPayload, requestBody io.ReadCloser) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "post_api_workspaces_by_slug_projects_by_project_id_bulk_archive_issues", payload, requestBody)
}

func (s *workItemCoreService) DeleteAPIWorkspacesBySlugProjectsByProjectIDBulkDeleteIssues(ctx context.Context, payload *api.DeleteAPIWorkspacesBySlugProjectsByProjectIDBulkDeleteIssuesPayload, requestBody io.ReadCloser) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "delete_api_workspaces_by_slug_projects_by_project_id_bulk_delete_issues", payload, requestBody)
}

func (s *workItemCoreService) PostAPIWorkspacesBySlugProjectsByProjectIDCyclesByCycleIDCycleIssues(ctx context.Context, payload *api.PostAPIWorkspacesBySlugProjectsByProjectIDCyclesByCycleIDCycleIssuesPayload, requestBody io.ReadCloser) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "post_api_workspaces_by_slug_projects_by_project_id_cycles_by_cycle_id_cycle_issues", payload, requestBody)
}

func (s *workItemCoreService) DeleteAPIWorkspacesBySlugProjectsByProjectIDCyclesByCycleIDCycleIssuesByIssueID(ctx context.Context, payload *api.DeleteAPIWorkspacesBySlugProjectsByProjectIDCyclesByCycleIDCycleIssuesByIssueIDPayload) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "delete_api_workspaces_by_slug_projects_by_project_id_cycles_by_cycle_id_cycle_issues_by_issue_id", payload, nil)
}

func (s *workItemCoreService) PatchAPIWorkspacesBySlugProjectsByProjectIDCyclesByCycleIDCycleIssuesByIssueID(ctx context.Context, payload *api.PatchAPIWorkspacesBySlugProjectsByProjectIDCyclesByCycleIDCycleIssuesByIssueIDPayload, requestBody io.ReadCloser) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "patch_api_workspaces_by_slug_projects_by_project_id_cycles_by_cycle_id_cycle_issues_by_issue_id", payload, requestBody)
}

func (s *workItemCoreService) PutAPIWorkspacesBySlugProjectsByProjectIDCyclesByCycleIDCycleIssuesByIssueID(ctx context.Context, payload *api.PutAPIWorkspacesBySlugProjectsByProjectIDCyclesByCycleIDCycleIssuesByIssueIDPayload, requestBody io.ReadCloser) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "put_api_workspaces_by_slug_projects_by_project_id_cycles_by_cycle_id_cycle_issues_by_issue_id", payload, requestBody)
}

func (s *workItemCoreService) PostAPIWorkspacesBySlugProjectsByProjectIDCyclesByCycleIDTransferIssues(ctx context.Context, payload *api.PostAPIWorkspacesBySlugProjectsByProjectIDCyclesByCycleIDTransferIssuesPayload, requestBody io.ReadCloser) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "post_api_workspaces_by_slug_projects_by_project_id_cycles_by_cycle_id_transfer_issues", payload, requestBody)
}

func (s *workItemCoreService) GetAPIWorkspacesBySlugProjectsByProjectIDDeletedIssues(ctx context.Context, payload *api.GetAPIWorkspacesBySlugProjectsByProjectIDDeletedIssuesPayload) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "get_api_workspaces_by_slug_projects_by_project_id_deleted_issues", payload, nil)
}

func (s *workItemCoreService) PostAPIWorkspacesBySlugProjectsByProjectIDInboxIssues(ctx context.Context, payload *api.PostAPIWorkspacesBySlugProjectsByProjectIDInboxIssuesPayload, requestBody io.ReadCloser) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "post_api_workspaces_by_slug_projects_by_project_id_inbox_issues", payload, requestBody)
}

func (s *workItemCoreService) DeleteAPIWorkspacesBySlugProjectsByProjectIDInboxIssuesByPk(ctx context.Context, payload *api.DeleteAPIWorkspacesBySlugProjectsByProjectIDInboxIssuesByPkPayload) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "delete_api_workspaces_by_slug_projects_by_project_id_inbox_issues_by_pk", payload, nil)
}

func (s *workItemCoreService) PatchAPIWorkspacesBySlugProjectsByProjectIDInboxIssuesByPk(ctx context.Context, payload *api.PatchAPIWorkspacesBySlugProjectsByProjectIDInboxIssuesByPkPayload, requestBody io.ReadCloser) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "patch_api_workspaces_by_slug_projects_by_project_id_inbox_issues_by_pk", payload, requestBody)
}

func (s *workItemCoreService) PostAPIWorkspacesBySlugProjectsByProjectIDIntakeIssues(ctx context.Context, payload *api.PostAPIWorkspacesBySlugProjectsByProjectIDIntakeIssuesPayload, requestBody io.ReadCloser) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "post_api_workspaces_by_slug_projects_by_project_id_intake_issues", payload, requestBody)
}

func (s *workItemCoreService) DeleteAPIWorkspacesBySlugProjectsByProjectIDIntakeIssuesByPk(ctx context.Context, payload *api.DeleteAPIWorkspacesBySlugProjectsByProjectIDIntakeIssuesByPkPayload) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "delete_api_workspaces_by_slug_projects_by_project_id_intake_issues_by_pk", payload, nil)
}

func (s *workItemCoreService) PatchAPIWorkspacesBySlugProjectsByProjectIDIntakeIssuesByPk(ctx context.Context, payload *api.PatchAPIWorkspacesBySlugProjectsByProjectIDIntakeIssuesByPkPayload, requestBody io.ReadCloser) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "patch_api_workspaces_by_slug_projects_by_project_id_intake_issues_by_pk", payload, requestBody)
}

func (s *workItemCoreService) PostAPIWorkspacesBySlugProjectsByProjectIDIssueDates(ctx context.Context, payload *api.PostAPIWorkspacesBySlugProjectsByProjectIDIssueDatesPayload, requestBody io.ReadCloser) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "post_api_workspaces_by_slug_projects_by_project_id_issue_dates", payload, requestBody)
}

func (s *workItemCoreService) GetAPIWorkspacesBySlugProjectsByProjectIDIssuesDetail(ctx context.Context, payload *api.GetAPIWorkspacesBySlugProjectsByProjectIDIssuesDetailPayload) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "get_api_workspaces_by_slug_projects_by_project_id_issues_detail", payload, nil)
}

func (s *workItemCoreService) GetAPIWorkspacesBySlugProjectsByProjectIDIssues(ctx context.Context, payload *api.GetAPIWorkspacesBySlugProjectsByProjectIDIssuesPayload) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "get_api_workspaces_by_slug_projects_by_project_id_issues", payload, nil)
}

func (s *workItemCoreService) PostAPIWorkspacesBySlugProjectsByProjectIDIssues(ctx context.Context, payload *api.PostAPIWorkspacesBySlugProjectsByProjectIDIssuesPayload, requestBody io.ReadCloser) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "post_api_workspaces_by_slug_projects_by_project_id_issues", payload, requestBody)
}

func (s *workItemCoreService) GetAPIWorkspacesBySlugProjectsByProjectIDIssuesList(ctx context.Context, payload *api.GetAPIWorkspacesBySlugProjectsByProjectIDIssuesListPayload) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "get_api_workspaces_by_slug_projects_by_project_id_issues_list", payload, nil)
}

func (s *workItemCoreService) GetAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueRelation(ctx context.Context, payload *api.GetAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueRelationPayload) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "get_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_issue_relation", payload, nil)
}

func (s *workItemCoreService) PostAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueRelation(ctx context.Context, payload *api.PostAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueRelationPayload, requestBody io.ReadCloser) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "post_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_issue_relation", payload, requestBody)
}

func (s *workItemCoreService) PostAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDModules(ctx context.Context, payload *api.PostAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDModulesPayload, requestBody io.ReadCloser) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "post_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_modules", payload, requestBody)
}

func (s *workItemCoreService) PostAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDRemoveRelation(ctx context.Context, payload *api.PostAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDRemoveRelationPayload, requestBody io.ReadCloser) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "post_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_remove_relation", payload, requestBody)
}

func (s *workItemCoreService) GetAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDSubIssues(ctx context.Context, payload *api.GetAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDSubIssuesPayload) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "get_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_sub_issues", payload, nil)
}

func (s *workItemCoreService) PostAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDSubIssues(ctx context.Context, payload *api.PostAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDSubIssuesPayload, requestBody io.ReadCloser) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "post_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_sub_issues", payload, requestBody)
}

func (s *workItemCoreService) DeleteAPIWorkspacesBySlugProjectsByProjectIDIssuesByPk(ctx context.Context, payload *api.DeleteAPIWorkspacesBySlugProjectsByProjectIDIssuesByPkPayload) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "delete_api_workspaces_by_slug_projects_by_project_id_issues_by_pk", payload, nil)
}

func (s *workItemCoreService) GetAPIWorkspacesBySlugProjectsByProjectIDIssuesByPk(ctx context.Context, payload *api.GetAPIWorkspacesBySlugProjectsByProjectIDIssuesByPkPayload) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "get_api_workspaces_by_slug_projects_by_project_id_issues_by_pk", payload, nil)
}

func (s *workItemCoreService) PatchAPIWorkspacesBySlugProjectsByProjectIDIssuesByPk(ctx context.Context, payload *api.PatchAPIWorkspacesBySlugProjectsByProjectIDIssuesByPkPayload, requestBody io.ReadCloser) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "patch_api_workspaces_by_slug_projects_by_project_id_issues_by_pk", payload, requestBody)
}

func (s *workItemCoreService) PutAPIWorkspacesBySlugProjectsByProjectIDIssuesByPk(ctx context.Context, payload *api.PutAPIWorkspacesBySlugProjectsByProjectIDIssuesByPkPayload, requestBody io.ReadCloser) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "put_api_workspaces_by_slug_projects_by_project_id_issues_by_pk", payload, requestBody)
}

func (s *workItemCoreService) DeleteAPIWorkspacesBySlugProjectsByProjectIDIssuesByPkArchive(ctx context.Context, payload *api.DeleteAPIWorkspacesBySlugProjectsByProjectIDIssuesByPkArchivePayload) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "delete_api_workspaces_by_slug_projects_by_project_id_issues_by_pk_archive", payload, nil)
}

func (s *workItemCoreService) GetAPIWorkspacesBySlugProjectsByProjectIDIssuesByPkArchive(ctx context.Context, payload *api.GetAPIWorkspacesBySlugProjectsByProjectIDIssuesByPkArchivePayload) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "get_api_workspaces_by_slug_projects_by_project_id_issues_by_pk_archive", payload, nil)
}

func (s *workItemCoreService) PostAPIWorkspacesBySlugProjectsByProjectIDIssuesByPkArchive(ctx context.Context, payload *api.PostAPIWorkspacesBySlugProjectsByProjectIDIssuesByPkArchivePayload, requestBody io.ReadCloser) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "post_api_workspaces_by_slug_projects_by_project_id_issues_by_pk_archive", payload, requestBody)
}

func (s *workItemCoreService) PostAPIWorkspacesBySlugProjectsByProjectIDModulesByModuleIDIssues(ctx context.Context, payload *api.PostAPIWorkspacesBySlugProjectsByProjectIDModulesByModuleIDIssuesPayload, requestBody io.ReadCloser) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "post_api_workspaces_by_slug_projects_by_project_id_modules_by_module_id_issues", payload, requestBody)
}

func (s *workItemCoreService) DeleteAPIWorkspacesBySlugProjectsByProjectIDModulesByModuleIDIssuesByIssueID(ctx context.Context, payload *api.DeleteAPIWorkspacesBySlugProjectsByProjectIDModulesByModuleIDIssuesByIssueIDPayload) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "delete_api_workspaces_by_slug_projects_by_project_id_modules_by_module_id_issues_by_issue_id", payload, nil)
}

func (s *workItemCoreService) PatchAPIWorkspacesBySlugProjectsByProjectIDModulesByModuleIDIssuesByIssueID(ctx context.Context, payload *api.PatchAPIWorkspacesBySlugProjectsByProjectIDModulesByModuleIDIssuesByIssueIDPayload, requestBody io.ReadCloser) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "patch_api_workspaces_by_slug_projects_by_project_id_modules_by_module_id_issues_by_issue_id", payload, requestBody)
}

func (s *workItemCoreService) PutAPIWorkspacesBySlugProjectsByProjectIDModulesByModuleIDIssuesByIssueID(ctx context.Context, payload *api.PutAPIWorkspacesBySlugProjectsByProjectIDModulesByModuleIDIssuesByIssueIDPayload, requestBody io.ReadCloser) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "put_api_workspaces_by_slug_projects_by_project_id_modules_by_module_id_issues_by_issue_id", payload, requestBody)
}

func (s *workItemCoreService) GetAPIWorkspacesBySlugProjectsByProjectIDSearchIssues(ctx context.Context, payload *api.GetAPIWorkspacesBySlugProjectsByProjectIDSearchIssuesPayload) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "get_api_workspaces_by_slug_projects_by_project_id_search_issues", payload, nil)
}

func (s *workItemCoreService) GetAPIWorkspacesBySlugProjectsByProjectIDV2Issues(ctx context.Context, payload *api.GetAPIWorkspacesBySlugProjectsByProjectIDV2IssuesPayload) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "get_api_workspaces_by_slug_projects_by_project_id_v2_issues", payload, nil)
}

func (s *workItemCoreService) GetAPIWorkspacesBySlugUserIssuesByUserID(ctx context.Context, payload *api.GetAPIWorkspacesBySlugUserIssuesByUserIDPayload) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "get_api_workspaces_by_slug_user_issues_by_user_id", payload, nil)
}

func (s *workItemCoreService) GetAPIWorkspacesBySlugWorkItemsProjectIdentifierIssueIdentifier(ctx context.Context, payload *api.GetAPIWorkspacesBySlugWorkItemsProjectIdentifierIssueIdentifierPayload) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.execute(ctx, "get_api_workspaces_by_slug_work_items__project_identifier___issue_identifier", payload, nil)
}
