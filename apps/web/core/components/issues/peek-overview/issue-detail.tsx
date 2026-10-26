/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { workItemServiceType } from "@/helpers/work-item-service-type";
// oxlint-disable no-shadow

// plane imports
import type { EditorRefApi } from "@plane/editor";
import { EFileAssetType } from "@plane/types";
import type { TNameDescriptionLoader } from "@plane/types";
// components
import { DescriptionVersionsRoot } from "@/components/core/description-versions";
import { IssueDescriptionInput } from "@/components/issues/description-input";
// hooks
import { useWorkItemDetailRecord } from "@/providers/work-item-detail-record";
import { useWorkspaceMemberRoster } from "@/hooks/use-member-rosters";
import { useCurrentUser } from "@/hooks/use-current-user";
// plane web components
import { IssueTypeSwitcher } from "@/components/issues/issue-type-switcher";
// plane web hooks
// services
import { WorkItemVersionService } from "@/services/issue";
// local components
import type { TIssueOperations } from "../issue-detail";
import { IssueParentDetail } from "../issue-detail/parent";
import { IssueReaction } from "../issue-detail/reactions";
import { IssueTitleInput } from "../title-input";
// services init
const workItemVersionService = new WorkItemVersionService();

type Props = {
  editorRef: React.RefObject<EditorRefApi | null>;
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  issueOperations: TIssueOperations;
  disabled: boolean;
  isArchived: boolean;
  onEditorOpenChange: (open: boolean) => void;
  setTitleStatus: (value: TNameDescriptionLoader) => void;
  setDescriptionStatus: (value: TNameDescriptionLoader) => void;
};

export const PeekOverviewIssueDetails = function PeekOverviewIssueDetails(props: Props) {
  const {
    editorRef,
    workspaceSlug,
    issueId,
    issueOperations,
    disabled,
    isArchived,
    onEditorOpenChange,
    setTitleStatus,
    setDescriptionStatus,
  } = props;
  // store hooks
  const { data: currentUser } = useCurrentUser();
  const record = useWorkItemDetailRecord();

  const { memberById } = useWorkspaceMemberRoster(workspaceSlug);
  // derived values
  const issue = issueId === record?.issueId ? record.issue : undefined;

  if (!issue || !issue.project_id) return <></>;

  return (
    <div className="space-y-2">
      {issue.parent_id && (
        <IssueParentDetail
          workspaceSlug={workspaceSlug}
          projectId={issue.project_id}
          issueId={issueId}
          issue={issue}
          issueOperations={issueOperations}
        />
      )}
      <div className="flex items-center justify-between gap-2">
        <IssueTypeSwitcher
          workspaceSlug={workspaceSlug}
          onEditorOpenChange={onEditorOpenChange}
          issueId={issueId}
          disabled={isArchived || disabled}
        />
      </div>
      <IssueTitleInput
        workspaceSlug={workspaceSlug}
        projectId={issue.project_id}
        issueId={issue.id}
        setIsSubmitting={setTitleStatus}
        issueOperations={issueOperations}
        disabled={disabled || isArchived}
        value={issue.name}
        containerClassName="-ml-3"
      />

      <IssueDescriptionInput
        issueSequenceId={issue.sequence_id}
        containerClassName="-ml-3 border-none"
        disabled={disabled || isArchived}
        editorRef={editorRef}
        entityId={issue.id}
        fileAssetType={EFileAssetType.ISSUE_DESCRIPTION}
        initialValue={issue.description_html}
        key={issue.id}
        issueOperations={issueOperations}
        setIsSubmitting={setDescriptionStatus}
        projectId={issue.project_id}
        workspaceSlug={workspaceSlug}
      />

      <div className="flex items-center justify-between gap-2">
        {currentUser && (
          <IssueReaction
            serviceType={workItemServiceType(issue)}
            workspaceSlug={workspaceSlug}
            projectId={issue.project_id}
            issueId={issueId}
            disabled={isArchived}
          />
        )}
        {!disabled && (
          <DescriptionVersionsRoot
            className="flex-shrink-0"
            entityInformation={{
              createdAt: new Date(issue.created_at ?? 0),
              createdByDisplayName: memberById(issue.created_by ?? "")?.member.display_name ?? "",
              id: issueId,
              isRestoreDisabled: disabled || isArchived,
            }}
            fetchHandlers={{
              listDescriptionVersions: (issueId) =>
                workItemVersionService.listDescriptionVersions(
                  workspaceSlug,
                  issue.project_id?.toString() ?? "",
                  issueId
                ),
              retrieveDescriptionVersion: (issueId, versionId) =>
                workItemVersionService.retrieveDescriptionVersion(
                  workspaceSlug,
                  issue.project_id?.toString() ?? "",
                  issueId,
                  versionId
                ),
            }}
            handleRestore={(descriptionHTML) => editorRef.current?.setEditorValue(descriptionHTML, true)}
            projectId={issue.project_id}
            workspaceSlug={workspaceSlug}
          />
        )}
      </div>
    </div>
  );
};
