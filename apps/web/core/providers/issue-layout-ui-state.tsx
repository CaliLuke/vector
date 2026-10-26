/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ReactNode } from "react";
import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { DRAG_ALLOWED_GROUPS } from "@plane/constants";
import type { TIssueGroupByOptions } from "@plane/types";

export type TCalendarDates = {
  activeMonthDate: Date;
  activeWeekDate: Date;
};

type TCalendarState = TCalendarDates & {
  updateCalendarDates: (filters: Partial<TCalendarDates>) => void;
};

const CalendarStateContext = createContext<TCalendarState | null>(null);

export function CalendarStateProvider({
  children,
  initialDates,
}: {
  children: ReactNode;
  initialDates?: TCalendarDates;
}) {
  const [dates, setDates] = useState<TCalendarDates>(() => {
    if (initialDates) return initialDates;
    const today = new Date();
    return { activeMonthDate: today, activeWeekDate: today };
  });
  const updateCalendarDates = useCallback((filters: Partial<TCalendarDates>) => {
    setDates((current) => ({ ...current, ...filters }));
  }, []);
  const value = useMemo(() => ({ ...dates, updateCalendarDates }), [dates, updateCalendarDates]);

  return <CalendarStateContext.Provider value={value}>{children}</CalendarStateContext.Provider>;
}

export function useCalendarState() {
  const state = useContext(CalendarStateContext);
  if (!state) throw new Error("useCalendarState must be used within CalendarStateProvider");
  return state;
}

type TKanbanDragState = {
  isDragging: boolean;
  beginDrag: (owner: symbol) => void;
  endDrag: (owner: symbol) => void;
};

const KanbanDragContext = createContext<TKanbanDragState | null>(null);

export function KanbanDragStateProvider({ children }: { children: ReactNode }) {
  const [dragOwner, setDragOwner] = useState<symbol | null>(null);
  const beginDrag = useCallback((owner: symbol) => setDragOwner(owner), []);
  const endDrag = useCallback((owner: symbol) => setDragOwner((current) => (current === owner ? null : current)), []);
  const value = useMemo(
    () => ({ isDragging: dragOwner !== null, beginDrag, endDrag }),
    [beginDrag, dragOwner, endDrag]
  );
  return <KanbanDragContext.Provider value={value}>{children}</KanbanDragContext.Provider>;
}

export function useKanbanDragState() {
  const state = useContext(KanbanDragContext);
  if (!state) throw new Error("useKanbanDragState must be used within KanbanDragContext");
  return state;
}

export function useKanbanDragOwner() {
  const { beginDrag, endDrag } = useKanbanDragState();
  const owner = useRef(Symbol("kanban-issue-drag"));
  const begin = useCallback(() => beginDrag(owner.current), [beginDrag]);
  const end = useCallback(() => endDrag(owner.current), [endDrag]);

  useEffect(() => end, [end]);

  return { beginDrag: begin, endDrag: end };
}

export function canDragDropForGrouping(
  groupBy: TIssueGroupByOptions | undefined,
  subGroupBy: TIssueGroupByOptions | undefined
) {
  return (
    !!groupBy && DRAG_ALLOWED_GROUPS.includes(groupBy) && (!subGroupBy || DRAG_ALLOWED_GROUPS.includes(subGroupBy))
  );
}
