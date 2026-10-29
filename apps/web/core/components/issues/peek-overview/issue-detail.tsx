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
import { IssueDescriptionVersions } from "../issue-detail/description-versions";
import { IssueDescriptionInput } from "@/components/issues/description-input";
// hooks
import { useWorkItemDetailRecord } from "@/providers/work-item-detail-record";
import { useCurrentUser } from "@/hooks/use-current-user";
// plane web components
import { IssueTypeSwitcher } from "@/components/issues/issue-type-switcher";
// plane web hooks
// services
// local components
import type { TIssueOperations } from "../issue-detail";
import { IssueParentDetail } from "../issue-detail/parent";
import { IssueReaction } from "../issue-detail/reactions";
import { IssueTitleInput } from "../title-input";

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

  // derived values
  const issue = issueId === record?.issueId ? record.issue : undefined;
  const editingDisabled = isArchived || disabled;

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
          disabled={editingDisabled}
        />
      )}
      <div className="flex items-center justify-between gap-2">
        <IssueTypeSwitcher
          workspaceSlug={workspaceSlug}
          onEditorOpenChange={onEditorOpenChange}
          issueId={issueId}
          disabled={editingDisabled}
        />
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
        containerClassName="-ml-3 border-none"
        disabled={editingDisabled}
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
          <IssueDescriptionVersions
            workspaceSlug={workspaceSlug}
            projectId={issue.project_id}
            issue={issue}
            disabled={editingDisabled}
            onRestore={(descriptionHTML) => editorRef.current?.setEditorValue(descriptionHTML, true)}
          />
        )}
      </div>
    </div>
  );
};
