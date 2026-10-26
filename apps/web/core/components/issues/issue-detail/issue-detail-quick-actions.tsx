/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRouter } from "@tanstack/react-router";
import { useRef } from "react";
// plane imports
import { useTranslation } from "@plane/i18n";
import { CopyLinkIcon } from "@plane/propel/icons";
import { IconButton } from "@plane/propel/icon-button";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { Tooltip } from "@plane/propel/tooltip";
import { EIssuesStoreType } from "@plane/types";
import { generateWorkItemLink } from "@plane/utils/work-item";
import { copyTextToClipboard } from "@plane/utils/string";
// hooks
import { useWorkItemDetail } from "@/hooks/use-work-item-detail";
import { useIssuesActions } from "@/hooks/use-issues-actions";
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";
import { useCurrentUser } from "@/hooks/use-current-user";
import { captureIssueDetailReconciliation } from "@/queries/issue-details";
import { useQueryClient } from "@tanstack/react-query";

import { usePlatformOS } from "@/hooks/use-platform-os";
// local imports
import { WorkItemDetailQuickActions } from "../issue-layouts/quick-action-dropdowns";
import { IssueSubscription } from "./subscription";

type Props = {
  workspaceSlug: string;
  projectId: string;
  issueId: string;
};

export function IssueDetailQuickActions(props: Props) {
  const { workspaceSlug, projectId, issueId } = props;
  const { t } = useTranslation();

  // ref
  const parentRef = useRef<HTMLDivElement>(null);

  // router
  const router = useRouter();

  // hooks
  const { data: currentUser } = useCurrentUser();
  const { isMobile } = usePlatformOS();
  const projectCatalogue = useWorkspaceProjectDetails();
  const projectRows = projectCatalogue.projects;
  const detail = useWorkItemDetail(workspaceSlug, projectId, issueId);
  const queryClient = useQueryClient();
  const projectActions = useIssuesActions(EIssuesStoreType.PROJECT);
  const archivedActions = useIssuesActions(EIssuesStoreType.ARCHIVED);

  // derived values
  const issue = detail.issue;
  if (!issue) return <></>;

  const projectIdentifier = projectRows?.find((project) => project.id === projectId)?.identifier ?? "";

  const workItemLink = generateWorkItemLink({
    workspaceSlug: workspaceSlug,
    projectId,
    issueId,
    projectIdentifier,
    sequenceId: issue?.sequence_id,
  });

  // handlers
  const handleCopyText = async () => {
    try {
      const originURL = typeof window !== "undefined" && window.location.origin ? window.location.origin : "";
      await copyTextToClipboard(`${originURL}${workItemLink}`);
      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: t("common.link_copied"),
        message: t("common.copied_to_clipboard"),
      });
    } catch (_error) {
      setToast({
        title: t("toast.error"),
        type: TOAST_TYPE.ERROR,
      });
    }
  };

  const handleDeleteIssue = async () => {
    const detailOwner = captureIssueDetailReconciliation(queryClient, detail.scope, [issueId]);
    const redirectionPath = issue?.archived_at
      ? `/${workspaceSlug}/projects/${projectId}/archives/issues`
      : `/${workspaceSlug}/projects/${projectId}/issues`;

    if (issue?.archived_at) await archivedActions.removeIssue(projectId, issueId);
    else await projectActions.removeIssue(projectId, issueId);
    detailOwner.remove();
    void router.navigate({ href: redirectionPath }).catch(() => {
      setToast({
        title: t("toast.error"),
        message: "The work item was deleted, but the list could not be opened. Refresh to continue.",
        type: TOAST_TYPE.ERROR,
      });
    });
  };

  const handleArchiveIssue = async () => {
    const detailOwner = captureIssueDetailReconciliation(queryClient, detail.scope, [issueId]);
    await projectActions.archiveIssue?.(projectId, issueId);
    void detailOwner.refresh();
    void router.navigate({ href: `/${workspaceSlug}/projects/${projectId}/issues` }).catch(() => {
      setToast({
        title: t("toast.error"),
        message: "The work item was archived, but the list could not be opened. Refresh to continue.",
        type: TOAST_TYPE.ERROR,
      });
    });
  };

  const handleRestore = async () => {
    if (!workspaceSlug || !projectId || !issueId || !archivedActions.restoreIssue) return;
    try {
      await archivedActions.restoreIssue(projectId, issueId);
      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: t("issue.restore.success.title"),
        message: t("issue.restore.success.message"),
      });
      void router.navigate({ href: workItemLink });
    } catch (_error) {
      setToast({
        title: t("toast.error"),
        type: TOAST_TYPE.ERROR,
        message: t("issue.restore.failed.message"),
      });
    }
  };

  return (
    <>
      <div className="flex flex-shrink-0 items-center justify-end">
        <div className="flex flex-wrap items-center gap-2">
          {currentUser && !issue?.archived_at && (
            <IssueSubscription workspaceSlug={workspaceSlug} projectId={projectId} issueId={issueId} />
          )}
          <div className="flex flex-wrap items-center gap-2 text-tertiary">
            <Tooltip tooltipContent={t("common.actions.copy_link")} isMobile={isMobile}>
              <IconButton variant="secondary" size="lg" onClick={handleCopyText} icon={CopyLinkIcon} />
            </Tooltip>
            <WorkItemDetailQuickActions
              parentRef={parentRef}
              issue={issue}
              handleDelete={handleDeleteIssue}
              handleArchive={handleArchiveIssue}
              handleRestore={handleRestore}
            />
          </div>
        </div>
      </div>
    </>
  );
}
