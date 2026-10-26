/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { mutationOptions, queryOptions } from "@tanstack/react-query";
import type { QueryClient } from "@tanstack/react-query";
import type { TLink, TWidgetEntityData } from "@plane/types";
import { WorkspaceService } from "@/services/workspace.service";

const service = new WorkspaceService();
export const homeKeys = {
  widgets: (workspaceSlug: string) => ["home", workspaceSlug, "widgets"] as const,
  links: (workspaceSlug: string) => ["home", workspaceSlug, "links"] as const,
};

export function homeWidgetsOptions(workspaceSlug: string) {
  return queryOptions({
    queryKey: homeKeys.widgets(workspaceSlug),
    queryFn: ({ signal }) => service.fetchWorkspaceWidgets(workspaceSlug, signal),
    staleTime: 30_000,
    refetchOnMount: true,
  });
}

export function homeLinksOptions(workspaceSlug: string) {
  return queryOptions({
    queryKey: homeKeys.links(workspaceSlug),
    queryFn: ({ signal }) => service.fetchWorkspaceLinks(workspaceSlug, signal),
    staleTime: 30_000,
    refetchOnMount: true,
  });
}

export function updateHomeWidgetOptions(client: QueryClient, workspaceSlug: string) {
  const queryKey = homeKeys.widgets(workspaceSlug);
  return mutationOptions({
    mutationKey: queryKey,
    mutationFn: ({ key, patch }: { key: string; patch: Partial<TWidgetEntityData> }) =>
      service.updateWorkspaceWidget(workspaceSlug, key, patch),
    onMutate: async ({ key, patch }) => {
      await client.cancelQueries({ queryKey });
      const previous = client.getQueryData<TWidgetEntityData[]>(queryKey)?.find((widget) => widget.key === key);
      client.setQueryData<TWidgetEntityData[]>(queryKey, (widgets) =>
        widgets?.map((widget) => (widget.key === key ? { ...widget, ...patch } : widget))
      );
      return previous;
    },
    onError: (_error, { key }, previous) => {
      if (previous)
        client.setQueryData<TWidgetEntityData[]>(queryKey, (widgets) =>
          widgets?.map((widget) => (widget.key === key ? previous : widget))
        );
    },
    onSettled: () =>
      client.isMutating({ mutationKey: queryKey }) === 1 ? client.invalidateQueries({ queryKey }) : undefined,
  });
}

type LinkChange =
  | { kind: "create"; data: Partial<TLink> }
  | { kind: "update"; id: string; data: Partial<TLink> }
  | { kind: "remove"; id: string };
export function homeLinkMutationOptions(client: QueryClient, workspaceSlug: string) {
  const queryKey = homeKeys.links(workspaceSlug);
  return mutationOptions({
    mutationKey: queryKey,
    mutationFn: async (change: LinkChange) => {
      if (change.kind === "create") return service.createWorkspaceLink(workspaceSlug, change.data);
      if (change.kind === "update") return service.updateWorkspaceLink(workspaceSlug, change.id, change.data);
      await service.deleteWorkspaceLink(workspaceSlug, change.id);
      return undefined;
    },
    onSuccess: async (link, change) => {
      await client.cancelQueries({ queryKey });
      client.setQueryData<TLink[]>(queryKey, (links) => {
        if (!links) return undefined;
        if (change.kind === "remove") return links.filter((item) => item.id !== change.id);
        if (!link) return links;
        return change.kind === "create"
          ? [link, ...links.filter((item) => item.id !== link.id)]
          : links.map((item) => (item.id === link.id ? link : item));
      });
      await client.invalidateQueries({ queryKey });
    },
  });
}
