/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { workItemServiceType } from "@/helpers/work-item-service-type";
import { useMemo, useRef } from "react";
import { useQueryClient } from "@tanstack/react-query";
// plane imports
import type { EditorRefApi } from "@plane/editor";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { TNameDescriptionLoader } from "@plane/types";
import {
  EFileAssetType,
  EInboxIssueSource,
  EInboxIssueStatus,
  EIssueServiceType,
  EIssuesStoreType,
} from "@plane/types";
// components
import { DescriptionVersionsRoot } from "@/components/core/description-versions";
import { IssueDescriptionInput } from "@/components/issues/description-input";
import { DescriptionInputLoader } from "@/components/editor/rich-text/description-input/loader";
import { IssueAttachmentRoot } from "@/components/issues/attachment";
import type { TIssueOperations } from "@/components/issues/issue-detail";
import { IssueActivity } from "@/components/issues/issue-detail/issue-activity";
import { IssueReaction } from "@/components/issues/issue-detail/reactions";
import { IssueTitleInput } from "@/components/issues/title-input";
// hooks
import { useIssuesActions } from "@/hooks/use-issues-actions";
import { useWorkspaceMemberRoster } from "@/hooks/use-member-rosters";
import { useIntakeMutation, useIntakeDetail } from "@/hooks/use-intake";
import { useCurrentUser } from "@/hooks/use-current-user";
import { useWorkItemDetailScope } from "@/hooks/use-work-item-detail";
import { captureIssueDetailReconciliation } from "@/queries/issue-details";
// services
import { IntakeWorkItemVersionService } from "@/services/inbox";
// stores
import type { TInboxIssue } from "@plane/types";
// local imports
import { InboxIssueContentProperties } from "./issue-properties";
// services init
const intakeWorkItemVersionService = new IntakeWorkItemVersionService();

type Props = {
  workspaceSlug: string;
  projectId: string;
  inboxIssue: TInboxIssue;
  isEditable: boolean;
  setTitleStatus: (value: TNameDescriptionLoader) => void;
  setDescriptionStatus: (value: TNameDescriptionLoader) => void;
};

function useInboxIssueOperations({
  inboxIssue,
  projectId,
  mutateAsync,
  refetch,
  issueActions,
  queryClient,
  detailScope,
}: {
  inboxIssue: TInboxIssue;
  projectId: string;
  mutateAsync: ReturnType<typeof useIntakeMutation>["mutateAsync"];
  refetch: ReturnType<typeof useIntakeDetail>["refetch"];
  issueActions: ReturnType<typeof useIssuesActions>;
  queryClient: ReturnType<typeof useQueryClient>;
  detailScope: ReturnType<typeof useWorkItemDetailScope>["scope"];
}) {
  return useMemo<TIssueOperations>(
    () => ({
      updateText: async (_workspaceSlug, _projectId, _issueId, data) => {
        const detailOwner = captureIssueDetailReconciliation(queryClient, detailScope, [inboxIssue.issue.id]);
        const receipt = await mutateAsync({ kind: "issue", issueId: inboxIssue.issue.id, data });
        if (receipt.saved?.issue) await detailOwner.reconcile(receipt.saved.issue);
      },
      fetch: async (_workspaceSlug, _projectId, _issueId) => {
        await refetch();
      },
      remove: async (_workspaceSlug, _projectId, issueId) => {
        const detailOwner = captureIssueDetailReconciliation(queryClient, detailScope, [issueId]);
        try {
          await issueActions.removeIssue(projectId, issueId);
          detailOwner.remove();
          setToast({
            title: "Success!",
            type: TOAST_TYPE.SUCCESS,
            message: "Work item deleted successfully",
          });
        } catch (error) {
          console.error("Error in deleting work item:", error);
          setToast({
            title: "Error!",
            type: TOAST_TYPE.ERROR,
            message: "Work item delete failed",
          });
          throw error;
        }
      },
      update: async (_workspaceSlug, _projectId, _issueId, data) => {
        const detailOwner = captureIssueDetailReconciliation(queryClient, detailScope, [inboxIssue.issue.id]);
        try {
          const receipt = await mutateAsync({ kind: "issue", issueId: inboxIssue.issue.id, data });
          if (receipt.saved?.issue) await detailOwner.reconcile(receipt.saved.issue);
        } catch (error) {
          setToast({
            title: "Work item update failed",
            type: TOAST_TYPE.ERROR,
            message: "Work item update failed",
          });
          throw error;
        }
      },
      archive: async (_workspaceSlug, _projectId, issueId) => {
        const detailOwner = captureIssueDetailReconciliation(queryClient, detailScope, [issueId]);
        await issueActions.archiveIssue?.(projectId, issueId);
        void detailOwner.refresh();
      },
    }),
    [detailScope, inboxIssue.issue.id, issueActions, mutateAsync, projectId, queryClient, refetch]
  );
}

