/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useLocation } from "@tanstack/react-router";
import { useCallback, useMemo, useState } from "react";
import isEmpty from "lodash-es/isEmpty.js";

import { Disclosure, Transition } from "@headlessui/react";
// plane imports
import { useTranslation } from "@plane/i18n";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { ChevronUpIcon, ChevronDownIcon } from "@plane/propel/icons";
import type { ICycle, TCyclePlotType } from "@plane/types";
import { getDate } from "@plane/utils/datetime";
import type { TWorkItemFilterCondition } from "@plane/shared-state";
import { updateWorkItemFilterFromSidebar } from "@plane/shared-state";
// hooks
import { progressCount, progressDistribution } from "@/components/estimates/progress-distribution";
import { useEstimateStatistics } from "@/hooks/use-estimate-statistics";
import { EstimateStatisticsBoundary } from "@/components/estimates/statistics-boundary";
import { useCycleDetails } from "@/hooks/use-cycles";
import { useParentChartPreferences } from "@/providers/cycle-module-preferences";
// plane web components
import { useParentIssuePreferences } from "@/providers/parent-issue-preferences";
import { readProgressFilterSelection } from "@/components/core/sidebar/progress-filter-selection";
// local imports
import { CycleProgressStats } from "./progress-stats";
import { SidebarChart } from "./sidebar-chart";

type TCycleAnalyticsProgress = {
  workspaceSlug: string;
  projectId: string;
  cycleId: string;
  isArchived?: boolean;
};
type Options = {
  value: string;
  label: string;
};

export const cycleEstimateOptions: Options[] = [
  { value: "issues", label: "Work items" },
  { value: "points", label: "Estimates" },
];

function isCycleRangeValid(start: Date | undefined, end: Date | undefined, today: Date) {
  return Boolean(start && end && start <= today && end >= start);
}

export function validateCycleSnapshot(cycle: ICycle | null | undefined): ICycle | null {
  if (!cycle) return null;
  return cycle.progress_snapshot && !isEmpty(cycle.progress_snapshot)
    ? { ...cycle, ...cycle.progress_snapshot }
    : cycle;
}

function getGroupedCycleIssues(cycle: ICycle | null | undefined, estimateType: string) {
  const estimateMode = estimateType === "points";
  return {
    backlog: progressCount(cycle?.backlog_estimate_points, cycle?.backlog_issues, estimateMode),
    unstarted: progressCount(cycle?.unstarted_estimate_points, cycle?.unstarted_issues, estimateMode),
    started: progressCount(cycle?.started_estimate_points, cycle?.started_issues, estimateMode),
    completed: progressCount(cycle?.completed_estimate_points, cycle?.completed_issues, estimateMode),
    cancelled: progressCount(cycle?.cancelled_estimate_points, cycle?.cancelled_issues, estimateMode),
  };
}

