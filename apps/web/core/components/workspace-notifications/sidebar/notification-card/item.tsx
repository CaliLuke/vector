/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { MouseEvent } from "react";
import { useState } from "react";

import { Clock } from "lucide-react";
// plane imports
import { Avatar, Row } from "@plane/ui";
import { cn } from "@plane/utils/common";
import { calculateTimeAgo, renderFormattedDate, renderFormattedTime } from "@plane/utils/datetime";
import { getFileURL } from "@plane/utils/file";
// hooks
import { useNotifications } from "@/components/workspace-notifications/provider";
import { useUpdateNotification } from "@/queries/notifications";
import { useIssueDetailUiState } from "@/providers/issue-detail-ui-state";
import { useWorkspaceBySlug } from "@/hooks/use-workspaces";
// local imports
import { NotificationContent } from "./content";
import { NotificationOption } from "./options";

type TNotificationItem = {
  workspaceSlug: string;
  notificationId: string;
};

const currentTimestamp = () => new Date().toISOString();

export const NotificationItem = function NotificationItem(props: TNotificationItem) {
  const { workspaceSlug, notificationId } = props;
  // hooks
  const { currentSelectedNotificationId, setCurrentSelectedNotificationId, notifications } = useNotifications();
  const notification = notifications.get(notificationId);
  const update = useUpdateNotification(workspaceSlug, notificationId);
  const { getIsIssuePeeked, setPeekIssue } = useIssueDetailUiState();
  const { data: workspaceDetails } = useWorkspaceBySlug(workspaceSlug);
  // states
  const [customSnoozeModal, setCustomSnoozeModal] = useState(false);

  if (!notification) return null;

  // derived values
  const projectId = notification.project;
  const issueId = notification.data?.issue?.id;
  const workspace = workspaceDetails;

  const notificationField = notification.data?.issue_activity.field;
  const notificationTriggeredBy = notification.triggered_by_details;

  const handleNotificationIssuePeekOverview = async (event: MouseEvent) => {
    if (
      !(event.target instanceof Element) ||
      !event.currentTarget.contains(event.target) ||
      event.target.closest("button, a, input, select, textarea, [role=button]")
    )
      return;
    if (workspaceSlug && projectId && issueId && !customSnoozeModal) {
      setPeekIssue(undefined);
      setCurrentSelectedNotificationId(notificationId);

      // make the notification as read
      if (!notification.read_at && !update.isPending) {
        try {
          await update.mutateAsync({ notification, field: "read_at", value: currentTimestamp() });
        } catch (error) {
          console.error(error);
        }
      }

      if (notification.is_inbox_issue === false) {
        if (!getIsIssuePeeked(issueId)) {
          setPeekIssue({ workspaceSlug, projectId, issueId });
        }
      }
    }
  };

  if (!notificationField || !workspace?.id || !projectId) return <></>;

  return (
    <Row
      className={cn(
        "group relative flex cursor-pointer items-center gap-2 border-b border-subtle py-4 transition-all",
        {
          "bg-layer-1/30": currentSelectedNotificationId === notification.id,
          "bg-accent-primary/5": !notification.read_at,
        }
      )}
      onClick={handleNotificationIssuePeekOverview}
    >
      {!notification.read_at && (
        <div className="absolute top-[50%] left-2 h-1.5 w-1.5 flex-shrink-0 rounded-full bg-accent-primary" />
      )}

      <div className="relative flex w-full gap-2">
        <div className="relative flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-full bg-layer-1">
          {notificationTriggeredBy && (
            <Avatar
              name={notificationTriggeredBy.display_name || notificationTriggeredBy?.first_name}
              src={getFileURL(notificationTriggeredBy.avatar_url)}
              size={42}
              shape="circle"
              className="bg-layer-1 text-body-sm-medium"
            />
          )}
        </div>

        <div className="-mt-2 w-full space-y-1">
          <div className="relative flex h-8 items-center gap-3">
            <div className="line-clamp-1 w-full truncate overflow-hidden text-body-xs-medium break-all whitespace-normal text-primary">
              <NotificationContent
                notification={notification}
                workspaceId={workspace.id}
                workspaceSlug={workspaceSlug}
                projectId={projectId}
              />
            </div>
            <NotificationOption
              workspaceSlug={workspaceSlug}
              notificationId={notification.id}
              customSnoozeModal={customSnoozeModal}
              setCustomSnoozeModal={setCustomSnoozeModal}
            />
          </div>

          <div className="relative flex items-center gap-3 text-caption-sm-regular text-secondary">
            <div className="line-clamp-1 w-full truncate overflow-hidden break-words whitespace-normal">
              {notification.data?.issue?.identifier}-{notification.data?.issue?.sequence_id}&nbsp;
              {notification.data?.issue?.name}
            </div>
            <div className="flex-shrink-0">
              {notification.snoozed_till ? (
                <p className="flex flex-shrink-0 items-center justify-end gap-x-1 text-tertiary">
                  <Clock className="h-4 w-4" />
                  <span>
                    Till {renderFormattedDate(notification.snoozed_till)},&nbsp;
                    {renderFormattedTime(notification.snoozed_till, "12-hour")}
                  </span>
                </p>
              ) : (
                <p className="mt-auto flex-shrink-0 text-tertiary">
                  {notification.created_at && calculateTimeAgo(notification.created_at)}
                </p>
              )}
            </div>
          </div>
        </div>
      </div>
    </Row>
  );
};