export const InboxIssueMainContent = function InboxIssueMainContent(props: Props) {
  const { workspaceSlug, projectId, inboxIssue, isEditable, setTitleStatus, setDescriptionStatus } = props;
  // refs
  const editorRef = useRef<EditorRefApi | null>(null);
  // store hooks
  const { data: currentUser } = useCurrentUser();
  const { memberById } = useWorkspaceMemberRoster(workspaceSlug);
  const { mutateAsync } = useIntakeMutation(workspaceSlug, projectId);
  const { refetch } = useIntakeDetail(workspaceSlug, projectId, inboxIssue.issue.id);
  const issueActions = useIssuesActions(EIssuesStoreType.PROJECT);
  const queryClient = useQueryClient();
  // derived values
  const issue = inboxIssue.issue;
  const isIntakeAccepted = inboxIssue.status === EInboxIssueStatus.ACCEPTED;
  const detailAccess = useWorkItemDetailScope(
    workspaceSlug,
    issue?.project_id ?? projectId,
    issue ? workItemServiceType(issue) : EIssueServiceType.ISSUES,
    issue?.id
  );
  const issueOperations = useInboxIssueOperations({
    inboxIssue,
    projectId,
    mutateAsync,
    refetch,
    issueActions,
    queryClient,
    detailScope: detailAccess.scope,
  });
  const issueCreatedAt = useMemo(
    () => new Date(issue?.created_at ?? issue?.updated_at ?? 0),
    [issue?.created_at, issue?.updated_at]
  );

  if (!issue?.project_id || !issue.id) return <></>;

  return (
    <>
      <div className="space-y-4 pb-4">
        <IssueTitleInput
          workspaceSlug={workspaceSlug}
          projectId={issue.project_id}
          issueId={issue.id}
          setIsSubmitting={setTitleStatus}
          issueOperations={issueOperations}
          disabled={!isEditable}
          value={issue.name}
          containerClassName="-ml-3"
        />

        <InboxIssueDescriptionSection
          currentUserId={currentUser?.id}
          editorRef={editorRef}
          inboxIssue={inboxIssue}
          issue={issue}
          issueProjectId={issue.project_id}
          issueCreatedAt={issueCreatedAt}
          issueOperations={issueOperations}
          isEditable={isEditable}
          memberById={memberById}
          projectId={projectId}
          setDescriptionStatus={setDescriptionStatus}
          workspaceSlug={workspaceSlug}
        />
      </div>

      <div className="py-4">
        <IssueAttachmentRoot
          workspaceSlug={workspaceSlug}
          projectId={projectId}
          issueId={issue.id}
          disabled={!isEditable}
        />
      </div>

      <div className="py-4">
        <InboxIssueContentProperties
          workspaceSlug={workspaceSlug}
          projectId={projectId}
          issue={issue}
          issueOperations={issueOperations}
          isEditable={isEditable}
          duplicateIssueDetails={inboxIssue?.duplicate_issue_detail}
          isIntakeAccepted={isIntakeAccepted}
        />
      </div>

      <div className="pt-4">
        <IssueActivity
          serviceType={workItemServiceType(issue)}
          workspaceSlug={workspaceSlug}
          projectId={projectId}
          issueId={issue.id}
          isIntakeIssue
        />
      </div>
    </>
  );
};

function InboxIssueDescriptionSection(props: {
  currentUserId: string | undefined;
  editorRef: React.RefObject<EditorRefApi | null>;
  inboxIssue: TInboxIssue;
  issue: TInboxIssue["issue"];
  issueProjectId: string;
  issueCreatedAt: Date;
  issueOperations: TIssueOperations;
  isEditable: boolean;
  memberById: ReturnType<typeof useWorkspaceMemberRoster>["memberById"];
  projectId: string;
  setDescriptionStatus: Props["setDescriptionStatus"];
  workspaceSlug: string;
}) {
  const {
    currentUserId,
    editorRef,
    inboxIssue,
    issue,
    issueProjectId,
    issueCreatedAt,
    issueOperations,
    isEditable,
    memberById,
    projectId,
    setDescriptionStatus,
    workspaceSlug,
  } = props;
  const description =
    issue.description_html === undefined ? (
      <DescriptionInputLoader />
    ) : (
      <IssueDescriptionInput
        issueSequenceId={issue.sequence_id ?? undefined}
        containerClassName="-ml-3 border-none"
        disabled={!isEditable}
        editorRef={editorRef}
        entityId={issue.id}
        fileAssetType={EFileAssetType.ISSUE_DESCRIPTION}
        initialValue={issue.description_html ?? "<p></p>"}
        key={issue.id}
        issueOperations={issueOperations}
        projectId={issueProjectId}
        setIsSubmitting={setDescriptionStatus}
        workspaceSlug={workspaceSlug}
      />
    );
  const createdBy =
    inboxIssue.source === EInboxIssueSource.FORMS
      ? "Intake Form user"
      : (memberById(issue.created_by ?? "")?.member.display_name ?? "");
  return (
    <>
      {description}
      <div className="flex items-center justify-between gap-2">
        {currentUserId && (
          <IssueReaction
            serviceType={workItemServiceType(issue)}
            workspaceSlug={workspaceSlug}
            projectId={projectId}
            issueId={issue.id}
          />
        )}
        {isEditable && (
          <DescriptionVersionsRoot
            className="flex-shrink-0"
            entityInformation={{
              createdAt: issueCreatedAt,
              createdByDisplayName: createdBy,
              id: issue.id,
              isRestoreDisabled: !isEditable,
            }}
            fetchHandlers={{
              listDescriptionVersions: (issueId) =>
                intakeWorkItemVersionService.listDescriptionVersions(workspaceSlug, projectId, issueId),
              retrieveDescriptionVersion: (issueId, versionId) =>
                intakeWorkItemVersionService.retrieveDescriptionVersion(workspaceSlug, projectId, issueId, versionId),
            }}
            handleRestore={(descriptionHTML) => editorRef.current?.setEditorValue(descriptionHTML, true)}
            projectId={projectId}
            workspaceSlug={workspaceSlug}
          />
        )}
      </div>
    </>
  );
}
