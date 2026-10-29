/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { isPast } from "date-fns/isPast";
import { isToday } from "date-fns/isToday";
import sortBy from "lodash-es/sortBy.js";
import { getDate } from "@plane/utils/datetime";
import { orderCycles, shouldFilterCycle } from "@plane/utils/cycle";
import { orderModules, shouldFilterModule } from "@plane/utils/module";
import type { ICycle, IModule, TCycleFilters, TModuleFilters, TModuleDisplayFilters } from "@plane/types";

type TFavoriteModuleRecord = { entity_type?: string | null; entity_identifier?: string | null };

export function favoriteModuleIds(favorites: readonly TFavoriteModuleRecord[] | undefined) {
  return new Set(
    favorites?.flatMap((row) =>
      row.entity_type === "module" && row.entity_identifier ? [row.entity_identifier] : []
    ) ?? []
  );
}

export function cycleIsCompleted(cycle: ICycle) {
  const end = getDate(cycle.end_date);
  return cycle.status?.toLowerCase() === "completed" || Boolean(end && isPast(end) && !isToday(end));
}
export function cycleHasPoints(cycle: ICycle | undefined) {
  if (!cycle) return false;
  if (cycle.version === 2) return cycle.progress?.some((row) => row.total_estimate_points > 0) ?? false;
  return (
    cycle.version === 1 &&
    Object.values(cycle.estimate_distribution?.completion_chart ?? {}).some(
      (value) => typeof value === "number" && value > 0
    )
  );
}
export function visibleCycles(
  cycles: readonly ICycle[] | undefined,
  filters: TCycleFilters,
  search: string,
  manual = false
) {
  return cycles
    ? orderCycles(
        cycles.filter(
          (cycle) => cycle.name.toLowerCase().includes(search.toLowerCase()) && shouldFilterCycle(cycle, filters)
        ),
        manual
      )
    : undefined;
}
export function activeCycle(cycles: readonly ICycle[] | undefined) {
  return cycles?.find((cycle) => !cycle.archived_at && cycle.status?.toLowerCase() === "current");
}
export function visibleModules(
  modules: readonly IModule[] | undefined,
  display: TModuleDisplayFilters,
  filters: TModuleFilters,
  search: string,
  favoriteIds: ReadonlySet<string>
) {
  if (!modules) return undefined;
  return orderModules(
    modules.filter(
      (module) =>
        module.name.toLowerCase().includes(search.toLowerCase()) &&
        shouldFilterModule({ ...module, is_favorite: favoriteIds.has(module.id) }, display, filters)
    ),
    display.order_by
  );
}

export function visibleCompletedCycles(cycles: readonly ICycle[] | undefined, filters: TCycleFilters, search: string) {
  return cycles
    ? sortBy(
        cycles.filter(
          (cycle) =>
            cycle.status?.toLowerCase() === "completed" &&
            cycle.name.toLowerCase().includes(search.toLowerCase()) &&
            shouldFilterCycle(cycle, filters)
        ),
        (cycle) => !cycle.start_date
      )
    : undefined;
}
export function visibleArchivedCycles(cycles: readonly ICycle[] | undefined, filters: TCycleFilters, search: string) {
  return cycles
    ? sortBy(
        cycles.filter(
          (cycle) => cycle.name.toLowerCase().includes(search.toLowerCase()) && shouldFilterCycle(cycle, filters)
        ),
        (cycle) => !cycle.start_date
      )
    : undefined;
}
