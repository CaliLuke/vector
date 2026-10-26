/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useLocation } from "@tanstack/react-router";
import { useCallback, useMemo, useState } from "react";

import { isValid } from "date-fns/isValid";
import { parseISO } from "date-fns/parseISO";
import { AlertCircle } from "lucide-react";
import { Disclosure, Transition } from "@headlessui/react";
import { useTranslation } from "@plane/i18n";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { ChevronUpIcon, ChevronDownIcon } from "@plane/propel/icons";
import { EEstimateSystem, type IModule, type TModulePlotType, type TWorkItemFilterExpression } from "@plane/types";
import { CustomSelect } from "@plane/ui";
import type { TWorkItemFilterCondition } from "@plane/shared-state";
import { updateWorkItemFilterFromSidebar } from "@plane/shared-state";
// components
// constants
// helpers
import { getDate, isValidDate } from "@plane/utils/datetime";
import ProgressChart from "@/components/core/sidebar/progress-chart";
import { ModuleProgressStats } from "@/components/modules";
import type { TSelectedFilterProgressStats } from "@/components/core/sidebar/progress-stats/shared";
// hooks
import { progressCount, progressDistribution } from "@/components/estimates/progress-distribution";
import { useEstimateStatistics } from "@/hooks/use-estimate-statistics";
import { EstimateStatisticsBoundary } from "@/components/estimates/statistics-boundary";
import { useModuleDetails } from "@/hooks/use-modules";
import { useParentChartPreferences } from "@/providers/cycle-module-preferences";
import { useParentIssuePreferences } from "@/providers/parent-issue-preferences";
import { readProgressFilterSelection } from "@/components/core/sidebar/progress-filter-selection";
// plane web constants
type TModuleAnalyticsProgress = {
  workspaceSlug: string;
  projectId: string;
  moduleId: string;
  isArchived?: boolean;
};

const moduleBurnDownChartOptions = [
  { value: "burndown", i18n_label: "issues" },
  { value: "points", i18n_label: "points" },
];

type Translation = ReturnType<typeof useTranslation>["t"];

type ModuleProgressDateState = {
  isValid: boolean;
  isInvalid: boolean;
  shouldShowAlert: boolean;
  messageKey: string;
};

function getModuleProgressDateState(
  module: IModule,
  startDate: Date | undefined,
  endDate: Date | undefined,
  today: Date
): ModuleProgressDateState {
  const isInvalid = hasInvalidModuleDate(module, startDate, endDate);
  const isDateRangeValid =
    startDate !== undefined && endDate !== undefined && startDate <= today && endDate >= startDate;
  const hasBothDates = Boolean(module.start_date && module.target_date);
  return {
    isValid: isDateRangeValid && !isInvalid,
    isInvalid,
    shouldShowAlert: isInvalid || hasBothDates,
    messageKey: getModuleProgressMessageKey(isInvalid, hasBothDates),
  };
}

function hasInvalidModuleDate(module: IModule, startDate: Date | undefined, endDate: Date | undefined) {
  return Boolean(
    (module.start_date && (!startDate || !isValidModuleDate(module.start_date))) ||
    (module.target_date && (!endDate || !isValidModuleDate(module.target_date))) ||
    (startDate && endDate && endDate < startDate)
  );
}

function getModuleProgressMessageKey(isInvalid: boolean, hasBothDates: boolean) {
  if (isInvalid) return "project_module.empty_state.sidebar.invalid_date";
  return hasBothDates ? "project_module.empty_state.sidebar.in_active" : "project_cycles.add_date";
}

function getProgressPercentage(completed: number, total: number) {
  return completed !== 0 && total !== 0 ? Math.round((completed / total) * 100) : 0;
}

function getModulePlotType(estimatesEnabled: boolean, measure: TModulePlotType): TModulePlotType {
  return estimatesEnabled && measure === "points" ? "points" : "burndown";
}

function updateModuleFilter(
  richFilters: TWorkItemFilterExpression | undefined,
  condition: TWorkItemFilterCondition,
  setRich: ReturnType<typeof useParentIssuePreferences>["setRich"],
  translate: Translation
) {
  const result = updateWorkItemFilterFromSidebar(richFilters, condition);
  if (!result.ok) {
    setToast({ type: TOAST_TYPE.ERROR, title: translate("common.error.label"), message: result.error });
    return;
  }
  void setRich(result.value).catch(() => {
    setToast({
      type: TOAST_TYPE.ERROR,
      title: translate("common.error.label"),
      message: translate("common.something_went_wrong"),
    });
  });
}

type ModuleProgressHeaderProps = {
  isDateValid: boolean;
  showDateAlert: boolean;
  dateMessageKey: string;
  percentage: number;
  isPointsEstimate: boolean;
  plotType: TModulePlotType;
  onChange: (value: TModulePlotType) => void;
  isOpen: boolean;
  translate: Translation;
};

