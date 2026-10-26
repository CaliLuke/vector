/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createContext, useCallback, useContext, useMemo, useState } from "react";
import type { ReactNode } from "react";

type TFilterUiState = {
  visible: boolean | undefined;
  hasActiveFilters: boolean;
  hasUnsavedChanges: boolean;
  clearFilters: (() => Promise<void>) | undefined;
};

type TFilterUiContext = {
  state: TFilterUiState;
  update: (updates: Partial<TFilterUiState>) => void;
};

const EMPTY_FILTER_UI_STATE: TFilterUiState = {
  visible: undefined,
  hasActiveFilters: false,
  hasUnsavedChanges: false,
  clearFilters: undefined,
};
const FilterUiContext = createContext<TFilterUiContext | null>(null);

type TWorkItemFilterUiProviderProps = { children: ReactNode; scopeKey: string };

export function WorkItemFilterUiProvider(props: TWorkItemFilterUiProviderProps) {
  return <ScopedFilterUi key={props.scopeKey}>{props.children}</ScopedFilterUi>;
}

function ScopedFilterUi(props: { children: ReactNode }) {
  const [state, setState] = useState(EMPTY_FILTER_UI_STATE);
  const update = useCallback((updates: Partial<TFilterUiState>) => {
    setState((current) => {
      const next = { ...current, ...updates };
      return next.visible === current.visible &&
        next.hasActiveFilters === current.hasActiveFilters &&
        next.hasUnsavedChanges === current.hasUnsavedChanges &&
        next.clearFilters === current.clearFilters
        ? current
        : next;
    });
  }, []);
  const context = useMemo(() => ({ state, update }), [state, update]);
  return <FilterUiContext.Provider value={context}>{props.children}</FilterUiContext.Provider>;
}

export function useWorkItemFilterUi(): TFilterUiContext {
  const context = useContext(FilterUiContext);
  if (!context) throw new Error("useWorkItemFilterUi must be used within WorkItemFilterUiProvider");
  return context;
}
