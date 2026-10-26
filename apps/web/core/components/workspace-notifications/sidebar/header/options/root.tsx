/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { CheckCheck, RefreshCw } from "lucide-react";
// plane imports
import { useTranslation } from "@plane/i18n";
import { Tooltip } from "@plane/propel/tooltip";
import { Spinner } from "@plane/ui";
// hooks
import { useNotifications } from "@/components/workspace-notifications/provider";
import { usePlatformOS } from "@/hooks/use-platform-os";
// local imports
import { NotificationFilter } from "../../filters/menu";
import { NotificationHeaderMenuOption } from "./menu-option";
import { IconButton } from "@plane/propel/icon-button";

export const NotificationSidebarHeaderOptions = function NotificationSidebarHeaderOptions() {
  // hooks
  const { isMobile } = usePlatformOS();
  const { list, markAllRead } = useNotifications();
  const { t } = useTranslation();

  const refreshNotifications = async () => {
    if (list.isFetching || markAllRead.isPending) return;
    try {
      await list.refetch();
    } catch (error) {
      console.error(error);
    }
  };

  const handleMarkAllNotificationsAsRead = async () => {
    // NOTE: We are using loader to prevent continues request when we are making all the notification to read
    if (list.isFetching || markAllRead.isPending) return;
    try {
      await markAllRead.mutateAsync();
    } catch (error) {
      console.error(error);
    }
  };

  return (
    <div className="relative flex items-center justify-center gap-2 text-body-xs-medium">
      {/* mark all notifications as read*/}
      <Tooltip tooltipContent={t("notification.options.mark_all_as_read")} isMobile={isMobile} position="bottom">
        <IconButton
          size="base"
          variant="ghost"
          icon={markAllRead.isPending ? Spinner : CheckCheck}
          onClick={() => {
            handleMarkAllNotificationsAsRead();
          }}
        />
      </Tooltip>

      {/* refetch current notifications */}
      <Tooltip tooltipContent={t("notification.options.refresh")} isMobile={isMobile} position="bottom">
        <IconButton
          size="base"
          variant="ghost"
          icon={RefreshCw}
          className={list.isRefetching ? "animate-spin" : ""}
          onClick={refreshNotifications}
        />
      </Tooltip>

      {/* notification filters */}
      <NotificationFilter />

      {/* notification menu options */}
      <NotificationHeaderMenuOption />
    </div>
  );
};