function ModuleProgressHeader(props: ModuleProgressHeaderProps) {
  const {
    isDateValid,
    showDateAlert,
    dateMessageKey,
    percentage,
    isPointsEstimate,
    plotType,
    onChange,
    isOpen,
    translate,
  } = props;

  if (!isDateValid)
    return (
      <div className="relative flex w-full items-center justify-between gap-2">
        <div className="text-13 font-medium text-secondary">Progress</div>
        <div className="flex items-center gap-1">
          {showDateAlert && <AlertCircle height={14} width={14} className="text-secondary" />}
          <span className="text-11 text-secondary italic">{translate(dateMessageKey)}</span>
        </div>
      </div>
    );

  return (
    <div className="relative flex w-full items-center justify-between gap-2">
      <Disclosure.Button className="relative flex w-full items-center gap-2">
        <div className="text-13 font-medium text-secondary">{translate("common.progress")}</div>
        {percentage > 0 && (
          <div className="bg-amber-500/20 text-amber-500 flex h-5 w-9 items-center justify-center rounded-sm text-11 font-medium">{`${percentage}%`}</div>
        )}
      </Disclosure.Button>
      {isPointsEstimate && (
        <CustomSelect
          value={plotType}
          label={
            <span>
              {translate(moduleBurnDownChartOptions.find((option) => option.value === plotType)?.i18n_label || "none")}
            </span>
          }
          onChange={onChange}
          maxHeight="lg"
        >
          {moduleBurnDownChartOptions.map((item) => (
            <CustomSelect.Option key={item.value} value={item.value}>
              {translate(item.i18n_label)}
            </CustomSelect.Option>
          ))}
        </CustomSelect>
      )}
      <Disclosure.Button className="ml-auto">
        {isOpen ? (
          <ChevronUpIcon className="h-3.5 w-3.5" aria-hidden="true" />
        ) : (
          <ChevronDownIcon className="h-3.5 w-3.5" aria-hidden="true" />
        )}
      </Disclosure.Button>
    </div>
  );
}

type ModuleProgressChartProps = {
  startDate: Date | undefined;
  endDate: Date | undefined;
  isDateInvalid: boolean;
  distribution: ReturnType<typeof progressDistribution>;
  plotType: TModulePlotType;
  totalIssues: number;
  totalEstimatePoints: number;
};

function ModuleProgressChart(props: ModuleProgressChartProps) {
  const { startDate, endDate, isDateInvalid, distribution, plotType, totalIssues, totalEstimatePoints } = props;
  if (!startDate || !endDate || isDateInvalid || !distribution?.completion_chart) return null;

  return (
    <ProgressChart
      distribution={distribution.completion_chart}
      totalIssues={plotType === "points" ? totalEstimatePoints : totalIssues}
      plotTitle={plotType === "points" ? "points" : "work items"}
    />
  );
}

type ModuleProgressDetailsProps = {
  distribution: ReturnType<typeof progressDistribution>;
  groupedIssues: ReturnType<typeof getGroupedModuleIssues>;
  handleFiltersUpdate: (condition: TWorkItemFilterCondition) => void;
  isEditable: boolean;
  moduleId: string;
  plotType: TModulePlotType;
  selectedFilters: TSelectedFilterProgressStats;
  totalIssues: number;
  totalEstimatePoints: number;
};

function ModuleProgressDetails(props: ModuleProgressDetailsProps) {
  const {
    distribution,
    groupedIssues,
    handleFiltersUpdate,
    isEditable,
    moduleId,
    plotType,
    selectedFilters,
    totalIssues,
    totalEstimatePoints,
  } = props;
  if (!distribution) return null;

  return (
    <div className="w-full border-t border-subtle pt-5">
      <ModuleProgressStats
        distribution={distribution}
        groupedIssues={groupedIssues}
        handleFiltersUpdate={handleFiltersUpdate}
        isEditable={isEditable}
        moduleId={moduleId}
        noBackground={false}
        plotType={plotType}
        roundedTab={false}
        selectedFilters={selectedFilters}
        size="xs"
        totalIssuesCount={plotType === "points" ? totalEstimatePoints : totalIssues}
      />
    </div>
  );
}

function isValidModuleDate(date: string | null | undefined) {
  return !date || (isValidDate(date) && isValid(parseISO(date)));
}

function getGroupedModuleIssues(module: IModule | undefined, plotType: TModulePlotType) {
  const estimateMode = plotType === "points";
  return {
    backlog: progressCount(module?.backlog_estimate_points, module?.backlog_issues, estimateMode),
    unstarted: progressCount(module?.unstarted_estimate_points, module?.unstarted_issues, estimateMode),
    started: progressCount(module?.started_estimate_points, module?.started_issues, estimateMode),
    completed: progressCount(module?.completed_estimate_points, module?.completed_issues, estimateMode),
    cancelled: progressCount(module?.cancelled_estimate_points, module?.cancelled_issues, estimateMode),
  };
}

