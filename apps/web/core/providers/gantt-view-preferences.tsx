/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createContext, useCallback, useContext, useMemo, useState } from "react";
import type { ReactNode } from "react";
import type { TGanttViews, TTimelineTypeCore } from "@plane/types";
import { useTimeLineType } from "@/components/gantt-chart/contexts";

type GanttViewPreferences = Partial<Record<TTimelineTypeCore, TGanttViews>>;

type GanttViewPreferenceContextValue = {
  preferences: GanttViewPreferences;
  selectView: (timelineType: TTimelineTypeCore, view: TGanttViews) => void;
};

const GanttViewPreferenceContext = createContext<GanttViewPreferenceContextValue | undefined>(undefined);

export function GanttViewPreferenceProvider({ children }: { children: ReactNode }) {
  const [preferences, setPreferences] = useState<GanttViewPreferences>({});
  const selectView = useCallback((timelineType: TTimelineTypeCore, view: TGanttViews) => {
    setPreferences((current) => ({ ...current, [timelineType]: view }));
  }, []);
  const value = useMemo(() => ({ preferences, selectView }), [preferences, selectView]);

  return <GanttViewPreferenceContext.Provider value={value}>{children}</GanttViewPreferenceContext.Provider>;
}

export function useGanttViewPreference() {
  const timelineType = useTimeLineType();
  const context = useContext(GanttViewPreferenceContext);

  if (!timelineType) throw new Error("useGanttViewPreference must be used inside a Gantt timeline scope");
  if (!context) throw new Error("useGanttViewPreference must be used within GanttViewPreferenceProvider");

  return {
    currentView: context.preferences[timelineType] ?? "week",
    selectView: (view: TGanttViews) => context.selectView(timelineType, view),
  };
}
