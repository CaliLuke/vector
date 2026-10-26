/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createContext, useContext, useState, useMemo } from "react";
import type { ReactNode } from "react";

const HomeContext = createContext<
  { showWidgetSettings: boolean; setShowWidgetSettings: (open: boolean) => void } | undefined
>(undefined);

export function HomeProvider({ children }: { children: ReactNode }) {
  const [showWidgetSettings, setShowWidgetSettings] = useState(false);
  const value = useMemo(() => ({ showWidgetSettings, setShowWidgetSettings }), [showWidgetSettings]);
  return <HomeContext.Provider value={value}>{children}</HomeContext.Provider>;
}

export function useHomeSettings() {
  const context = useContext(HomeContext);
  if (!context) throw new Error("useHomeSettings must be used within HomeProvider");
  return context;
}
