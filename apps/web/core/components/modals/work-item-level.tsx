/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams, useRouter } from "@tanstack/react-router";

// plane imports
import type { TIssue } from "@plane/types";
import { EIssueServiceType, EIssuesStoreType } from "@plane/types";
// components
import { BulkDeleteIssuesModal } from "@/components/core/modals/bulk-delete-issues-modal";
import { DeleteIssueModal } from "@/components/issues/delete-issue-modal";
import { CreateUpdateIssueModal } from "@/components/issues/issue-modal/modal";
// hooks
import { useCommandPalette } from "@/hooks/use-command-palette";

import { useIssuesActions } from "@/hooks/use-issues-actions";
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";
import { useWorkItemByIdentifier, useWorkItemSubItems } from "@/hooks/use-work-item-detail";

export type TWorkItemLevelModalsProps = {
  workItemIdentifier: string | undefined;
};

export function WorkItemLevelModals(props: TWorkItemLevelModalsProps) {
  const { workItemIdentifier } = props;
  // router
  const { workspaceSlug, cycleId, moduleId } = useParams({ strict: false });
  const router = useRouter();
  // store hooks
  const projects = useWorkspaceProjectDetails(workspaceSlug?.toString());
  const [projectIdentifier, sequenceId] = (workItemIdentifier ?? "").split("-");
  const identifierProjectId = projects.projects?.find((project) => project.identifier === projectIdentifier)?.id;
  const workItemQuery = useWorkItemByIdentifier(
    workspaceSlug?.toString(),
    projectIdentifier,
    sequenceId,
    identifierProjectId
  );
  const epicQuery = useWorkItemByIdentifier(
    workspaceSlug?.toString(),
    projectIdentifier,
    sequenceId,
    identifierProjectId,
    EIssueServiceType.EPICS
  );
  const workItemDetails = workItemQuery.issue ?? epicQuery.issue;

  const { removeIssue: removeEpic } = useIssuesActions(EIssuesStoreType.EPIC);
  const { removeIssue: removeWorkItem } = useIssuesActions(EIssuesStoreType.PROJECT);

  const {
    isCreateIssueModalOpen,
    toggleCreateIssueModal,
    isDeleteIssueModalOpen,
    toggleDeleteIssueModal,
    isBulkDeleteIssueModalOpen,
    toggleBulkDeleteIssueModal,
    createWorkItemAllowedProjectIds,
  } = useCommandPalette();
  // derived values
  const parentSubItems = useWorkItemSubItems(
    workspaceSlug?.toString(),
    workItemDetails?.project_id ?? undefined,
    workItemDetails?.id,
    workItemDetails?.is_epic ? EIssueServiceType.EPICS : EIssueServiceType.ISSUES
  );

  const handleCreateIssueSubmit = async (newIssue: TIssue) => {
    if (!isCreatedSubItem(newIssue, workspaceSlug, workItemDetails?.id)) return;

    await parentSubItems.query.refetch();
  };
  const deleteIssue = async (projectId: string, issueId: string) =>
    deleteWorkItem({
      workspaceSlug: workspaceSlug?.toString() ?? "",
      projectId,
      issueId,
      isEpic: workItemDetails?.is_epic ?? false,
      removeEpic,
      removeWorkItem,
      navigate: (href) => router.navigate({ href }),
    });

  return (
    <>
      <CreateUpdateIssueModal
        isOpen={isCreateIssueModalOpen}
        onClose={() => toggleCreateIssueModal(false)}
        data={createIssueModalData(cycleId, moduleId)}
        onSubmit={handleCreateIssueSubmit}
        allowedProjectIds={createWorkItemAllowedProjectIds}
      />
      <WorkItemDeleteModal
        workspaceSlug={workspaceSlug?.toString()}
        workItem={workItemDetails}
        isOpen={isDeleteIssueModalOpen}
        onClose={() => toggleDeleteIssueModal(false)}
        onDelete={deleteIssue}
      />
      <BulkDeleteIssuesModal isOpen={isBulkDeleteIssueModalOpen} onClose={() => toggleBulkDeleteIssueModal(false)} />
    </>
  );
}

function isCreatedSubItem(issue: TIssue, workspaceSlug: string | undefined, parentId: string | undefined) {
  return !!workspaceSlug && !!issue.project_id && !!issue.id && issue.parent_id === parentId;
}

function WorkItemDeleteModal({
  workspaceSlug,
  workItem,
  isOpen,
  onClose,
  onDelete,
}: {
  workspaceSlug: string | undefined;
  workItem: ReturnType<typeof useWorkItemByIdentifier>["issue"];
  isOpen: boolean;
  onClose: () => void;
  onDelete: (projectId: string, issueId: string) => Promise<void>;
}) {
  if (!workspaceSlug || !workItem?.id || !workItem.project_id) return null;
  const { project_id: projectId, id: issueId } = workItem;

  return (
    <DeleteIssueModal
      handleClose={onClose}
      isOpen={isOpen}
      data={workItem}
      onSubmit={() => onDelete(projectId, issueId)}
      isEpic={workItem.is_epic}
    />
  );
}

async function deleteWorkItem({
  projectId,
  workspaceSlug,
  issueId,
  isEpic,
  removeEpic,
  removeWorkItem,
  navigate,
}: {
  projectId: string;
  workspaceSlug: string;
  issueId: string;
  isEpic: boolean;
  removeEpic: (projectId: string, issueId: string) => Promise<void>;
  removeWorkItem: (projectId: string, issueId: string) => Promise<void>;
  navigate: (href: string) => Promise<void>;
}) {
  const deleteAction = isEpic ? removeEpic : removeWorkItem;
  try {
    await deleteAction(projectId, issueId);
    await navigate(`/${workspaceSlug}/projects/${projectId}/${isEpic ? "epics" : "issues"}`);
  } catch (error) {
    console.error("Failed to delete issue:", error);
  }
}

function createIssueModalData(cycleId: string | undefined, moduleId: string | undefined) {
  if (cycleId) return { cycle_id: cycleId };
  if (moduleId) return { module_ids: [moduleId] };
  return undefined;
}
