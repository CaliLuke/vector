/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import { useCallback } from "react";

// plane imports
import type { TNotificationTab } from "@plane/constants";
import { NOTIFICATION_TABS } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { Header, Row, ERowVariant, EHeaderVariant, ContentWrapper } from "@plane/ui";
import { cn } from "@plane/utils/common";
import { getNumberCount } from "@plane/utils/string";
// components
import { CountChip } from "@/components/common/count-chip";
// hooks
import { useNotifications } from "@/components/workspace-notifications/provider";
import { useWorkspaceBySlug } from "@/hooks/use-workspaces";
// local imports
import { NotificationEmptyState } from "./empty-state";
import { AppliedFilters } from "./filters/applied-filter";
import { NotificationSidebarHeader } from "./header";
import { NotificationsLoader } from "./loader";
import { NotificationCardListRoot } from "../notification-card/root";

export const NotificationsSidebarRoot = function NotificationsSidebarRoot() {
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  // hooks
  const { data: workspace } = useWorkspaceBySlug(workspaceSlug);
  const {
    currentSelectedNotificationId,
    unreadNotificationsCount,
    list,
    notificationIds,
    currentNotificationTab,
    setCurrentNotificationTab,
  } = useNotifications();

  const { t } = useTranslation();

  // derived values

  const handleTabClick = useCallback(
    (tabValue: TNotificationTab) => {
      if (currentNotificationTab !== tabValue) {
        setCurrentNotificationTab(tabValue);
      }
    },
    [currentNotificationTab, setCurrentNotificationTab]
  );

  if (!workspaceSlug || !workspace) return <></>;

  return (
    <div
      className={cn(
        "relative z-[10] h-full flex-shrink-0 border-0 border-subtle bg-surface-1 transition-all max-md:overflow-hidden md:border-r",
        currentSelectedNotificationId ? "w-0 md:w-3/12" : "w-full md:w-3/12"
      )}
    >
      <div className="relative flex h-full w-full flex-col">
        <Row className="flex h-header flex-shrink-0 border-b border-subtle">
          <NotificationSidebarHeader workspaceSlug={workspaceSlug.toString()} />
        </Row>

        <Header variant={EHeaderVariant.SECONDARY} className="justify-start">
          {NOTIFICATION_TABS.map((tab) => (
            // oxlint-disable-next-line jsx_a11y/click-events-have-key-events oxlint-disable-next-line jsx_a11y/no-static-element-interactions
            <div
              key={tab.value}
              className="relative h-full cursor-pointer px-3"
              onClick={() => handleTabClick(tab.value)}
            >
              <div
                className={cn(
                  "relative flex h-full items-center justify-center gap-1 text-body-xs-medium transition-all",
                  {
                    "text-accent-primary": currentNotificationTab === tab.value,
                    "text-primary hover:text-secondary": currentNotificationTab !== tab.value,
                  }
                )}
              >
                <div className="font-medium">{t(tab.i18n_label)}</div>
                {tab.count(unreadNotificationsCount) > 0 && (
                  <CountChip count={getNumberCount(tab.count(unreadNotificationsCount))} />
                )}
              </div>
              {currentNotificationTab === tab.value && (
                <div className="absolute right-0 bottom-0 left-0 rounded-t-md border border-accent-strong" />
              )}
            </div>
          ))}
        </Header>

        {/* applied filters */}
        <AppliedFilters workspaceSlug={workspaceSlug.toString()} />

        {/* rendering notifications */}
        {list.isPending ? (
          <div className="relative h-full w-full overflow-hidden">
            <NotificationsLoader />
          </div>
        ) : list.isError ? (
          <div role="alert" className="p-4 text-body-sm-regular">
            <p>Unable to load notifications.</p>
            <button
              type="button"
              onClick={() => {
                void list.refetch();
              }}
            >
              Try again
            </button>
          </div>
        ) : (
          <>
            {notificationIds && notificationIds.length > 0 ? (
              <ContentWrapper variant={ERowVariant.HUGGING}>
                <NotificationCardListRoot workspaceSlug={workspaceSlug.toString()} />
              </ContentWrapper>
            ) : (
              <div className="relative flex h-full w-full items-center justify-center">
                <NotificationEmptyState currentNotificationTab={currentNotificationTab} />
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
};
