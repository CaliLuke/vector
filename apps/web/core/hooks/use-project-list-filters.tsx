/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import type { PropsWithChildren } from "react";
import type { TProjectAppliedDisplayFilterKeys, TProjectDisplayFilters, TProjectFilters } from "@plane/types";

type TProjectListFilters = {
  displayFilters: TProjectDisplayFilters;
  filters: TProjectFilters;
  searchQuery: string;
  updateDisplayFilters: (patch: Partial<TProjectDisplayFilters>) => void;
  updateFilters: (patch: Partial<TProjectFilters>) => void;
  updateSearchQuery: (query: string) => void;
  clearAllFilters: () => void;
  clearAllAppliedDisplayFilters: () => void;
  appliedDisplayFilters: TProjectAppliedDisplayFilterKeys[];
};

const ProjectListFiltersContext = createContext<TProjectListFilters | undefined>(undefined);
const appliedDisplayFilterKeys: readonly TProjectAppliedDisplayFilterKeys[] = ["my_projects", "archived_projects"];

function defaultFilters(): Pick<TProjectListFilters, "displayFilters" | "filters" | "searchQuery"> {
  return { displayFilters: { order_by: "created_at" }, filters: {}, searchQuery: "" };
}

export function ProjectListFiltersProvider({ children, workspaceSlug }: PropsWithChildren<{ workspaceSlug: string }>) {
  const [workspaces, setWorkspaces] = useState<Record<string, ReturnType<typeof defaultFilters>>>({});
  const previousWorkspaceSlug = useRef<string | undefined>(undefined);
  useEffect(() => {
    if (workspaceSlug && previousWorkspaceSlug.current !== workspaceSlug) {
      setWorkspaces((current) => {
        const state = current[workspaceSlug] ?? defaultFilters();
        return {
          ...current,
          [workspaceSlug]: {
            ...state,
            displayFilters: { order_by: state.displayFilters.order_by || "created_at" },
            searchQuery: "",
          },
        };
      });
    }
    previousWorkspaceSlug.current = workspaceSlug;
  }, [workspaceSlug]);
  const state = workspaces[workspaceSlug] ?? defaultFilters();
  const updateWorkspace = useCallback(
    (update: (current: ReturnType<typeof defaultFilters>) => ReturnType<typeof defaultFilters>) =>
      setWorkspaces((current) => ({ ...current, [workspaceSlug]: update(current[workspaceSlug] ?? defaultFilters()) })),
    [workspaceSlug]
  );
  const updateDisplayFilters = useCallback(
    (patch: Partial<TProjectDisplayFilters>) =>
      updateWorkspace((current) => ({ ...current, displayFilters: { ...current.displayFilters, ...patch } })),
    [updateWorkspace]
  );
  const updateFilters = useCallback(
    (patch: Partial<TProjectFilters>) =>
      updateWorkspace((current) => ({ ...current, filters: { ...current.filters, ...patch } })),
    [updateWorkspace]
  );
  const updateSearchQuery = useCallback(
    (searchQuery: string) => updateWorkspace((current) => ({ ...current, searchQuery })),
    [updateWorkspace]
  );
  const clearAllFilters = useCallback(
    () => updateWorkspace((current) => ({ ...current, filters: {} })),
    [updateWorkspace]
  );
  const clearAllAppliedDisplayFilters = useCallback(
    () =>
      updateWorkspace((current) => ({
        ...current,
        displayFilters: { ...current.displayFilters, my_projects: false, archived_projects: false },
      })),
    [updateWorkspace]
  );
  const value = useMemo<TProjectListFilters>(() => {
    return {
      ...state,
      updateDisplayFilters,
      updateFilters,
      updateSearchQuery,
      clearAllFilters,
      clearAllAppliedDisplayFilters,
      appliedDisplayFilters: appliedDisplayFilterKeys.filter((key) => Boolean(state.displayFilters[key])),
    };
  }, [clearAllAppliedDisplayFilters, clearAllFilters, state, updateDisplayFilters, updateFilters, updateSearchQuery]);
  return <ProjectListFiltersContext.Provider value={value}>{children}</ProjectListFiltersContext.Provider>;
}

export function useProjectListFilters() {
  const value = useContext(ProjectListFiltersContext);
  if (!value) throw new Error("useProjectListFilters must be used within ProjectListFiltersProvider");
  return value;
}
