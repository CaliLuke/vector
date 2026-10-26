/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import {
  infiniteQueryOptions,
  mutationOptions,
  queryOptions,
  useIsMutating,
  useMutation,
  useQueryClient,
} from "@tanstack/react-query";
import type { InfiniteData, QueryClient } from "@tanstack/react-query";
import { ENotificationTab } from "@plane/constants";
import type { TNotificationTab } from "@plane/constants";
import type {
  TNotification,
  TNotificationFilter,
  TNotificationPaginatedInfo,
  TUnreadNotificationsCount,
} from "@plane/types";
import service from "@/services/workspace-notification.service";

export const notificationKeys = {
  workspace: (workspaceSlug: string) => ["notifications", workspaceSlug] as const,
  lists: (workspaceSlug: string) => ["notifications", workspaceSlug, "list"] as const,
  count: (workspaceSlug: string) => ["notifications", workspaceSlug, "count"] as const,
  mutations: (workspaceSlug: string) => ["notifications", workspaceSlug, "mutation"] as const,
};

export const emptyNotificationCounts: TUnreadNotificationsCount = {
  total_unread_notifications_count: 0,
  mention_unread_notifications_count: 0,
};

export function notificationCountOptions(workspaceSlug: string) {
  return queryOptions({
    queryKey: notificationKeys.count(workspaceSlug),
    queryFn: ({ signal }) => service.fetchUnreadNotificationsCount(workspaceSlug, signal),
    staleTime: 30_000,
    refetchOnMount: true,
  });
}

function notificationFilterParams(filters: TNotificationFilter) {
  return {
    type:
      Object.entries(filters.type)
        .filter(([, selected]) => selected)
        .map(([key]) => key)
        .join(",") || undefined,
    snoozed: filters.snoozed,
    archived: filters.archived,
    read: filters.read ? false : undefined,
  };
}

export function notificationListOptions(workspaceSlug: string, tab: TNotificationTab, filters: TNotificationFilter) {
  const params = { ...notificationFilterParams(filters), mentioned: tab === ENotificationTab.MENTIONS || undefined };
  return infiniteQueryOptions({
    queryKey: [...notificationKeys.lists(workspaceSlug), params],
    queryFn: ({ pageParam, signal }) =>
      service.fetchNotifications(workspaceSlug, { ...params, per_page: 300, cursor: pageParam }, signal),
    initialPageParam: "300:0:0",
    getNextPageParam: (page) => (page.next_page_results ? page.next_cursor : undefined),
    staleTime: 30_000,
    refetchOnMount: true,
  });
}

type NotificationPages = InfiniteData<TNotificationPaginatedInfo, string>;
type NotificationChange = { notification: TNotification } & (
  | { field: "read_at" | "archived_at"; value: string | null }
  | { field: "snoozed_till"; value: string | null }
);

function updateCachedNotification(
  client: QueryClient,
  workspaceSlug: string,
  id: string,
  patch: Partial<TNotification>
) {
  client.setQueriesData<NotificationPages>(
    { queryKey: notificationKeys.lists(workspaceSlug) },
    (data) =>
      data && {
        ...data,
        pages: data.pages.map((page) => ({
          ...page,
          results: page.results?.map((item) => (item.id === id ? { ...item, ...patch } : item)),
        })),
      }
  );
}

function adjustUnreadCount(client: QueryClient, workspaceSlug: string, notification: TNotification, delta: number) {
  const field = notification.is_mentioned_notification
    ? "mention_unread_notifications_count"
    : "total_unread_notifications_count";
  client.setQueryData<TUnreadNotificationsCount>(
    notificationKeys.count(workspaceSlug),
    (counts) =>
      counts && {
        ...counts,
        [field]: Math.max(0, counts[field] + delta),
      }
  );
}

function changeNotification(workspaceSlug: string, { notification, field, value }: NotificationChange) {
  if (field === "read_at")
    return value
      ? service.markNotificationAsRead(workspaceSlug, notification.id)
      : service.markNotificationAsUnread(workspaceSlug, notification.id);
  if (field === "archived_at")
    return value
      ? service.markNotificationAsArchived(workspaceSlug, notification.id)
      : service.markNotificationAsUnArchived(workspaceSlug, notification.id);
  return service.updateNotificationById(workspaceSlug, notification.id, { snoozed_till: value });
}

export function notificationMutationOptions(client: QueryClient, workspaceSlug: string, notificationId: string) {
  return mutationOptions({
    mutationKey: [...notificationKeys.mutations(workspaceSlug), notificationId],
    mutationFn: (change: NotificationChange) => changeNotification(workspaceSlug, change),
    onMutate: async ({ notification, field, value }: NotificationChange) => {
      await client.cancelQueries({ queryKey: notificationKeys.workspace(workspaceSlug) });
      const previousValue = notification[field];
      const delta = field === "read_at" ? Number(!!previousValue) - Number(!!value) : 0;
      updateCachedNotification(client, workspaceSlug, notification.id, { [field]: value });
      if (delta) adjustUnreadCount(client, workspaceSlug, notification, delta);
      return { previousValue, delta };
    },
    onError: (_error, { notification, field }, context) => {
      if (!context) return;
      updateCachedNotification(client, workspaceSlug, notification.id, { [field]: context.previousValue });
      if (context.delta) adjustUnreadCount(client, workspaceSlug, notification, -context.delta);
    },
    onSuccess: (notification, { field }) => {
      if (notification)
        updateCachedNotification(client, workspaceSlug, notification.id, { [field]: notification[field] });
    },
    onSettled: () => {
      if (client.isMutating({ mutationKey: notificationKeys.mutations(workspaceSlug) }) === 1) {
        return client.invalidateQueries({ queryKey: notificationKeys.workspace(workspaceSlug) });
      }
      return undefined;
    },
  });
}

export function markAllNotificationsReadOptions(
  client: QueryClient,
  workspaceSlug: string,
  filters: TNotificationFilter
) {
  return mutationOptions({
    mutationKey: notificationKeys.mutations(workspaceSlug),
    mutationFn: () => service.markAllNotificationsAsRead(workspaceSlug, notificationFilterParams(filters)),
    onSuccess: async () => {
      await client.cancelQueries({ queryKey: notificationKeys.workspace(workspaceSlug) });
      await client.invalidateQueries({ queryKey: notificationKeys.workspace(workspaceSlug) });
    },
  });
}

export function useUpdateNotification(workspaceSlug: string, notificationId: string) {
  const options = notificationMutationOptions(useQueryClient(), workspaceSlug, notificationId);
  const mutation = useMutation(options);
  const isPending = useIsMutating({ mutationKey: options.mutationKey }) > 0;
  return { ...mutation, isPending };
}
