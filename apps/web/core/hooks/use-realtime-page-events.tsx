/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TPage } from "@plane/types";
import { useRouter } from "@tanstack/react-router";
import { useCallback, useMemo } from "react";
import type { EventToPayloadMap } from "@plane/editor";
import { setToast, TOAST_TYPE } from "@plane/propel/toast";
import { EPageAccess } from "@plane/types";
import { useQueryClient } from "@tanstack/react-query";
import type { TEditorBodyHandlers } from "@/components/pages/editor/editor-body";
import { usePageRecord } from "@/components/pages/page-record-context";
import { useCurrentUser } from "@/hooks/use-current-user";
import { captureWorkspaceFavoritesRefresh } from "@/queries/favorites";
import { removeProjectPageCache, updateProjectPageCache } from "@/queries/pages";

export type PageUpdateHandler<T extends keyof EventToPayloadMap = keyof EventToPayloadMap> = (params: {
  pageIds: string[];
  data: EventToPayloadMap[T];
  performAction: boolean;
}) => void;

export type TCustomEventHandlers = { [K in keyof EventToPayloadMap]?: PageUpdateHandler<K> };

interface UsePageEventsProps {
  page: TPage;
  getMemberDisplayName: (userId: string) => string | undefined;
  customRealtimeEventHandlers?: TCustomEventHandlers;
  handlers: TEditorBodyHandlers;
}

export const useRealtimePageEvents = ({
  page,
  getMemberDisplayName,
  customRealtimeEventHandlers,
  handlers,
}: UsePageEventsProps) => {
  const router = useRouter();
  const client = useQueryClient();
  const { scope } = usePageRecord();
  const { data: currentUser } = useCurrentUser();
  const pageId = page.id;
  const getUserDisplayText = useCallback(
    (userId: string | undefined) => {
      if (!userId) return "";
      const displayName = getMemberDisplayName(userId);
      return displayName ? ` by ${displayName}` : "";
    },
    [getMemberDisplayName]
  );

  const updatePage = useCallback(
    (ids: string[], change: (current: TPage) => TPage) => {
      ids.forEach((id) => updateProjectPageCache(client, scope, id, change));
    },
    [client, scope]
  );

  const ACTION_HANDLERS = useMemo<TCustomEventHandlers>(
    () => ({
      archived: ({ pageIds, data }: { pageIds: string[]; data: EventToPayloadMap["archived"] }) =>
        updatePage(pageIds, (current) => ({ ...current, archived_at: data.archived_at })),
      unarchived: ({ pageIds }: { pageIds: string[] }) =>
        updatePage(pageIds, (current) => ({ ...current, archived_at: null })),
      locked: ({ pageIds }: { pageIds: string[] }) =>
        updatePage(pageIds, (current) => ({ ...current, is_locked: true })),
      unlocked: ({ pageIds }: { pageIds: string[] }) =>
        updatePage(pageIds, (current) => ({ ...current, is_locked: false })),
      "made-public": ({ pageIds }: { pageIds: string[] }) =>
        updatePage(pageIds, (current) => ({ ...current, access: EPageAccess.PUBLIC })),
      "made-private": ({ pageIds }: { pageIds: string[] }) =>
        updatePage(pageIds, (current) => ({ ...current, access: EPageAccess.PRIVATE })),
      deleted: ({ pageIds, data }: { pageIds: string[]; data: EventToPayloadMap["deleted"] }) => {
        pageIds.forEach((id) => removeProjectPageCache(client, scope, id));
        if (scope.workspaceSlug) void captureWorkspaceFavoritesRefresh(client, scope.workspaceSlug)();
        if (pageId && pageIds.includes(pageId)) {
          if (data?.user_id !== currentUser?.id) {
            setToast({
              type: TOAST_TYPE.ERROR,
              title: "Page deleted",
              message: `Page deleted${getUserDisplayText(data.user_id)}`,
            });
          }
          void router.navigate({ href: handlers.getRedirectionLink() });
        }
      },
      property_updated: ({ pageIds, data }: { pageIds: string[]; data: EventToPayloadMap["property_updated"] }) =>
        // The collaboration service persists titles; local and remote editor events only update the visible cache.
        updatePage(pageIds, (current) => ({ ...current, ...data })),
      error: ({ pageIds, data }: { pageIds: string[]; data: EventToPayloadMap["error"] }) => {
        if (!pageId || !pageIds.includes(pageId)) return;
        setToast({
          type: TOAST_TYPE.ERROR,
          title: data.error_type === "fetch" ? "Failed to load page" : "Failed to save page",
          message: data.error_message || "An error occurred",
        });
        if (data.error_code === "page_locked") updatePage([pageId], (current) => ({ ...current, is_locked: true }));
        if (data.error_code === "page_archived")
          updatePage([pageId], (current) => ({
            ...current,
            archived_at: current.archived_at ?? new Date().toISOString(),
          }));
      },
      ...customRealtimeEventHandlers,
    }),
    [
      client,
      currentUser?.id,
      getUserDisplayText,
      handlers,
      pageId,
      router,
      scope,
      updatePage,
      customRealtimeEventHandlers,
    ]
  );

  const updatePageProperties = useCallback(
    <T extends keyof EventToPayloadMap>(
      pageIds: string | string[],
      actionType: T,
      data: EventToPayloadMap[T],
      performAction = false
    ) => {
      const normalizedPageIds = Array.isArray(pageIds) ? pageIds : [pageIds];
      if (!normalizedPageIds.length) return;
      const handler = ACTION_HANDLERS[actionType];
      handler?.({ pageIds: normalizedPageIds, data, performAction });
    },
    [ACTION_HANDLERS]
  );

  return { updatePageProperties };
};
