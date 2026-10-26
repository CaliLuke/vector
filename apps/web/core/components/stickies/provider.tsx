/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import orderBy from "lodash-es/orderBy.js";
import { createContext, useContext, useState } from "react";
import type { ReactNode } from "react";
import { useInfiniteQuery, useIsMutating, useQueryClient } from "@tanstack/react-query";
import { stickyKeys, stickyListOptions } from "@/queries/stickies";
import useDebounce from "@/hooks/use-debounce";
import { useCurrentUserId } from "@/hooks/use-current-user";
import { useIdentityGeneration } from "@/hooks/use-identity-generation";

function useStickyState(workspaceSlug: string) {
  const [searchQuery, setSearchQuery] = useState("");
  const accountId = useCurrentUserId();
  const client = useQueryClient();
  const generation = useIdentityGeneration(client);
  const search = useDebounce(searchQuery, 500);
  const list = useInfiniteQuery(stickyListOptions(workspaceSlug, search));
  const stickies = new Map(list.data?.pages.flatMap((page) => page.results).map((sticky) => [sticky.id, sticky]));
  const stickyIds = orderBy([...stickies.values()], "sort_order", "desc").map((sticky) => sticky.id);
  const creatingSticky =
    useIsMutating({ mutationKey: [...stickyKeys.create(workspaceSlug), accountId, generation] }) > 0;
  return { list, stickies, stickyIds, searchQuery, setSearchQuery, creatingSticky };
}
const StickiesContext = createContext<ReturnType<typeof useStickyState> | undefined>(undefined);

export function StickiesProvider({ workspaceSlug, children }: { workspaceSlug: string; children: ReactNode }) {
  const value = useStickyState(workspaceSlug);
  return <StickiesContext.Provider value={value}>{children}</StickiesContext.Provider>;
}
export function useStickies() {
  const context = useContext(StickiesContext);
  if (!context) throw new Error("useStickies must be used within StickiesProvider");
  return context;
}