function getModuleProgressModel(
  module: IModule | undefined,
  estimatesEnabled: boolean,
  measure: TModulePlotType,
  isPointsEstimate: boolean,
  today: Date
) {
  const plotType = getModulePlotType(estimatesEnabled, measure);
  const totalIssues = module?.total_issues ?? 0;
  const totalEstimatePoints = module?.total_estimate_points ?? 0;
  const completedIssues = module?.completed_issues ?? 0;
  const completedEstimatePoints = module?.completed_estimate_points ?? 0;
  const startDate = getDate(module?.start_date);
  const endDate = getDate(module?.target_date);

  return {
    plotType,
    isPointsEstimate,
    totalIssues,
    totalEstimatePoints,
    percentage:
      plotType === "points"
        ? getProgressPercentage(completedEstimatePoints, totalEstimatePoints)
        : getProgressPercentage(completedIssues, totalIssues),
    distribution: progressDistribution(module ?? null, plotType),
    groupedIssues: getGroupedModuleIssues(module, plotType),
    startDate,
    endDate,
    dateState: module ? getModuleProgressDateState(module, startDate, endDate, today) : undefined,
  };
}

export function ModuleAnalyticsProgress(props: TModuleAnalyticsProgress) {
  // props
  const { workspaceSlug, projectId, moduleId, isArchived = false } = props;
  // router
  const searchParams = new URLSearchParams(useLocation({ select: (location) => location.searchStr }));
  const peekModule = searchParams.get("peekModule") || undefined;
  // plane hooks
  const { t } = useTranslation();
  // hooks
  const estimateStats = useEstimateStatistics("module", workspaceSlug, projectId, moduleId, isArchived);
  const { isEnabled: areEstimateEnabled, activeEstimate: estimateDetails } = estimateStats;
  const { module: moduleDetails } = useModuleDetails(workspaceSlug, projectId, moduleId, isArchived);
  const chart = useParentChartPreferences("module", workspaceSlug, projectId, moduleId);
  const preferences = useParentIssuePreferences();
  const { setRich, filters } = preferences;
  // state
  // derived values
  const richFilters = filters?.richFilters;
  const selectedFilters = useMemo(() => readProgressFilterSelection(richFilters), [richFilters]);
  const handleFiltersUpdate = useCallback(
    (condition: TWorkItemFilterCondition) => updateModuleFilter(richFilters, condition, setRich, t),
    [setRich, richFilters, t]
  );
  const [today] = useState(() => new Date());
  const progress = getModuleProgressModel(
    moduleDetails ?? undefined,
    areEstimateEnabled,
    chart.measure === "points" ? "points" : "burndown",
    estimateDetails?.type === EEstimateSystem.POINTS,
    today
  );

  // handlers
  const onChange = (value: TModulePlotType) => {
    chart.update({ measure: value === "points" ? "points" : "issues" });
  };

  const dateState = progress.dateState;
  if (!moduleDetails || !dateState) return null;
  return (
    <EstimateStatisticsBoundary state={estimateStats.statistics}>
      <div className="space-y-4 border-t border-subtle px-3 py-4">
        <Disclosure defaultOpen={dateState.isValid}>
          {({ open }) => (
            <div className="space-y-6">
              <ModuleProgressHeader
                isDateValid={dateState.isValid}
                showDateAlert={dateState.shouldShowAlert}
                dateMessageKey={dateState.messageKey}
                percentage={progress.percentage}
                isPointsEstimate={progress.isPointsEstimate}
                plotType={progress.plotType}
                onChange={onChange}
                isOpen={open}
                translate={t}
              />
              <Transition show={open}>
                <Disclosure.Panel className="space-y-4">
                  <ModuleProgressChart
                    startDate={progress.startDate}
                    endDate={progress.endDate}
                    isDateInvalid={dateState.isInvalid}
                    distribution={progress.distribution}
                    plotType={progress.plotType}
                    totalIssues={progress.totalIssues}
                    totalEstimatePoints={progress.totalEstimatePoints}
                  />
                  <ModuleProgressDetails
                    distribution={progress.distribution}
                    groupedIssues={progress.groupedIssues}
                    handleFiltersUpdate={handleFiltersUpdate}
                    isEditable={!peekModule && Boolean(filters)}
                    moduleId={moduleId}
                    plotType={progress.plotType}
                    selectedFilters={{
                      assignees: selectedFilters.assignees,
                      labels: selectedFilters.labels,
                      stateGroups: selectedFilters.stateGroups,
                    }}
                    totalIssues={progress.totalIssues}
                    totalEstimatePoints={progress.totalEstimatePoints}
                  />
                </Disclosure.Panel>
              </Transition>
            </div>
          )}
        </Disclosure>
      </div>
    </EstimateStatisticsBoundary>
  );
}
