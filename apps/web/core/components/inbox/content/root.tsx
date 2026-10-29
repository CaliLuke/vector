/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useQueryClient } from "@tanstack/react-query";
import { useRouter } from "@tanstack/react-router";
import { useCommittedScope } from "@/hooks/use-committed-scope";
import { useIdentityGeneration } from "@/hooks/use-identity-generation";
import { captureIdentityScope, isIdentityScopeCurrent } from "@/queries/identity";
import { EUserPermissions } from "@plane/constants";
import { useTextSaveStatus } from "@/components/issues/use-text-save-status";
// components
import { ContentWrapper } from "@plane/ui/content-wrapper";
// hooks
import { useIntakeDetail } from "@/hooks/use-intake";
import type { TInboxIssueCurrentTab } from "@plane/types";
import { useProjectPermissions } from "@/hooks/use-permissions";
import { useCurrentUser } from "@/hooks/use-current-user";

// local imports
import { InboxIssueActionsHeader } from "./inbox-issue-header";
import { InboxIssueMainContent } from "./issue-root";

type TInboxContentRoot = {
  workspaceSlug: string;
  projectId: string;
  inboxIssueId: string;
  isMobileSidebar: boolean;
  setIsMobileSidebar: (value: boolean) => void;
  isNotificationEmbed?: boolean;
  embedRemoveCurrentNotification?: () => void;
  navigation?: { tab: TInboxIssueCurrentTab; issueIds: string[] };
};

export const InboxContentRoot = function InboxContentRoot(props: TInboxContentRoot) {
  const {
    workspaceSlug,
    projectId,
    inboxIssueId,
    isMobileSidebar,
    setIsMobileSidebar,
    isNotificationEmbed = false,
    embedRemoveCurrentNotification,
  } = props;
  /// router
  // states
  const {
    status: isSubmitting,
    setTitleStatus,
    setDescriptionStatus,
  } = useTextSaveStatus(`${workspaceSlug}/${projectId}/${inboxIssueId}`);
  // hooks
  const { data: currentUser } = useCurrentUser();
  const client = useQueryClient();
  const router = useRouter();
  const generation = useIdentityGeneration(client);
  const scopeKey = JSON.stringify([
    currentUser?.id,
    generation,
    workspaceSlug,
    projectId,
    inboxIssueId,
    props.navigation?.tab,
    isNotificationEmbed,
  ]);
  const mounted = useCommittedScope(scopeKey);
  const identity = captureIdentityScope(client, currentUser?.id);
  const onDeleteAccepted = () => {
    if (!mounted() || !isIdentityScopeCurrent(client, identity) || isNotificationEmbed) return;
    void router.navigate({ href: `/${workspaceSlug}/projects/${projectId}/intake` });
  };
  const detail = useIntakeDetail(workspaceSlug, projectId, inboxIssueId);
  const inboxIssue = detail.data;
  const { can: canProject, projectRole } = useProjectPermissions(workspaceSlug, projectId);

  if (detail.isError)
    return (
      <div role="alert">
        Could not load the intake work item.{" "}
        <button type="button" onClick={() => void detail.retry()}>
          Retry
        </button>
      </div>
    );
  if (detail.accessDenied) return <div>This intake work item is unavailable.</div>;
  if (!inboxIssue) return <div>Loading intake work item…</div>;

  const isOwner = Boolean(currentUser?.id && inboxIssue.issue.created_by === currentUser.id);
  const isEditable = canProject([EUserPermissions.ADMIN]) || isOwner;

  const isGuest = projectRole === EUserPermissions.GUEST;
  const readOnly = !isOwner && isGuest;

  const isIssueDisabled = [-1, 1, 2].includes(inboxIssue.status);

  return (
    <>
      <div className="relative flex h-full w-full flex-col overflow-hidden">
        <div className="z-[11] min-h-[52px] flex-shrink-0">
          <InboxIssueActionsHeader
            navigation={props.navigation}
            onDeleteAccepted={onDeleteAccepted}
            setIsMobileSidebar={setIsMobileSidebar}
            isMobileSidebar={isMobileSidebar}
            workspaceSlug={workspaceSlug}
            projectId={projectId}
            inboxIssue={inboxIssue}
            isSubmitting={isSubmitting}
            isNotificationEmbed={isNotificationEmbed}
            embedRemoveCurrentNotification={embedRemoveCurrentNotification}
          />
        </div>
        <ContentWrapper className="divide-y-2 divide-subtle-1">
          <InboxIssueMainContent
            workspaceSlug={workspaceSlug}
            projectId={projectId}
            inboxIssue={inboxIssue}
            isEditable={isEditable && !isIssueDisabled && !readOnly}
            setTitleStatus={setTitleStatus}
            setDescriptionStatus={setDescriptionStatus}
          />
        </ContentWrapper>
      </div>
    </>
  );
};
