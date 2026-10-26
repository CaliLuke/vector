/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useQuery } from "@tanstack/react-query";
// plane imports
import { getNumberCount } from "@plane/utils/string";
// components
import { CountChip } from "@/components/common/count-chip";
// hooks
import { emptyNotificationCounts, notificationCountOptions } from "@/queries/notifications";

type TNotificationAppSidebarOption = {
  workspaceSlug: string;
};

export const NotificationAppSidebarOption = function NotificationAppSidebarOption(
  props: TNotificationAppSidebarOption
) {
  const { workspaceSlug } = props;
  // hooks

  const { data: unreadNotificationsCount = emptyNotificationCounts } = useQuery(
    notificationCountOptions(workspaceSlug)
  );

  // derived values
  const isMentionsEnabled = unreadNotificationsCount.mention_unread_notifications_count > 0 ? true : false;
  const totalNotifications = isMentionsEnabled
    ? unreadNotificationsCount.mention_unread_notifications_count
    : unreadNotificationsCount.total_unread_notifications_count;

  if (totalNotifications <= 0) return <></>;

  return (
    <div className="ml-auto">
      <CountChip count={`${isMentionsEnabled ? `@ ` : ``}${getNumberCount(totalNotifications)}`} />
    </div>
  );
};