export function CycleAnalyticsProgress(props: TCycleAnalyticsProgress) {
  // props
  const { workspaceSlug, projectId, cycleId, isArchived = false } = props;
  const estimateStats = useEstimateStatistics("cycle", workspaceSlug, projectId, cycleId, isArchived);
  const { isEnabled: estimatesEnabled } = estimateStats;
  // router
  const searchParams = new URLSearchParams(useLocation({ select: (location) => location.searchStr }));
  const peekCycle = searchParams.get("peekCycle") || undefined;
  // plane hooks
  const { t } = useTranslation();
  // store hooks
  const { cycle } = useCycleDetails(workspaceSlug, projectId, cycleId, isArchived);
  const chart = useParentChartPreferences("cycle", workspaceSlug, projectId, cycleId);
  const preferences = useParentIssuePreferences();
  const { setRich, filters } = preferences;
  // derived values
  const richFilters = filters?.richFilters;
  const selectedFilters = useMemo(() => readProgressFilterSelection(richFilters), [richFilters]);
  const handleFiltersUpdate = useCallback(
    (condition: TWorkItemFilterCondition) => {
      const result = updateWorkItemFilterFromSidebar(richFilters, condition);
      if (!result.ok) {
        setToast({ type: TOAST_TYPE.ERROR, title: t("common.error.label"), message: result.error });
        return;
      }
      void setRich(result.value).catch(() => {
        setToast({
          type: TOAST_TYPE.ERROR,
          title: t("common.error.label"),
          message: t("common.something_went_wrong"),
        });
      });
    },
    [setRich, richFilters, t]
  );
  const cycleDetails = validateCycleSnapshot(cycle);
  const plotType: TCyclePlotType = "burndown";
  const estimateType = estimatesEnabled ? chart.measure : "issues";
  const totalIssues = cycleDetails?.total_issues || 0;
  const totalEstimatePoints = cycleDetails?.total_estimate_points || 0;
  const chartDistributionData = progressDistribution(cycleDetails, estimateType);
  const groupedIssues = getGroupedCycleIssues(cycleDetails, estimateType);
  const cycleStartDate = getDate(cycleDetails?.start_date);
  const cycleEndDate = getDate(cycleDetails?.end_date);
  const [today] = useState(() => new Date());
  const isCycleDateValid = isCycleRangeValid(cycleStartDate, cycleEndDate, today);

  if (!cycleDetails) return <></>;
  return (
    <EstimateStatisticsBoundary state={estimateStats.statistics}>
      {
        <div className="space-y-4 border-t border-subtle py-5">
          <Disclosure defaultOpen>
            {({ open }) => (
              <div className="flex flex-col">
                {/* progress bar header */}
                {isCycleDateValid ? (
                  <div className="relative flex w-full items-center justify-between gap-2">
                    <Disclosure.Button className="relative flex w-full items-center gap-2">
                      <div className="text-13 font-medium text-secondary">
                        {t("project_cycles.active_cycle.progress")}
                      </div>
                    </Disclosure.Button>
                    <Disclosure.Button className="ml-auto">
                      {open ? (
                        <ChevronUpIcon className="h-3.5 w-3.5" aria-hidden="true" />
                      ) : (
                        <ChevronDownIcon className="h-3.5 w-3.5" aria-hidden="true" />
                      )}
                    </Disclosure.Button>
                  </div>
                ) : (
                  <div className="relative flex w-full items-center justify-between gap-2">
                    <div className="text-13 font-medium text-secondary">
                      {t("project_cycles.active_cycle.progress")}
                    </div>
                  </div>
                )}
                <Transition show={open}>
                  <Disclosure.Panel className="flex flex-col divide-y divide-subtle-1">
                    {cycleStartDate && cycleEndDate ? (
                      <>
                        {isCycleDateValid && (
                          <SidebarChart
                            workspaceSlug={workspaceSlug}
                            projectId={projectId}
                            cycleId={cycleId}
                            isArchived={isArchived}
                          />
                        )}
                        {/* progress detailed view */}
                        {chartDistributionData && (
                          <div className="w-full py-4">
                            <CycleProgressStats
                              cycleId={cycleId}
                              distribution={chartDistributionData}
                              groupedIssues={groupedIssues}
                              handleFiltersUpdate={handleFiltersUpdate}
                              isEditable={Boolean(!peekCycle) && Boolean(filters)}
                              noBackground={false}
                              plotType={plotType}
                              roundedTab={false}
                              selectedFilters={{
                                assignees: selectedFilters.assignees,
                                labels: selectedFilters.labels,
                                stateGroups: selectedFilters.stateGroups,
                              }}
                              size="xs"
                              totalIssuesCount={estimateType === "points" ? totalEstimatePoints || 0 : totalIssues || 0}
                            />
                          </div>
                        )}
                      </>
                    ) : (
                      <div className="my-2 w-full rounded-md bg-surface-2 px-2 py-2 text-13 text-tertiary">
                        {t("no_data_yet")}
                      </div>
                    )}
                  </Disclosure.Panel>
                </Transition>
              </div>
            )}
          </Disclosure>
        </div>
      }
    </EstimateStatisticsBoundary>
  );
}
