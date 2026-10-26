/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createContext, useCallback, useContext, useMemo, useState } from "react";
import type { EditorRefApi, TEditorAsset } from "@plane/editor";
import type { TPage, TPageFilters, TPageNavigationTabs } from "@plane/types";
import { useProjectPage, useProjectPages } from "@/hooks/use-project-pages";
import { pageFilterList } from "@/queries/pages";
import type { TProjectPageScope } from "@/queries/pages";

type TPageRecordContext = {
  page: TPage | undefined;
  pageQuery: ReturnType<typeof useProjectPage>;
  pages: TPage[];
  pageListQuery: ReturnType<typeof useProjectPages>;
  scope: TProjectPageScope;
  editorRef: EditorRefApi | null;
  setEditorRef: (value: EditorRefApi | null) => void;
  assets: TEditorAsset[];
  setAssets: (value: TEditorAsset[]) => void;
  syncStatus: "syncing" | "synced" | "error";
  setSyncStatus: (value: "syncing" | "synced" | "error") => void;
};

const PageRecordContext = createContext<TPageRecordContext | undefined>(undefined);

export function PageRecordProvider({
  workspaceSlug,
  projectId,
  pageId,
  children,
}: {
  workspaceSlug: string;
  projectId: string;
  pageId: string;
  children: React.ReactNode;
}) {
  const pageQuery = useProjectPage(workspaceSlug, projectId, pageId);
  const pageListQuery = useProjectPages(workspaceSlug, projectId);
  const [editorRef, setEditorRef] = useState<EditorRefApi | null>(null);
  const [assets, setAssets] = useState<TEditorAsset[]>([]);
  const [syncStatus, setSyncStatus] = useState<"syncing" | "synced" | "error">("syncing");
  const value = useMemo<TPageRecordContext>(
    () => ({
      page: pageQuery.isError || !pageQuery.access.isReady ? undefined : pageQuery.data,
      pageQuery,
      pages: pageListQuery.isError || !pageListQuery.access.isReady ? [] : (pageListQuery.data ?? []),
      pageListQuery,
      scope: pageQuery.access.scope,
      editorRef,
      setEditorRef,
      assets,
      setAssets,
      syncStatus,
      setSyncStatus,
    }),
    [pageQuery, pageListQuery, editorRef, assets, syncStatus]
  );
  return <PageRecordContext.Provider value={value}>{children}</PageRecordContext.Provider>;
}

export function usePageRecord() {
  const context = useContext(PageRecordContext);
  if (!context) throw new Error("usePageRecord must be used within PageRecordProvider");
  return context;
}

export function useOptionalPageRecord() {
  return useContext(PageRecordContext);
}

type TPageListContext = {
  projectId: string;
  pages: TPage[];
  visiblePages: TPage[];
  isPending: boolean;
  isError: boolean;
  isDenied: boolean;
  error: Error | null;
  refetch: () => Promise<unknown>;
  filters: TPageFilters;
  updateFilters: <K extends keyof TPageFilters>(key: K, value: TPageFilters[K]) => void;
  clearAllFilters: () => void;
};

const PageListContext = createContext<TPageListContext | undefined>(undefined);

const DEFAULT_PAGE_FILTERS: TPageFilters = { searchQuery: "", sortKey: "updated_at", sortBy: "desc" };

export function PageListProvider({
  workspaceSlug,
  projectId,
  pageType,
  children,
}: {
  workspaceSlug: string;
  projectId: string;
  pageType: TPageNavigationTabs;
  children: React.ReactNode;
}) {
  const query = useProjectPages(workspaceSlug, projectId);
  const scopeKey = `${workspaceSlug}/${projectId}`;
  const [filterState, setFilterState] = useState<{ scopeKey: string; filters: TPageFilters }>({
    scopeKey,
    filters: DEFAULT_PAGE_FILTERS,
  });
  const filters = useMemo(
    () => (filterState.scopeKey === scopeKey ? filterState.filters : { ...filterState.filters, searchQuery: "" }),
    [filterState, scopeKey]
  );
  const updateFilters = useCallback(
    <K extends keyof TPageFilters>(key: K, value: TPageFilters[K]) => {
      setFilterState((previous) => ({
        scopeKey,
        filters: {
          ...(previous.scopeKey === scopeKey ? previous.filters : { ...previous.filters, searchQuery: "" }),
          [key]: value,
        },
      }));
    },
    [scopeKey]
  );
  const clearAllFilters = useCallback(() => {
    setFilterState((previous) => ({
      scopeKey,
      filters: {
        ...(previous.scopeKey === scopeKey ? previous.filters : { ...previous.filters, searchQuery: "" }),
        filters: {},
      },
    }));
  }, [scopeKey]);
  const visiblePages = useMemo(
    () =>
      pageFilterList(query.isError || !query.access.isReady ? [] : (query.data ?? []), projectId, pageType, filters),
    [query.data, query.isError, query.access.isReady, projectId, pageType, filters]
  );
  const value = useMemo<TPageListContext>(
    () => ({
      projectId,
      pages: query.isError || !query.access.isReady ? [] : (query.data ?? []),
      visiblePages,
      isPending: query.isPending,
      isError: query.isError || query.access.isError,
      isDenied: query.access.isDenied,
      error: query.error,
      refetch: query.refetch,
      filters,
      updateFilters,
      clearAllFilters,
    }),
    [
      projectId,
      query.data,
      query.isPending,
      query.isError,
      query.error,
      query.refetch,
      query.access.isReady,
      query.access.isError,
      query.access.isDenied,
      visiblePages,
      filters,
      updateFilters,
      clearAllFilters,
    ]
  );
  return <PageListContext.Provider value={value}>{children}</PageListContext.Provider>;
}

export function usePageList() {
  const context = useContext(PageListContext);
  if (!context) throw new Error("usePageList must be used within PageListProvider");
  return context;
}
