/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback, useEffect } from "react";

// plane imports
import { EmptyStateCompact } from "@plane/propel/empty-state";
import { cn } from "@plane/utils/common";
// components
// hooks
import { useNotifications } from "@/components/workspace-notifications/provider";
import { useWorkspaceIssueProperties } from "@/hooks/use-workspace-issue-properties";
import { useNotificationPreview } from "@/hooks/use-notification-preview";
// local imports
import { InboxContentRoot } from "../inbox/content";

type NotificationsRootProps = {
  workspaceSlug: string;
};

export const NotificationsRoot = function NotificationsRoot({ workspaceSlug }: NotificationsRootProps) {
  // hooks
  const { currentSelectedNotificationId, setCurrentSelectedNotificationId, notifications } = useNotifications();
  const { isWorkItem, PeekOverviewComponent, setPeekWorkItem } = useNotificationPreview();
  // derived values
  const selectedNotification = currentSelectedNotificationId
    ? notifications.get(currentSelectedNotificationId)
    : undefined;
  const workspace_slug = workspaceSlug;
  const project_id = selectedNotification?.project;
  const issue_id = selectedNotification?.data?.issue?.id;
  const is_inbox_issue = selectedNotification?.is_inbox_issue;

  useWorkspaceIssueProperties(workspaceSlug);

  const embedRemoveCurrentNotification = useCallback(
    () => setCurrentSelectedNotificationId(undefined),
    [setCurrentSelectedNotificationId]
  );

  // clearing up the selected notifications when unmounting the page
  useEffect(
    () => () => {
      setPeekWorkItem(undefined);
    },
    [setPeekWorkItem]
  );

  return (
    <div className={cn("h-full w-full overflow-hidden", isWorkItem && "overflow-y-auto")}>
      {!currentSelectedNotificationId ? (
        <div className="flex size-full items-center justify-center">
          <EmptyStateCompact assetKey="unknown" assetClassName="size-20" />
        </div>
      ) : (
        <>
          {is_inbox_issue && project_id && issue_id ? (
            <InboxContentRoot
              setIsMobileSidebar={() => {}}
              isMobileSidebar={false}
              workspaceSlug={workspace_slug}
              projectId={project_id}
              inboxIssueId={issue_id}
              isNotificationEmbed
              embedRemoveCurrentNotification={embedRemoveCurrentNotification}
            />
          ) : (
            <PeekOverviewComponent embedIssue embedRemoveCurrentNotification={embedRemoveCurrentNotification} />
          )}
        </>
      )}
    </div>
  );
};
