/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// oxlint-disable no-shadow
import { workItemServiceType } from "@/helpers/work-item-service-type";
import { useRef } from "react";

// plane imports
import type { EditorRefApi } from "@plane/editor";
import { EFileAssetType, EIssueServiceType } from "@plane/types";
// components
import { DescriptionVersionsRoot } from "@/components/core/description-versions";
import { IssueDescriptionInput } from "@/components/issues/description-input";
import { IssueTypeSwitcher } from "@/components/issues/issue-type-switcher";
// hooks
import { useIssueDetailUiState } from "@/providers/issue-detail-ui-state";
import { useWorkItemDetailRecord } from "@/providers/work-item-detail-record";
import { useWorkspaceMemberRoster } from "@/hooks/use-member-rosters";
import { useCurrentUser } from "@/hooks/use-current-user";
import { useTextSaveStatus } from "../use-text-save-status";
import useSize from "@/hooks/use-window-size";
// services
import { WorkItemVersionService } from "@/services/issue";
// local imports
import { IssueDetailWidgets } from "../issue-detail-widgets";
import { NameDescriptionUpdateStatus } from "../issue-update-status";
import { PeekOverviewProperties } from "../peek-overview/properties";
import { IssueTitleInput } from "../title-input";
import { IssueActivity } from "./issue-activity";
import { IssueParentDetail } from "./parent";
import { IssueReaction } from "./reactions";
import type { TIssueOperations } from "./root";
// services init
const workItemVersionService = new WorkItemVersionService();

type Props = {
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  issueOperations: TIssueOperations;
  isEditable: boolean;
  isArchived: boolean;
};

export const IssueMainContent = function IssueMainContent(props: Props) {
  const { workspaceSlug, projectId, issueId, issueOperations, isEditable, isArchived } = props;
  // refs
  const editorRef = useRef<EditorRefApi>(null);
  // states
  const {
    status: isSubmitting,
    setTitleStatus,
    setDescriptionStatus,
  } = useTextSaveStatus(`${workspaceSlug}/${projectId}/${issueId}`);
  // hooks
  const windowSize = useSize();
  const { data: currentUser } = useCurrentUser();
  const { memberById } = useWorkspaceMemberRoster(workspaceSlug);
  const { peekIssue } = useIssueDetailUiState();
  const record = useWorkItemDetailRecord();
  // derived values
  const issue = issueId === record?.issueId ? record.issue : undefined;

  if (!issue || !issue.project_id) return <></>;

  const isPeekModeActive = Boolean(peekIssue);

  return (
    <>
      <div className="space-y-4 rounded-lg">
        <IssueParentDetail
          workspaceSlug={workspaceSlug}
          projectId={projectId}
          issueId={issueId}
          issue={issue}
          issueOperations={issueOperations}
        />

        <div className="mb-2.5 flex items-center justify-between gap-4">
          <IssueTypeSwitcher workspaceSlug={workspaceSlug} issueId={issueId} disabled={isArchived || !isEditable} />
          <div className="flex items-center gap-3">
            <NameDescriptionUpdateStatus isSubmitting={isSubmitting} />
          </div>
        </div>

        <IssueTitleInput
          workspaceSlug={workspaceSlug}
          projectId={issue.project_id}
          issueId={issue.id}
          setIsSubmitting={setTitleStatus}
          issueOperations={issueOperations}
          disabled={isArchived || !isEditable}
          value={issue.name}
          containerClassName="-ml-3"
        />

        <IssueDescriptionInput
          issueSequenceId={issue.sequence_id}
          containerClassName="-ml-6 border-none p-0! pl-6!"
          disabled={isArchived || !isEditable}
          editorRef={editorRef}
          entityId={issue.id}
          fileAssetType={EFileAssetType.ISSUE_DESCRIPTION}
          initialValue={issue.description_html}
          key={issue.id}
          issueOperations={issueOperations}
          projectId={issue.project_id}
          setIsSubmitting={setDescriptionStatus}
          workspaceSlug={workspaceSlug}
        />

        <div className="flex items-center justify-between gap-2">
          {currentUser && (
            <IssueReaction
              serviceType={workItemServiceType(issue)}
              className="flex-shrink-0"
              workspaceSlug={workspaceSlug}
              projectId={projectId}
              issueId={issueId}
              disabled={isArchived}
            />
          )}
          {isEditable && (
            <DescriptionVersionsRoot
              className="flex-shrink-0"
              entityInformation={{
                createdAt: new Date(issue.created_at ?? 0),
                createdByDisplayName: memberById(issue.created_by ?? "")?.member.display_name ?? "",
                id: issueId,
                isRestoreDisabled: !isEditable || isArchived,
              }}
              fetchHandlers={{
                listDescriptionVersions: (issueId) =>
                  workItemVersionService.listDescriptionVersions(workspaceSlug, projectId, issueId),
                retrieveDescriptionVersion: (issueId, versionId) =>
                  workItemVersionService.retrieveDescriptionVersion(workspaceSlug, projectId, issueId, versionId),
              }}
              handleRestore={(descriptionHTML) => editorRef.current?.setEditorValue(descriptionHTML, true)}
              projectId={projectId}
              workspaceSlug={workspaceSlug}
            />
          )}
        </div>
      </div>

      <IssueDetailWidgets
        workspaceSlug={workspaceSlug}
        projectId={projectId}
        issueId={issueId}
        disabled={!isEditable || isArchived}
        renderWidgetModals={!isPeekModeActive}
        issueServiceType={EIssueServiceType.ISSUES}
      />

      {windowSize[0] < 768 && (
        <PeekOverviewProperties
          workspaceSlug={workspaceSlug}
          projectId={projectId}
          issueId={issueId}
          issueOperations={issueOperations}
          disabled={!isEditable || isArchived}
        />
      )}

      <IssueActivity
        serviceType={workItemServiceType(issue)}
        workspaceSlug={workspaceSlug}
        projectId={projectId}
        issueId={issueId}
        disabled={isArchived}
      />
    </>
  );
};
