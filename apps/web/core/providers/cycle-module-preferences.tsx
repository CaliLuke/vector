/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { createContext, useCallback, useContext, useMemo, useState, useEffect } from "react";
import type { ReactNode } from "react";
import type {
  TCycleDisplayFilters,
  TCycleFilters,
  TModuleDisplayFilters,
  TModuleFilters,
  TCycleEstimateType,
} from "@plane/types";
import { storage } from "@/lib/local-storage";
import { useCurrentUserId } from "@/hooks/use-current-user";

type CyclePreferences = {
  display: TCycleDisplayFilters;
  filters: TCycleFilters;
  archivedFilters: TCycleFilters;
  search: string;
  archivedSearch: string;
};
type ModulePreferences = {
  display: TModuleDisplayFilters;
  filters: TModuleFilters;
  archivedFilters: TModuleFilters;
  search: string;
  archivedSearch: string;
};
const defaultCycle: CyclePreferences = {
  display: { active_tab: "active", layout: "list" },
  filters: {},
  archivedFilters: {},
  search: "",
  archivedSearch: "",
};
const defaultModule: ModulePreferences = {
  display: { layout: "list", order_by: "name" },
  filters: {},
  archivedFilters: {},
  search: "",
  archivedSearch: "",
};
type ChartPreferences = {
  measure: TCycleEstimateType;
  sidebarCollapsed: boolean;
  statisticsTab: "Priority-Issues" | "Assignees" | "Labels";
};
const defaultChart: ChartPreferences = { measure: "issues", sidebarCollapsed: false, statisticsTab: "Assignees" };
type PreferenceState = {
  charts: Record<string, ChartPreferences>;
  setChart: (key: string, patch: Partial<ChartPreferences>, initial: ChartPreferences) => void;
  cycles: Record<string, CyclePreferences>;
  modules: Record<string, ModulePreferences>;
  setCycle: (key: string, patch: Partial<CyclePreferences>) => void;
  setModule: (key: string, patch: Partial<ModulePreferences>, initial: ModulePreferences) => void;
};
const Context = createContext<PreferenceState | undefined>(undefined);
function moduleStorageKey(key: string) {
  return `vector:module-catalogue-preferences:${key}`;
}
function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}
function isModuleLayout(value: unknown): value is NonNullable<TModuleDisplayFilters["layout"]> {
  return value === "list" || value === "board" || value === "gantt";
}
function isModuleOrder(value: unknown): value is NonNullable<TModuleDisplayFilters["order_by"]> {
  return (
    value === "name" ||
    value === "-name" ||
    value === "progress" ||
    value === "-progress" ||
    value === "issues_length" ||
    value === "-issues_length" ||
    value === "target_date" ||
    value === "-target_date" ||
    value === "created_at" ||
    value === "-created_at" ||
    value === "sort_order"
  );
}
function readModuleDisplay(value: unknown): TModuleDisplayFilters {
  const display: TModuleDisplayFilters = { ...defaultModule.display };
  if (!isRecord(value)) return display;
  if (isModuleLayout(value.layout)) display.layout = value.layout;
  if (typeof value.favorites === "boolean") display.favorites = value.favorites;
  if (isModuleOrder(value.order_by)) display.order_by = value.order_by;
  return display;
}
function moduleFilters(value: unknown): TModuleFilters {
  if (!isRecord(value)) return {};
  const result: TModuleFilters = {};
  for (const key of ["lead", "members", "start_date", "status", "target_date"] as const) {
    if (!Object.hasOwn(value, key)) continue;
    const selected: unknown = Object.getOwnPropertyDescriptor(value, key)?.value;
    if (selected === null || (Array.isArray(selected) && selected.every((item) => typeof item === "string")))
      result[key] = selected;
  }
  return result;
}
/** Unowned legacy project-only records cannot identify an account and are deliberately ignored. */
function readModule(key: string): ModulePreferences {
  const raw = storage.get(moduleStorageKey(key));
  if (!raw) return defaultModule;
  try {
    const value: unknown = JSON.parse(raw);
    if (!isRecord(value)) return defaultModule;
    return {
      ...defaultModule,
      display: readModuleDisplay(value.display),
      filters: moduleFilters(value.filters),
      archivedFilters: moduleFilters(value.archivedFilters),
    };
  } catch (error) {
    console.warn("Module preferences could not be read.", error);
    return defaultModule;
  }
}
export function CycleModulePreferencesProvider({ children }: { children: ReactNode }) {
  const [charts, setCharts] = useState<Record<string, ChartPreferences>>({});
  const setChart = useCallback(
    (key: string, patch: Partial<ChartPreferences>, initial: ChartPreferences) =>
      setCharts((current) => ({ ...current, [key]: { ...(current[key] ?? initial), ...patch } })),
    []
  );
  const [cycles, setCycles] = useState<Record<string, CyclePreferences>>({}),
    [modules, setModules] = useState<Record<string, ModulePreferences>>({});
  const setCycle = useCallback(
    (key: string, patch: Partial<CyclePreferences>) =>
      setCycles((current) => ({ ...current, [key]: { ...defaultCycle, ...current[key], ...patch } })),
    []
  );
  const setModule = useCallback((key: string, patch: Partial<ModulePreferences>, initial: ModulePreferences) => {
    setModules((current) => {
      const next = { ...(current[key] ?? initial), ...patch };
      return { ...current, [key]: next };
    });
  }, []);
  useEffect(() => {
    for (const [key, value] of Object.entries(charts))
      storage.set(`vector:parent-chart-preferences:${key}`, {
        sidebarCollapsed: value.sidebarCollapsed,
        statisticsTab: value.statisticsTab,
      });
  }, [charts]);
  useEffect(() => {
    for (const [key, next] of Object.entries(modules))
      storage.set(moduleStorageKey(key), {
        display: next.display,
        filters: next.filters,
        archivedFilters: next.archivedFilters,
      });
  }, [modules]);
  const value = useMemo(
    () => ({ cycles, modules, setCycle, setModule, charts, setChart }),
    [cycles, modules, setCycle, setModule, charts, setChart]
  );
  return <Context.Provider value={value}>{children}</Context.Provider>;
}
function useScope(workspaceSlug: string | undefined, projectId: string | undefined) {
  const accountId = useCurrentUserId();
  const context = useContext(Context);
  if (!context) throw new Error("Cycle/module preferences require their provider.");
  return {
    context,
    key: JSON.stringify([accountId, workspaceSlug, projectId]),
    ready: Boolean(accountId && workspaceSlug && projectId),
  };
}
export function useCycleCataloguePreferences(workspaceSlug: string | undefined, projectId: string | undefined) {
  const { context, key, ready } = useScope(workspaceSlug, projectId),
    value = context.cycles[key] ?? defaultCycle;
  return {
    ...value,
    update: (patch: Partial<CyclePreferences>) => {
      if (ready) context.setCycle(key, patch);
    },
  };
}
export function useModuleCataloguePreferences(workspaceSlug: string | undefined, projectId: string | undefined) {
  const { context, key, ready } = useScope(workspaceSlug, projectId),
    initial = useMemo(() => (ready ? readModule(key) : defaultModule), [key, ready]);
  return {
    ...(context.modules[key] ?? initial),
    update: (patch: Partial<ModulePreferences>) => {
      if (ready) context.setModule(key, patch, initial);
    },
  };
}

