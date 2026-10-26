/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createContext, useCallback, useContext, useMemo, useState } from "react";
import type { ReactNode } from "react";
import type {
  IIssueDisplayFilterOptions,
  IIssueDisplayProperties,
  IIssueFilterOptions,
  TIssueServiceType,
} from "@plane/types";

import { DEFAULT_SUB_WORK_ITEM_DISPLAY_PROPERTIES } from "@/helpers/sub-work-item-preferences";

type SubWorkItemPreferences = {
  filters: IIssueFilterOptions;
  displayFilters: Partial<IIssueDisplayFilterOptions>;
  displayProperties: Partial<IIssueDisplayProperties>;
};

type PreferenceScope = {
  workspaceSlug: string | undefined;
  projectId: string;
  parentId: string;
  serviceType: TIssueServiceType;
};

const defaultPreferences: SubWorkItemPreferences = {
  filters: {},
  displayFilters: {},
  displayProperties: DEFAULT_SUB_WORK_ITEM_DISPLAY_PROPERTIES,
};

type PreferenceContext = {
  preferences: Record<string, SubWorkItemPreferences>;
  update: (key: string, patch: Partial<SubWorkItemPreferences>) => void;
  reset: (key: string) => void;
};

const SubWorkItemPreferenceContext = createContext<PreferenceContext | null>(null);

export function SubWorkItemPreferencesProvider({ children }: { children: ReactNode }) {
  const [preferences, setPreferences] = useState<Record<string, SubWorkItemPreferences>>({});
  const update = useCallback((key: string, patch: Partial<SubWorkItemPreferences>) => {
    setPreferences((current) => {
      const saved = current[key] ?? defaultPreferences;
      return {
        ...current,
        [key]: {
          filters: { ...saved.filters, ...patch.filters },
          displayFilters: { ...saved.displayFilters, ...patch.displayFilters },
          displayProperties: { ...saved.displayProperties, ...patch.displayProperties },
        },
      };
    });
  }, []);
  const reset = useCallback((key: string) => {
    setPreferences((current) => {
      const next = { ...current };
      delete next[key];
      return next;
    });
  }, []);
  const value = useMemo(() => ({ preferences, update, reset }), [preferences, update, reset]);
  return <SubWorkItemPreferenceContext.Provider value={value}>{children}</SubWorkItemPreferenceContext.Provider>;
}

export function useSubWorkItemPreferences(scope: PreferenceScope) {
  const context = useContext(SubWorkItemPreferenceContext);
  if (!context) throw new Error("Sub-item preferences require SubWorkItemPreferencesProvider.");
  const key = JSON.stringify([scope.workspaceSlug, scope.projectId, scope.parentId, scope.serviceType]);
  const { preferences, update, reset } = context;
  const updatePreferences = useCallback((patch: Partial<SubWorkItemPreferences>) => update(key, patch), [key, update]);
  const resetPreferences = useCallback(() => reset(key), [key, reset]);
  return { preferences: preferences[key] ?? defaultPreferences, updatePreferences, resetPreferences };
}
