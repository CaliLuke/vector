/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { RefObject } from "react";
import { VirtualList } from "@/components/common/virtual-list";

// plane imports
import { useTranslation } from "@plane/i18n";
// components
import { NotificationItem } from "@/components/workspace-notifications/sidebar/notification-card/item";
// hooks
import { useNotifications } from "@/components/workspace-notifications/provider";

type TNotificationCardListRoot = {
  workspaceSlug: string;
  scrollRef: RefObject<HTMLDivElement | null>;
};

export const NotificationCardListRoot = function NotificationCardListRoot(props: TNotificationCardListRoot) {
  const { workspaceSlug, scrollRef } = props;
  // hooks
  const { list, notificationIds, currentNotificationTab, filters } = useNotifications();
  const { t } = useTranslation();

  return (
    <div>
      <VirtualList
        key={JSON.stringify([workspaceSlug, currentNotificationTab, filters])}
        items={notificationIds}
        getKey={(id) => id}
        scrollRef={scrollRef}
        estimateSize={112}
        label={t("notifications")}
        renderItem={(notificationId) => (
          <NotificationItem workspaceSlug={workspaceSlug} notificationId={notificationId} />
        )}
      />

      {/* fetch next page notifications */}
      {list.hasNextPage && (
        <>
          {list.isFetchingNextPage ? (
            <div className="flex items-center justify-center py-4 text-13 font-medium">
              <div className="text-accent-secondary">{t("loading")}...</div>
            </div>
          ) : (
            <button
              type="button"
              className="flex w-full items-center justify-center py-4 text-13 font-medium"
              onClick={() => {
                if (!list.isFetching) void list.fetchNextPage();
              }}
            >
              <div className="cursor-pointer text-accent-secondary transition-all hover:text-accent-primary">
                {t("load_more")}
              </div>
            </button>
          )}
        </>
      )}
    </div>
  );
};
