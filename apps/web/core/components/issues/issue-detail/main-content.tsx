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
import { IssueDescriptionVersions } from "./description-versions";
import { IssueDescriptionInput } from "@/components/issues/description-input";
import { IssueTypeSwitcher } from "@/components/issues/issue-type-switcher";
// hooks
import { useIssueDetailUiState } from "@/providers/issue-detail-ui-state";
import { useWorkItemDetailRecord } from "@/providers/work-item-detail-record";
import { useCurrentUser } from "@/hooks/use-current-user";
import { useTextSaveStatus } from "../use-text-save-status";
import useSize from "@/hooks/use-window-size";
// services
// local imports
import { IssueDetailWidgets } from "../issue-detail-widgets";
import { NameDescriptionUpdateStatus } from "../issue-update-status";
import { PeekOverviewProperties } from "../peek-overview/properties";
import { IssueTitleInput } from "../title-input";
import { IssueActivity } from "./issue-activity";
import { IssueParentDetail } from "./parent";
import { IssueReaction } from "./reactions";
import type { TIssueOperations } from "./root";

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
  const { peekIssue } = useIssueDetailUiState();
  const record = useWorkItemDetailRecord();
  // derived values
  const issue = issueId === record?.issueId ? record.issue : undefined;
  const editingDisabled = isArchived || !isEditable;

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
          disabled={editingDisabled}
        />

        <div className="mb-2.5 flex items-center justify-between gap-4">
          <IssueTypeSwitcher workspaceSlug={workspaceSlug} issueId={issueId} disabled={editingDisabled} />
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
          disabled={editingDisabled}
          value={issue.name}
          containerClassName="-ml-3"
        />

        <IssueDescriptionInput
          issueSequenceId={issue.sequence_id}
          containerClassName="-ml-6 border-none p-0! pl-6!"
          disabled={editingDisabled}
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
            <IssueDescriptionVersions
              workspaceSlug={workspaceSlug}
              projectId={projectId}
              issue={issue}
              disabled={editingDisabled}
              onRestore={(descriptionHTML) => editorRef.current?.setEditorValue(descriptionHTML, true)}
            />
          )}
        </div>
      </div>

      <IssueDetailWidgets
        workspaceSlug={workspaceSlug}
        projectId={projectId}
        issueId={issueId}
        disabled={editingDisabled}
        renderWidgetModals={!isPeekModeActive}
        issueServiceType={EIssueServiceType.ISSUES}
      />

      {windowSize[0] < 768 && (
        <PeekOverviewProperties
          workspaceSlug={workspaceSlug}
          projectId={projectId}
          issueId={issueId}
          issueOperations={issueOperations}
          disabled={editingDisabled}
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
