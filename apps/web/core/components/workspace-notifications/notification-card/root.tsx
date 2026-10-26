/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// plane imports
import { useTranslation } from "@plane/i18n";
// components
import { NotificationItem } from "@/components/workspace-notifications/sidebar/notification-card/item";
// hooks
import { useNotifications } from "@/components/workspace-notifications/provider";

type TNotificationCardListRoot = {
  workspaceSlug: string;
};

export const NotificationCardListRoot = function NotificationCardListRoot(props: TNotificationCardListRoot) {
  const { workspaceSlug } = props;
  // hooks
  const { list, notificationIds } = useNotifications();
  const { t } = useTranslation();

  return (
    <div>
      {notificationIds.map((notificationId: string) => (
        <NotificationItem key={notificationId} workspaceSlug={workspaceSlug} notificationId={notificationId} />
      ))}

      {/* fetch next page notifications */}
      {list.hasNextPage && (
        <>
          {list.isFetchingNextPage ? (
            <div className="flex items-center justify-center py-4 text-13 font-medium">
              <div className="text-accent-secondary">{t("loading")}...</div>
            </div>
          ) : (
            // oxlint-disable-next-line jsx_a11y/click-events-have-key-events oxlint-disable-next-line jsx_a11y/no-static-element-interactions
            <div
              className="flex items-center justify-center py-4 text-13 font-medium"
              onClick={() => {
                if (!list.isFetching) void list.fetchNextPage();
              }}
            >
              <div className="cursor-pointer text-accent-secondary transition-all hover:text-accent-primary">
                {t("load_more")}
              </div>
            </div>
          )}
        </>
      )}
    </div>
  );
};
