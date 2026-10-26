/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createContext, useCallback, useContext, useMemo, useState } from "react";
import type { ReactNode, Dispatch, SetStateAction } from "react";
import type { TInboxIssueCurrentTab, TInboxIssueFilter, TInboxIssueSorting } from "@plane/types";
import { defaultIntakePreferences } from "@/helpers/intake-preferences";
import type { IntakePreferences } from "@/helpers/intake-preferences";

type PreferenceMap = Record<string, IntakePreferences>;
const PreferencesContext = createContext<{
  saved: PreferenceMap;
  setSaved: Dispatch<SetStateAction<PreferenceMap>>;
} | null>(null);

type ScopedPreferences = IntakePreferences & {
  workspaceSlug: string;
  projectId: string;
  appliedFilterCount: number;
  setTab: (tab: TInboxIssueCurrentTab) => void;
  setFilter: <K extends keyof TInboxIssueFilter>(key: K, value: TInboxIssueFilter[K]) => void;
  setSorting: <K extends keyof TInboxIssueSorting>(key: K, value: TInboxIssueSorting[K]) => void;
};
const IntakePreferencesContext = createContext<ScopedPreferences | null>(null);

export function IntakePreferencesProvider({ children }: { children: ReactNode }) {
  const [saved, setSaved] = useState<PreferenceMap>({});
  const value = useMemo(() => ({ saved, setSaved }), [saved]);
  return <PreferencesContext.Provider value={value}>{children}</PreferencesContext.Provider>;
}

export function IntakePreferencesScope({
  workspaceSlug,
  projectId,
  navigationTab,
  children,
}: {
  workspaceSlug: string;
  projectId: string;
  navigationTab?: TInboxIssueCurrentTab;
  children: ReactNode;
}) {
  const context = useContext(PreferencesContext);
  if (!context) throw new Error("Intake preferences require the session provider.");
  const key = JSON.stringify([workspaceSlug, projectId]);
  const { saved, setSaved } = context;
  const preferences = useMemo(() => {
    const current = saved[key];
    return current && (!navigationTab || current.tab === navigationTab)
      ? current
      : defaultIntakePreferences(navigationTab);
  }, [key, navigationTab, saved]);
  const update = useCallback(
    (change: (current: IntakePreferences) => IntakePreferences) => {
      setSaved((previous) => {
        const current = previous[key];
        const base =
          current && (!navigationTab || current.tab === navigationTab)
            ? current
            : defaultIntakePreferences(navigationTab);
        return { ...previous, [key]: change(base) };
      });
    },
    [key, navigationTab, setSaved]
  );
  const setTab = useCallback((tab: TInboxIssueCurrentTab) => update(() => defaultIntakePreferences(tab)), [update]);
  const setFilter = useCallback(
    <K extends keyof TInboxIssueFilter>(filterKey: K, value: TInboxIssueFilter[K]) => {
      update((current) => ({ ...current, filters: { ...current.filters, [filterKey]: value } }));
    },
    [update]
  );
  const setSorting = useCallback(
    <K extends keyof TInboxIssueSorting>(field: K, value: TInboxIssueSorting[K]) => {
      update((current) => ({ ...current, sorting: { ...current.sorting, [field]: value } }));
    },
    [update]
  );
  const value = useMemo(
    () => ({
      ...preferences,
      workspaceSlug,
      projectId,
      setTab,
      setFilter,
      setSorting,
      appliedFilterCount: Object.values(preferences.filters).reduce(
        (total, values) => total + (values?.length ?? 0),
        0
      ),
    }),
    [preferences, workspaceSlug, projectId, setTab, setFilter, setSorting]
  );
  return <IntakePreferencesContext.Provider value={value}>{children}</IntakePreferencesContext.Provider>;
}

export function useIntakePreferences() {
  const context = useContext(IntakePreferencesContext);
  if (!context) throw new Error("Intake filters require a project scope.");
  return context;
}