function readChart(key: string): ChartPreferences {
  const raw = storage.get(`vector:parent-chart-preferences:${key}`);
  if (!raw) return defaultChart;
  try {
    const value: unknown = JSON.parse(raw);
    if (!value || typeof value !== "object") return defaultChart;
    return {
      ...defaultChart,
      ...("sidebarCollapsed" in value && typeof value.sidebarCollapsed === "boolean"
        ? { sidebarCollapsed: value.sidebarCollapsed }
        : {}),
      ...("statisticsTab" in value &&
      (value.statisticsTab === "Priority-Issues" ||
        value.statisticsTab === "Assignees" ||
        value.statisticsTab === "Labels")
        ? { statisticsTab: value.statisticsTab }
        : {}),
    };
  } catch (error) {
    console.warn("Chart preferences could not be read.", error);
    return defaultChart;
  }
}
export function useParentChartPreferences(
  domain: "cycle" | "module",
  workspaceSlug: string | undefined,
  projectId: string | undefined,
  parentId: string | undefined
) {
  const { context, key: projectKey, ready } = useScope(workspaceSlug, projectId),
    key = JSON.stringify([projectKey, domain, parentId]);
  const initial = useMemo(() => (ready ? readChart(key) : defaultChart), [key, ready]);
  return {
    ...(context.charts[key] ?? initial),
    update: (patch: Partial<ChartPreferences>) => {
      if (ready && parentId) context.setChart(key, patch, initial);
    },
  };
}
