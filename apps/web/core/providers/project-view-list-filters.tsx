/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors.
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createContext, useCallback, useContext, useMemo, useState } from "react";
import type { PropsWithChildren } from "react";
import type { TViewFilters } from "@plane/types";
import { useQueryClient } from "@tanstack/react-query";
import { useCurrentUser } from "@/hooks/use-current-user";
import { useIdentityGeneration } from "@/hooks/use-identity-generation";

type TProjectViewListFilters = {
  filters: TViewFilters;
  updateFilters: <T extends keyof TViewFilters>(key: T, value: TViewFilters[T]) => void;
  clearAllFilters: () => void;
};

const ProjectViewListFiltersContext = createContext<TProjectViewListFilters | undefined>(undefined);

const initialFilters: TViewFilters = { searchQuery: "", sortBy: "desc", sortKey: "updated_at" };

export function ProjectViewListFiltersProvider({
  workspaceSlug,
  projectId,
  children,
}: PropsWithChildren<{ workspaceSlug: string | undefined; projectId: string | undefined }>) {
  const client = useQueryClient();
  const user = useCurrentUser();
  const generation = useIdentityGeneration(client);
  const scopeKey = JSON.stringify([user.data?.id, generation, workspaceSlug, projectId]);
  return <ScopedProjectViewListFilters key={scopeKey}>{children}</ScopedProjectViewListFilters>;
}

function ScopedProjectViewListFilters({ children }: PropsWithChildren) {
  const [filters, setFilters] = useState(initialFilters);
  const updateFilters = useCallback<TProjectViewListFilters["updateFilters"]>((key, value) => {
    setFilters((current) => ({ ...current, [key]: value }));
  }, []);
  const clearAllFilters = useCallback(() => {
    setFilters((current) => ({ ...current, filters: {} }));
  }, []);
  const value = useMemo(() => ({ filters, updateFilters, clearAllFilters }), [clearAllFilters, filters, updateFilters]);
  return <ProjectViewListFiltersContext.Provider value={value}>{children}</ProjectViewListFiltersContext.Provider>;
}

export function useProjectViewListFilters() {
  const value = useContext(ProjectViewListFiltersContext);
  if (!value) throw new Error("Project view list filters need their scoped provider.");
  return value;
}
