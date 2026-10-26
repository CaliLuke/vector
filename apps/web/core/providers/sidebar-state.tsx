/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createContext, useCallback, useContext, useEffect, useMemo, useReducer } from "react";
import type { Context, ReactNode } from "react";

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

const SidebarProviderContext = createContext(false);
const SidebarActionsContext = createContext<TSidebarActions | undefined>(undefined);
const SidebarDropdownContext = createContext<boolean | undefined>(undefined);
const SidebarCollapsedContext = createContext<boolean | undefined>(undefined);
const SidebarPeekContext = createContext<boolean | undefined>(undefined);
const ExtendedSidebarContext = createContext<boolean | undefined>(undefined);
const ExtendedProjectSidebarContext = createContext<boolean | undefined>(undefined);
const ProfileSidebarCollapsedContext = createContext<boolean | undefined>(undefined);
const IssueDetailSidebarCollapsedContext = createContext<boolean | undefined>(undefined);

export function SidebarStateProvider({ children }: { children: ReactNode }) {
  const [values, dispatch] = useReducer(sidebarReducer, initialValues);

  useEffect(() => {
    dispatch({ type: "hydrate", values: readPersistedValues() });
  }, []);

  useEffect(() => {
    if (typeof window === "undefined") return;
    for (const { stateKey, storageKey } of persistedSidebarPreferences) {
      const value = values[stateKey];
      if (value !== undefined) window.localStorage.setItem(storageKey, String(value));
    }
  }, [values]);

  const toggle = useCallback((key: TSidebarKey, value?: boolean) => {
    dispatch({ type: "toggle", key, value });
  }, []);
  const actions = useMemo<TSidebarActions>(
    () => ({
      toggleAnySidebarDropdown: (open) => toggle("isAnySidebarDropdownOpen", open),
      toggleSidebar: (collapsed) => toggle("sidebarCollapsed", collapsed),
      toggleSidebarPeek: (peek) => toggle("sidebarPeek", peek),
      toggleExtendedSidebar: (opened) => toggle("isExtendedSidebarOpened", opened),
      toggleExtendedProjectSidebar: (opened) => toggle("isExtendedProjectSidebarOpened", opened),
      toggleProfileSidebar: (collapsed) => toggle("profileSidebarCollapsed", collapsed),
      toggleIssueDetailSidebar: (collapsed) => toggle("issueDetailSidebarCollapsed", collapsed),
    }),
    [toggle]
  );

  return (
    <SidebarProviderContext.Provider value={true}>
      <SidebarActionsContext.Provider value={actions}>
        <SidebarDropdownContext.Provider value={values.isAnySidebarDropdownOpen}>
          <SidebarCollapsedContext.Provider value={values.sidebarCollapsed}>
            <SidebarPeekContext.Provider value={values.sidebarPeek}>
              <ExtendedSidebarContext.Provider value={values.isExtendedSidebarOpened}>
                <ExtendedProjectSidebarContext.Provider value={values.isExtendedProjectSidebarOpened}>
                  <ProfileSidebarCollapsedContext.Provider value={values.profileSidebarCollapsed}>
                    <IssueDetailSidebarCollapsedContext.Provider value={values.issueDetailSidebarCollapsed}>
                      {children}
                    </IssueDetailSidebarCollapsedContext.Provider>
                  </ProfileSidebarCollapsedContext.Provider>
                </ExtendedProjectSidebarContext.Provider>
              </ExtendedSidebarContext.Provider>
            </SidebarPeekContext.Provider>
          </SidebarCollapsedContext.Provider>
        </SidebarDropdownContext.Provider>
      </SidebarActionsContext.Provider>
    </SidebarProviderContext.Provider>
  );
}

function useSidebarContextValue<T>(context: Context<T | undefined>): T | undefined {
  if (!useContext(SidebarProviderContext))
    throw new Error("Sidebar state hooks must be used within SidebarStateProvider");
  return useContext(context);
}

export function useSidebarActions(): TSidebarActions {
  const actions = useSidebarContextValue(SidebarActionsContext);
  if (!actions) throw new Error("Sidebar state hooks must be used within SidebarStateProvider");
  return actions;
}

export function useSidebarDropdownOpen(): boolean | undefined {
  return useSidebarContextValue(SidebarDropdownContext);
}

export function useSidebarCollapsed(): boolean | undefined {
  return useSidebarContextValue(SidebarCollapsedContext);
}

export function useSidebarPeek(): boolean | undefined {
  return useSidebarContextValue(SidebarPeekContext);
}

export function useExtendedSidebarOpen(): boolean | undefined {
  return useSidebarContextValue(ExtendedSidebarContext);
}

export function useExtendedProjectSidebarOpen(): boolean | undefined {
  return useSidebarContextValue(ExtendedProjectSidebarContext);
}

export function useProfileSidebarCollapsed(): boolean | undefined {
  return useSidebarContextValue(ProfileSidebarCollapsedContext);
}

export function useIssueDetailSidebarCollapsed(): boolean | undefined {
  return useSidebarContextValue(IssueDetailSidebarCollapsedContext);
}
