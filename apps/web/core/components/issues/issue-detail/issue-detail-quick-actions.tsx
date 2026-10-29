/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useLocation, useRouter } from "@tanstack/react-router";
import { useRef } from "react";
// plane imports
import { useTranslation } from "@plane/i18n";
import { CopyLinkIcon } from "@plane/propel/icons";
import { IconButton } from "@plane/propel/icon-button";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { Tooltip } from "@plane/propel/tooltip";
import { generateWorkItemLink } from "@plane/utils/work-item";
import { useWorkItemLinkCopy } from "@/hooks/use-work-item-link-copy";
// hooks
import { useWorkItemDetail } from "@/hooks/use-work-item-detail";
import { useIssueDetailLifecycle } from "@/hooks/use-issue-detail-lifecycle";
import { useCommittedScope } from "@/hooks/use-committed-scope";
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";
import { useCurrentUser } from "@/hooks/use-current-user";

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
  const lifecycle = useIssueDetailLifecycle(detail.scope, issueId, detail.isReady);
  const pathname = useLocation({ select: (location) => location.pathname });
  const isRouteCurrent = useCommittedScope(JSON.stringify([detail.scope, issueId, pathname]));

  // derived values
  const issue = detail.issue;

  const projectIdentifier = projectRows?.find((project) => project.id === projectId)?.identifier ?? "";

  const workItemLink = generateWorkItemLink({
    workspaceSlug: workspaceSlug,
    projectId,
    issueId,
    projectIdentifier,
    sequenceId: issue?.sequence_id,
  });

  const copyLink = useWorkItemLinkCopy({ workspaceSlug, projectId, issueId }, workItemLink);
  if (!issue) return <></>;

  const navigateAfterWrite = (isCurrent: () => boolean, href: string, message: string) => {
    if (!isRouteCurrent() || !isCurrent()) return;
    void router.navigate({ href }).catch(() => {
      if (isRouteCurrent() && isCurrent()) setToast({ title: t("toast.error"), type: TOAST_TYPE.ERROR, message });
    });
  };

  const handleDeleteIssue = async () => {
    const receipt = await lifecycle.remove(workspaceSlug, projectId, issueId);
    navigateAfterWrite(
      receipt.isCurrent,
      issue.archived_at
        ? `/${workspaceSlug}/projects/${projectId}/archives/issues`
        : `/${workspaceSlug}/projects/${projectId}/issues`,
      "The work item was deleted, but the list could not be opened. Refresh to continue."
    );
  };

  const handleArchiveIssue = async () => {
    const receipt = await lifecycle.archive(workspaceSlug, projectId, issueId);
    navigateAfterWrite(
      receipt.isCurrent,
      `/${workspaceSlug}/projects/${projectId}/issues`,
      "The work item was archived, but the list could not be opened. Refresh to continue."
    );
  };

  const handleRestore = async () => {
    const receipt = await lifecycle.restore(workspaceSlug, projectId, issueId);
    navigateAfterWrite(
      receipt.isCurrent,
      workItemLink,
      "The work item was restored, but it could not be opened. Refresh to continue."
    );
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
              <IconButton
                variant="secondary"
                size="lg"
                onClick={copyLink.copy}
                disabled={copyLink.disabled}
                aria-label={t("common.actions.copy_link")}
                icon={CopyLinkIcon}
              />
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
