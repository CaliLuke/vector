/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { infiniteQueryOptions, mutationOptions } from "@tanstack/react-query";
import type { InfiniteData, QueryClient } from "@tanstack/react-query";
import { STICKIES_PER_PAGE } from "@plane/constants";
import type { TSticky } from "@plane/types";
import { StickyService } from "@/services/sticky.service";
import { captureIdentityScope, getCurrentUserId, isIdentityScopeCurrent, isIdentityScopeVerified } from "./identity";

const service = new StickyService();
export const stickyKeys = {
  lists: (workspaceSlug: string) => ["stickies", workspaceSlug, "list"] as const,
  mutations: (workspaceSlug: string) => ["stickies", workspaceSlug, "mutation"] as const,
  create: (workspaceSlug: string) => ["stickies", workspaceSlug, "mutation", "create"] as const,
};

type StickyPages = InfiniteData<Awaited<ReturnType<StickyService["getStickies"]>>, string>;
export function stickyListOptions(workspaceSlug: string, search: string) {
  return infiniteQueryOptions({
    queryKey: [...stickyKeys.lists(workspaceSlug), search],
    queryFn: ({ pageParam, signal }) =>
      service.getStickies(workspaceSlug, pageParam, search, STICKIES_PER_PAGE, signal),
    initialPageParam: `${STICKIES_PER_PAGE}:0:0`,
    getNextPageParam: (page) => (page.next_page_results ? page.next_cursor : undefined),
    staleTime: 30_000,
    refetchOnMount: true,
  });
}

export function createStickyOptions(client: QueryClient, workspaceSlug: string) {
  const identity = captureIdentityScope(client, getCurrentUserId(client));
  const queryKey = stickyKeys.lists(workspaceSlug);
  return mutationOptions({
    mutationKey: [...stickyKeys.create(workspaceSlug), identity?.userId, identity?.generation],
    mutationFn: (data: Partial<TSticky>) => {
      if (!isIdentityScopeVerified(client, identity)) throw new Error("The account session changed or is unverified.");
      return service.createSticky(workspaceSlug, data);
    },
    onMutate: () => {
      if (!isIdentityScopeVerified(client, identity)) throw new Error("The account session changed or is unverified.");
      return client.getQueryCache().findAll({ queryKey });
    },
    onSuccess: async (sticky, _input, queries) => {
      if (!isIdentityScopeCurrent(client, identity)) return;
      await client.cancelQueries({ queryKey });
      if (!isIdentityScopeCurrent(client, identity)) return;
      const listKey = stickyListOptions(workspaceSlug, "").queryKey;
      const list = client.getQueryCache().find({ queryKey: listKey, exact: true });
      if (list && queries?.includes(list))
        client.setQueryData<StickyPages>(
          stickyListOptions(workspaceSlug, "").queryKey,
          (data) =>
            data && {
              ...data,
              pages: data.pages.map((page, index) =>
                index === 0
                  ? { ...page, results: [sticky, ...page.results.filter((item) => item.id !== sticky.id)] }
                  : page
              ),
            }
        );
      await Promise.all(
        (queries ?? [])
          .filter((query) => client.getQueryCache().find({ queryKey: query.queryKey, exact: true }) === query)
          .map((query) => client.invalidateQueries({ queryKey: query.queryKey, exact: true }))
      );
    },
  });
}

type StickyChange = { kind: "update"; id: string; patch: Partial<TSticky> } | { kind: "remove"; id: string };
export function changeStickyOptions(client: QueryClient, workspaceSlug: string, stickyId: string) {
  const queryKey = stickyKeys.lists(workspaceSlug);
  const identity = captureIdentityScope(client, getCurrentUserId(client));
  return mutationOptions({
    mutationKey: [...stickyKeys.mutations(workspaceSlug), identity?.userId, identity?.generation, stickyId],
    scope: { id: JSON.stringify(["sticky", identity?.userId, identity?.generation, workspaceSlug, stickyId]) },
    mutationFn: async (change: StickyChange) => {
      if (!isIdentityScopeVerified(client, identity)) throw new Error("The account session changed or is unverified.");
      if (change.kind === "update") return service.updateSticky(workspaceSlug, change.id, change.patch);
      await service.deleteSticky(workspaceSlug, change.id);
      return undefined;
    },
    onMutate: async (change) => {
      if (!isIdentityScopeVerified(client, identity)) throw new Error("The account session changed or is unverified.");
      await client.cancelQueries({ queryKey });
      if (!isIdentityScopeVerified(client, identity)) throw new Error("The account session changed or is unverified.");
      const snapshots = client
        .getQueryCache()
        .findAll({ queryKey })
        .map((query) => ({
          query,
          data: client.getQueryData<StickyPages>(query.queryKey),
        }));
      if (change.kind === "update")
        client.setQueriesData<StickyPages>(
          { queryKey },
          (data) =>
            data && {
              ...data,
              pages: data.pages.map((page) => ({
                ...page,
                results: page.results.map((sticky) =>
                  sticky.id === change.id ? { ...sticky, ...change.patch } : sticky
                ),
              })),
            }
        );
      return snapshots;
    },
    onSuccess: (_result, change, snapshots) => {
      if (change.kind !== "remove" || !isIdentityScopeCurrent(client, identity)) return;
      for (const { query } of snapshots ?? []) {
        if (client.getQueryCache().find({ queryKey: query.queryKey, exact: true }) !== query) continue;
        client.setQueryData<StickyPages>(
          query.queryKey,
          (data) =>
            data && {
              ...data,
              pages: data.pages.map((page) => ({
                ...page,
                results: page.results.filter((sticky) => sticky.id !== change.id),
              })),
            }
        );
      }
    },
    onError: (_error, change, snapshots) => {
      if (change.kind === "remove" || !isIdentityScopeCurrent(client, identity)) return;
      for (const { query, data: snapshot } of snapshots ?? []) {
        const key = query.queryKey;
        if (client.getQueryCache().find({ queryKey: key, exact: true }) !== query) continue;
        const previous = snapshot?.pages.flatMap((page) => page.results).find((sticky) => sticky.id === change.id);
        if (!previous) continue;
        client.setQueryData<StickyPages>(key, (data) => {
          if (!data) return undefined;
          return {
            ...data,
            pages: data.pages.map((page) => ({
              ...page,
              results: page.results.map((sticky) => {
                if (sticky.id !== change.id) return sticky;
                const currentValues = new Map(Object.entries(sticky));
                const previousValues = new Map(Object.entries(previous));
                const restored = Object.fromEntries(
                  Object.entries(change.patch)
                    .filter(([field, value]) => currentValues.get(field) === value)
                    .map(([field]) => [field, previousValues.get(field)])
                );
                return { ...sticky, ...restored };
              }),
            })),
          };
        });
      }
    },
    onSettled: async (_data, _error, _change, snapshots) => {
      if (
        !isIdentityScopeCurrent(client, identity) ||
        client.isMutating({
          mutationKey: [...stickyKeys.mutations(workspaceSlug), identity?.userId, identity?.generation],
        }) !== 1
      )
        return;
      await Promise.all(
        (snapshots ?? [])
          .filter(({ query }) => client.getQueryCache().find({ queryKey: query.queryKey, exact: true }) === query)
          .map(({ query }) => client.invalidateQueries({ queryKey: query.queryKey, exact: true }))
      );
    },
  });
}
