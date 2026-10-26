/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createContext, useContext, useMemo, useSyncExternalStore } from "react";
import type { PropsWithChildren } from "react";

const SidebarContext = createContext<
  | {
      isSidebarCollapsed: boolean;
      toggleSidebar: (collapsed: boolean) => void;
    }
  | undefined
>(undefined);

const storageKey = "god_mode_sidebar_collapsed";
const changeEvent = `local-storage:${storageKey}`;
const readCollapsed = () => window.localStorage.getItem(storageKey) === "true";
const serverCollapsed = () => false;

function subscribe(onChange: () => void) {
  const onStorage = (event: StorageEvent) => {
    if (event.key === storageKey || event.key === null) onChange();
  };
  window.addEventListener("storage", onStorage);
  window.addEventListener(changeEvent, onChange);
  return () => {
    window.removeEventListener("storage", onStorage);
    window.removeEventListener(changeEvent, onChange);
  };
}
function toggleSidebar(collapsed: boolean) {
  window.localStorage.setItem(storageKey, String(collapsed));
  window.dispatchEvent(new Event(changeEvent));
}

export function SidebarProvider({ children }: PropsWithChildren) {
  const isSidebarCollapsed = useSyncExternalStore(subscribe, readCollapsed, serverCollapsed);
  const value = useMemo(() => ({ isSidebarCollapsed, toggleSidebar }), [isSidebarCollapsed]);
  return <SidebarContext.Provider value={value}>{children}</SidebarContext.Provider>;
}
export function useSidebar() {
  const sidebar = useContext(SidebarContext);
  if (!sidebar) throw new Error("useSidebar must be used within SidebarProvider");
  return sidebar;
}
