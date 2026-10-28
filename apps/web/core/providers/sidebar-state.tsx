/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createContext, useContext, useEffect, useState } from "react";
import type { ReactNode } from "react";
import { createStore, useSelector } from "@tanstack/react-store";

type TSidebarValues = {
  isAnySidebarDropdownOpen: boolean | undefined;
  sidebarCollapsed: boolean | undefined;
  sidebarPeek: boolean | undefined;
  isExtendedSidebarOpened: boolean | undefined;
  isExtendedProjectSidebarOpened: boolean | undefined;
  profileSidebarCollapsed: boolean | undefined;
  issueDetailSidebarCollapsed: boolean | undefined;
};

type TSidebarKey = keyof TSidebarValues;
type TPersistedSidebarKey =
  | "sidebarCollapsed"
  | "isExtendedSidebarOpened"
  | "isExtendedProjectSidebarOpened"
  | "profileSidebarCollapsed"
  | "issueDetailSidebarCollapsed";

type TSidebarActions = {
  toggleAnySidebarDropdown: (open?: boolean) => void;
  toggleSidebar: (collapsed?: boolean) => void;
  toggleSidebarPeek: (peek?: boolean) => void;
  toggleExtendedSidebar: (opened?: boolean) => void;
  toggleExtendedProjectSidebar: (opened?: boolean) => void;
  toggleProfileSidebar: (collapsed?: boolean) => void;
  toggleIssueDetailSidebar: (collapsed?: boolean) => void;
};

type TSidebarAction =
  | { type: "hydrate"; values: Partial<TSidebarValues> }
  | { type: "toggle"; key: TSidebarKey; value?: boolean };

const initialValues: TSidebarValues = {
  isAnySidebarDropdownOpen: undefined,
  sidebarCollapsed: undefined,
  sidebarPeek: undefined,
  isExtendedSidebarOpened: undefined,
  isExtendedProjectSidebarOpened: undefined,
  profileSidebarCollapsed: undefined,
  issueDetailSidebarCollapsed: undefined,
};

const persistedSidebarPreferences = [
  { stateKey: "sidebarCollapsed", storageKey: "app_sidebar_collapsed" },
  { stateKey: "isExtendedSidebarOpened", storageKey: "extended_sidebar_collapsed" },
  { stateKey: "isExtendedProjectSidebarOpened", storageKey: "extended_project_sidebar_collapsed" },
  { stateKey: "profileSidebarCollapsed", storageKey: "profile_sidebar_collapsed" },
  { stateKey: "issueDetailSidebarCollapsed", storageKey: "issue_detail_sidebar_collapsed" },
] satisfies { stateKey: TPersistedSidebarKey; storageKey: string }[];

function readPersistedValues(): Partial<TSidebarValues> {
  if (typeof window === "undefined") return {};

  const values: Partial<TSidebarValues> = {};
  for (const { stateKey, storageKey } of persistedSidebarPreferences) {
    const value = window.localStorage.getItem(storageKey);
    if (value !== null) values[stateKey] = value === "true";
  }
  return values;
}

function sidebarReducer(state: TSidebarValues, action: TSidebarAction): TSidebarValues {
  if (action.type === "hydrate") {
    const hydrated = { ...state };
    for (const { stateKey } of persistedSidebarPreferences) {
      // Mounted consumers can choose viewport defaults before this effect runs.
      if (hydrated[stateKey] === undefined) hydrated[stateKey] = action.values[stateKey];
    }
    return hydrated;
  }
  return { ...state, [action.key]: action.value ?? !state[action.key] };
}

const SidebarContext = createContext<ReturnType<typeof createSidebarState> | undefined>(undefined);

function createSidebarState() {
  const store = createStore(initialValues);
  const toggle = (key: TSidebarKey, value?: boolean) =>
    store.setState((state) => sidebarReducer(state, { type: "toggle", key, value }));
  const actions: TSidebarActions = {
    toggleAnySidebarDropdown: (open) => toggle("isAnySidebarDropdownOpen", open),
    toggleSidebar: (collapsed) => toggle("sidebarCollapsed", collapsed),
    toggleSidebarPeek: (peek) => toggle("sidebarPeek", peek),
    toggleExtendedSidebar: (opened) => toggle("isExtendedSidebarOpened", opened),
    toggleExtendedProjectSidebar: (opened) => toggle("isExtendedProjectSidebarOpened", opened),
    toggleProfileSidebar: (collapsed) => toggle("profileSidebarCollapsed", collapsed),
    toggleIssueDetailSidebar: (collapsed) => toggle("issueDetailSidebarCollapsed", collapsed),
  };
  return { store, actions };
}

export function SidebarStateProvider({ children }: { children: ReactNode }) {
  const [sidebar] = useState(createSidebarState);
  useEffect(() => {
    sidebar.store.setState((state) => sidebarReducer(state, { type: "hydrate", values: readPersistedValues() }));
    const persist = () => {
      for (const { stateKey, storageKey } of persistedSidebarPreferences) {
        const value = sidebar.store.state[stateKey];
        if (value !== undefined) window.localStorage.setItem(storageKey, String(value));
      }
    };
    persist();
    const subscription = sidebar.store.subscribe(persist);
    return () => subscription.unsubscribe();
  }, [sidebar]);
  return <SidebarContext.Provider value={sidebar}>{children}</SidebarContext.Provider>;
}

function useSidebarState() {
  const sidebar = useContext(SidebarContext);
  if (!sidebar) throw new Error("Sidebar state hooks must be used within SidebarStateProvider");
  return sidebar;
}
function useSidebarValue(key: TSidebarKey) {
  return useSelector(useSidebarState().store, (state) => state[key]);
}
export function useSidebarActions(): TSidebarActions {
  return useSidebarState().actions;
}

export function useSidebarDropdownOpen(): boolean | undefined {
  return useSidebarValue("isAnySidebarDropdownOpen");
}

export function useSidebarCollapsed(): boolean | undefined {
  return useSidebarValue("sidebarCollapsed");
}

export function useSidebarPeek(): boolean | undefined {
  return useSidebarValue("sidebarPeek");
}

export function useExtendedSidebarOpen(): boolean | undefined {
  return useSidebarValue("isExtendedSidebarOpened");
}

export function useExtendedProjectSidebarOpen(): boolean | undefined {
  return useSidebarValue("isExtendedProjectSidebarOpened");
}

export function useProfileSidebarCollapsed(): boolean | undefined {
  return useSidebarValue("profileSidebarCollapsed");
}

export function useIssueDetailSidebarCollapsed(): boolean | undefined {
  return useSidebarValue("issueDetailSidebarCollapsed");
}
