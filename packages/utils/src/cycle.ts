/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { startOfToday } from "date-fns/startOfToday";
import { format } from "date-fns/format";
import isEmpty from "lodash-es/isEmpty.js";
import orderBy from "lodash-es/orderBy.js";
import sortBy from "lodash-es/sortBy.js";
import uniqBy from "lodash-es/uniqBy.js";
// plane imports
import type { ICycle, TCycleFilters, TProgressSnapshot, TCycleGroups } from "@plane/types";
// local imports
import { findTotalDaysInRange, generateDateArray } from "./datetime";
import { matchesFilterValue, matchesDateFilters } from "./filter";

export function getCycleGroup(status: unknown): TCycleGroups {
  const normalized = typeof status === "string" ? status.toLowerCase() : "";
  switch (normalized) {
    case "current":
    case "upcoming":
    case "completed":
    case "draft":
      return normalized;
    default:
      return "draft";
  }
}

/**
 * Orders cycles based on their status
 * @param {ICycle[]} cycles - Array of cycles to be ordered
 * @param {boolean} sortByManual - Whether to sort by manual order
 * @returns {ICycle[]} Ordered array of cycles
 */
export const orderCycles = (cycles: ICycle[], sortByManual: boolean): ICycle[] => {
  if (cycles.length === 0) return [];

  const acceptedStatuses = new Set(["current", "upcoming", "draft"]);
  const STATUS_ORDER: {
    [key: string]: number;
  } = {
    current: 1,
    upcoming: 2,
    draft: 3,
  };

  let filteredCycles = cycles.filter((c) => acceptedStatuses.has(c.status?.toLowerCase() ?? ""));
  if (sortByManual) filteredCycles = sortBy(filteredCycles, [(c) => c.sort_order]);
  else
    filteredCycles = sortBy(filteredCycles, [
      (c) => STATUS_ORDER[c.status?.toLowerCase() ?? ""],
      (c) => (c.status?.toLowerCase() === "upcoming" ? c.start_date : c.name.toLowerCase()),
    ]);

  return filteredCycles;
};

/**
 * Filters cycles based on provided filter criteria
 * @param {ICycle} cycle - The cycle to be filtered
 * @param {TCycleFilters} filter - Filter criteria to apply
 * @returns {boolean} Whether the cycle passes the filter
 */
export const shouldFilterCycle = (cycle: ICycle, filter: TCycleFilters): boolean => {
  return (
    matchesFilterValue(cycle.status?.toLowerCase() ?? "", filter.status) &&
    matchesDateFilters(cycle.start_date, filter.start_date) &&
    matchesDateFilters(cycle.end_date, filter.end_date)
  );
};

/**
 * Calculates the scope based on whether it's an issue or estimate points
 * @param {Partial<TProgressSnapshot>} p - Progress data
 * @param {boolean} isTypeIssue - Whether the type is an issue
 * @returns {number} Calculated scope
 */
const scope = (p: Partial<TProgressSnapshot>, isTypeIssue: boolean) =>
  (isTypeIssue ? p.total_issues : p.total_estimate_points) ?? 0;

/**
 * Calculates the ideal progress value
 * @param {string} date - Current date
 * @param {number} scope - Total scope
 * @param {ICycle} cycle - Cycle data
 * @returns {number} Ideal progress value
 */
const ideal = (date: string, totalScope: number, cycle: ICycle) =>
  Math.floor(
    ((findTotalDaysInRange(date, cycle.end_date) || 0) /
      (findTotalDaysInRange(cycle.start_date, cycle.end_date) || 0)) *
      totalScope
  );

const PROGRESS_METRICS = {
  issues: {
    total: "total_issues",
    completed: "completed_issues",
    cancelled: "cancelled_issues",
    started: "started_issues",
    unstarted: "unstarted_issues",
    backlog: "backlog_issues",
  },
  points: {
    total: "total_estimate_points",
    completed: "completed_estimate_points",
    cancelled: "cancelled_estimate_points",
    started: "started_estimate_points",
    unstarted: "unstarted_estimate_points",
    backlog: "backlog_estimate_points",
  },
} as const;

/**
 * Formats cycle data for version 1
 * @param {boolean} isTypeIssue - Whether the type is an issue
 * @param {ICycle} cycle - Cycle data
 * @param {boolean} isBurnDown - Whether it's a burn down chart
 * @param {Date|string} endDate - End date
 * @returns {TProgressChartData} Formatted progress data
 */
const formatV1Data = (isTypeIssue: boolean, cycle: ICycle, isBurnDown: boolean, endDate: Date | string) => {
  const today = format(startOfToday(), "yyyy-MM-dd");
  const data = isTypeIssue ? cycle.distribution : cycle.estimate_distribution;
  const metrics = PROGRESS_METRICS[isTypeIssue ? "issues" : "points"];
  const extendedArray = generateDateArray(endDate, endDate).map((d) => d.date);

  if (isEmpty(data)) return [];
  const progress = [...Object.keys(data.completion_chart), ...extendedArray].map((p) => {
    const pending = data.completion_chart[p] || 0;
    const total = cycle[metrics.total];
    const completed = scope(cycle, isTypeIssue) - pending;

    return {
      date: p,
      scope: p < today ? scope(cycle, isTypeIssue) : null,
      completed,
      backlog: cycle[metrics.backlog],
      started: p === today ? cycle[metrics.started] : undefined,
      unstarted: p === today ? cycle[metrics.unstarted] : undefined,
      cancelled: p === today ? cycle[metrics.cancelled] : undefined,
      pending: Math.abs(pending || 0),
      ideal: p < today ? ideal(p, total || 0, cycle) : p <= cycle.end_date! ? ideal(today, total || 0, cycle) : null,
      actual: p <= today ? (isBurnDown ? Math.abs(pending) : completed) : undefined,
    };
  });

  return progress;
};

