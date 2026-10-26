/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createContext, useContext, useState } from "react";
import type { ReactNode } from "react";
import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ENotificationTab } from "@plane/constants";
import type { TNotificationTab } from "@plane/constants";
import type { TNotificationFilter } from "@plane/types";
import {
  emptyNotificationCounts,
  markAllNotificationsReadOptions,
  notificationCountOptions,
  notificationListOptions,
} from "@/queries/notifications";

const defaultFilters: TNotificationFilter = {
  type: { assigned: false, created: false, subscribed: false },
  snoozed: false,
  archived: false,
  read: false,
};

function useNotificationState(workspaceSlug: string) {
  const client = useQueryClient();
  const [currentNotificationTab, setCurrentNotificationTab] = useState<TNotificationTab>(ENotificationTab.ALL);
  const [currentSelectedNotificationId, setCurrentSelectedNotificationId] = useState<string>();
  const [filters, setFilters] = useState(defaultFilters);
  const list = useInfiniteQuery(notificationListOptions(workspaceSlug, currentNotificationTab, filters));
  const { data: unreadNotificationsCount = emptyNotificationCounts } = useQuery(
    notificationCountOptions(workspaceSlug)
  );
  const markAllRead = useMutation(markAllNotificationsReadOptions(client, workspaceSlug, filters));
  const notifications = new Map(list.data?.pages.flatMap((page) => page.results ?? []).map((item) => [item.id, item]));
  const notificationIds = [...notifications.values()]
    .filter((item) => {
      if (!!item.is_mentioned_notification !== (currentNotificationTab === ENotificationTab.MENTIONS)) return false;
      if (filters.read && item.read_at) return false;
      if (filters.snoozed) return !!item.snoozed_till;
      if (filters.archived) return !!item.archived_at;
      return !item.archived_at && !item.snoozed_till;
    })
    .map((item) => item.id);

  const updateBulkFilters = (patch: Partial<TNotificationFilter>) =>
    setFilters((previous) => ({ ...previous, ...patch }));
  const updateFilters = <K extends keyof TNotificationFilter>(key: K, value: TNotificationFilter[K]) =>
    updateBulkFilters({ [key]: value });
  return {
    workspaceSlug,
    list,
    notifications,
    notificationIds,
    unreadNotificationsCount,
    markAllRead,
    currentNotificationTab,
    setCurrentNotificationTab,
    currentSelectedNotificationId,
    setCurrentSelectedNotificationId,
    filters,
    updateFilters,
    updateBulkFilters,
  };
}

const NotificationsContext = createContext<ReturnType<typeof useNotificationState> | undefined>(undefined);

export function NotificationsProvider({ workspaceSlug, children }: { workspaceSlug: string; children: ReactNode }) {
  const value = useNotificationState(workspaceSlug);
  return <NotificationsContext.Provider value={value}>{children}</NotificationsContext.Provider>;
}

export function useNotifications() {
  const context = useContext(NotificationsContext);
  if (!context) throw new Error("useNotifications must be used within NotificationsProvider");
  return context;
}