/**
 * Formats cycle data for version 2
 * @param {boolean} isTypeIssue - Whether the type is an issue
 * @param {ICycle} cycle - Cycle data
 * @param {boolean} isBurnDown - Whether it's a burn down chart
 * @param {Date|string} endDate - End date
 * @returns {TProgressChartData} Formatted progress data
 */
const pendingProgress = (total: number, completed: number | undefined, cancelled: number | undefined) =>
  total - (completed ?? 0) - (cancelled ?? 0);

const formatV2Data = (isTypeIssue: boolean, cycle: ICycle, isBurnDown: boolean, endDate: Date | string) => {
  if (!cycle.progress) return [];
  let today: Date | string = startOfToday();

  const extendedArray = endDate > today ? generateDateArray(today, endDate) : [];
  if (isEmpty(cycle.progress)) return extendedArray;
  today = format(startOfToday(), "yyyy-MM-dd");
  const todaysData = cycle?.progress[cycle?.progress.length - 1];
  const scopeToday = scope(todaysData, isTypeIssue);
  const idealToday = ideal(todaysData.date, scopeToday, cycle);

  const rows: (Partial<TProgressSnapshot> & { date: string; progress_date?: string })[] = [
    ...orderBy(cycle.progress, "date"),
    ...extendedArray,
  ];
  let progress = rows.map((p) => {
    const completed = isTypeIssue ? p.completed_issues : p.completed_estimate_points;
    const cancelled = isTypeIssue ? p.cancelled_issues : p.cancelled_estimate_points;
    const pending = pendingProgress(scope(p, isTypeIssue), completed, cancelled);
    const dataDate = p.progress_date ? format(new Date(p.progress_date), "yyyy-MM-dd") : p.date;

    return {
      date: dataDate,
      scope: dataDate < today ? scope(p, isTypeIssue) : dataDate <= cycle.end_date! ? scopeToday : null,
      completed,
      backlog: isTypeIssue ? p.backlog_issues : p.backlog_estimate_points,
      started: isTypeIssue ? p.started_issues : p.started_estimate_points,
      unstarted: isTypeIssue ? p.unstarted_issues : p.unstarted_estimate_points,
      cancelled,
      pending: Math.abs(pending),
      ideal:
        dataDate < today
          ? ideal(dataDate, scope(p, isTypeIssue), cycle)
          : dataDate < cycle.end_date!
            ? idealToday
            : null,
      actual: dataDate <= today ? (isBurnDown ? Math.abs(pending) : completed) : undefined,
    };
  });
  progress = uniqBy(progress, "date");

  return progress;
};

export const formatActiveCycle = (args: {
  cycle: ICycle;
  isBurnDown?: boolean | undefined;
  isTypeIssue?: boolean | undefined;
}) => {
  const { cycle, isBurnDown, isTypeIssue } = args;
  const endDate: Date | string = new Date(cycle.end_date!);

  return cycle.version === 1
    ? formatV1Data(isTypeIssue!, cycle, isBurnDown!, endDate)
    : formatV2Data(isTypeIssue!, cycle, isBurnDown!, endDate);
};

/**
 * Calculates cycle progress percentage excluding cancelled issues from total count
 * Formula: completed / (total - cancelled) * 100
 * This gives accurate progress based on: pendingIssues = totalIssues - completedIssues - cancelledIssues
 * @param cycle - Cycle data object
 * @param estimateType - Whether to calculate based on "issues" or "points"
 * @param includeInProgress - Whether to include started/in-progress items in completion calculation
 * @returns Progress percentage (0-100)
 */
export const calculateCycleProgress = (
  cycle: ICycle | undefined,
  estimateType: "issues" | "points" = "issues",
  includeInProgress: boolean = false
): number => {
  if (!cycle) return 0;

  const progressSnapshot: TProgressSnapshot | undefined = cycle.progress_snapshot;
  const cycleDetails = progressSnapshot && !isEmpty(progressSnapshot) ? progressSnapshot : cycle;

  const metrics = PROGRESS_METRICS[estimateType];
  const completed =
    (cycleDetails[metrics.completed] || 0) + (includeInProgress ? cycleDetails[metrics.started] || 0 : 0);
  const cancelled = cycleDetails[metrics.cancelled] || 0;
  const total = cycleDetails[metrics.total] || 0;

  // Exclude cancelled issues from total (pendingIssues = total - completed - cancelled)
  const adjustedTotal = total - cancelled;

  // Handle edge cases
  if (adjustedTotal === 0) return 0;
  if (completed < 0 || adjustedTotal < 0) return 0;
  if (completed > adjustedTotal) return 100;

  // Calculate percentage and round
  const percentage = (completed / adjustedTotal) * 100;
  return Math.round(percentage);
};
