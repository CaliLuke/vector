/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Tab } from "@headlessui/react";

import { useTranslation } from "@plane/i18n";
import type { TWorkItemFilterCondition } from "@plane/shared-state";
import { cn } from "@plane/utils/common";
import { toFilterArray } from "@plane/utils/rich-filters";
import useLocalStorage from "@/hooks/use-local-storage";
import { AssigneeStatComponent, type TAssigneeData } from "./assignee";
import { LabelStatComponent, type TLabelData } from "./label";
import { PROGRESS_STATS, createFilterUpdateHandler, type TSelectedFilterProgressStats } from "./shared";
import { StateGroupStatComponent, type TStateGroupData } from "./state_group";

type TProgressMetricData =
  | { completed_issues: number; total_issues: number }
  | { completed_estimates: number; total_estimates: number };

type TAssigneeDistribution = TProgressMetricData & {
  assignee_id: string | null;
  display_name: string | null;
  avatar_url: string | null;
};

type TLabelDistribution = TProgressMetricData & {
  label_id: string | null;
  label_name: string | null;
  color: string | null;
};

type TProgressDistribution = {
  assignees?: TAssigneeDistribution[];
  labels?: TLabelDistribution[];
};

type TProgressStatsProps = {
  distribution: TProgressDistribution | undefined;
  groupedIssues: Record<string, number>;
  handleFiltersUpdate: (condition: TWorkItemFilterCondition) => void;
  isEditable: boolean;
  noBackground: boolean;
  roundedTab: boolean;
  selectedFilters: TSelectedFilterProgressStats;
  size: "xs" | "sm";
  storageKey: string;
  totalIssuesCount: number;
};

function progressCounts(data: TProgressMetricData) {
  return "completed_issues" in data
    ? { completed: data.completed_issues, total: data.total_issues }
    : { completed: data.completed_estimates, total: data.total_estimates };
}

function getProgressStatsDistribution(
  distribution: TProgressDistribution | undefined,
  groupedIssues: Record<string, number>,
  totalIssuesCount: number
): { assignees: TAssigneeData; labels: TLabelData; states: TStateGroupData } {
  return {
    assignees: (distribution?.assignees || []).map((assignee) => {
      const { completed, total } = progressCounts(assignee);
      return {
        id: assignee.assignee_id || undefined,
        title: assignee.display_name || undefined,
        avatar_url: assignee.avatar_url || undefined,
        completed,
        total,
      };
    }),
    labels: (distribution?.labels || []).map((label) => {
      const { completed, total } = progressCounts(label);
      return {
        id: label.label_id || undefined,
        title: label.label_name || undefined,
        color: label.color || undefined,
        completed,
        total,
      };
    }),
    states: Object.keys(groupedIssues || {}).map((state) => ({
      state,
      completed: groupedIssues[state] || 0,
      total: totalIssuesCount || 0,
    })),
  };
}

export const ProgressStats = function ProgressStats(props: TProgressStatsProps) {
  const {
    distribution,
    groupedIssues,
    handleFiltersUpdate,
    isEditable,
    noBackground,
    roundedTab,
    selectedFilters,
    size,
    storageKey,
    totalIssuesCount,
  } = props;
  const { t } = useTranslation();
  const { storedValue: currentTab, setValue: setCurrentTab } = useLocalStorage(storageKey, "stat-assignees");
  const { assignees, labels, states } = getProgressStatsDistribution(distribution, groupedIssues, totalIssuesCount);
  const selectedAssigneeIds = toFilterArray(selectedFilters?.assignees?.value || []).filter(
    (value) => typeof value === "string"
  );
  const selectedLabelIds = toFilterArray(selectedFilters?.labels?.value || []).filter(
    (value) => typeof value === "string"
  );
  const selectedStateGroups = toFilterArray(selectedFilters?.stateGroups?.value || []).filter(
    (value) => typeof value === "string"
  );

  const handleAssigneeFiltersUpdate = createFilterUpdateHandler(
    "assignee_id",
    selectedAssigneeIds,
    handleFiltersUpdate
  );
  const handleLabelFiltersUpdate = createFilterUpdateHandler("label_id", selectedLabelIds, handleFiltersUpdate);
  const handleStateGroupFiltersUpdate = createFilterUpdateHandler(
    "state_group",
    selectedStateGroups,
    handleFiltersUpdate
  );

  return (
    <div>
      <Tab.Group defaultIndex={PROGRESS_STATS.findIndex((stat) => stat.key === (currentTab || "stat-assignees"))}>
        <Tab.List
          as="div"
          className={cn(
            `flex w-full items-center justify-between gap-2 rounded-md p-1`,
            roundedTab ? `rounded-3xl` : `rounded-md`,
            noBackground ? `` : `bg-layer-2`,
            size === "xs" ? `text-11` : `text-13`
          )}
        >
          {PROGRESS_STATS.map((stat) => (
            <Tab
              className={cn(
                `w-full cursor-pointer p-1 text-primary transition-all outline-none focus:outline-none`,
                roundedTab ? `rounded-3xl border border-subtle` : `rounded-sm`,
                stat.key === currentTab
                  ? "bg-layer-transparent-active text-secondary"
                  : "text-placeholder hover:text-secondary"
              )}
              key={stat.key}
              onClick={() => setCurrentTab(stat.key)}
            >
              {t(stat.i18n_title)}
            </Tab>
          ))}
        </Tab.List>
        <Tab.Panels className="py-3 text-secondary">
          {PROGRESS_STATS.map((stat) => {
            if (stat.key === "stat-states") {
              return (
                <Tab.Panel key={stat.key}>
                  <StateGroupStatComponent
                    distribution={states}
                    handleStateGroupFiltersUpdate={handleStateGroupFiltersUpdate}
                    isEditable={isEditable}
                    selectedStateGroups={selectedStateGroups}
                    totalIssuesCount={totalIssuesCount}
                  />
                </Tab.Panel>
              );
            }

            if (stat.key === "stat-assignees") {
              return (
                <Tab.Panel key={stat.key}>
                  <AssigneeStatComponent
                    distribution={assignees}
                    handleAssigneeFiltersUpdate={handleAssigneeFiltersUpdate}
                    isEditable={isEditable}
                    selectedAssigneeIds={selectedAssigneeIds}
                  />
                </Tab.Panel>
              );
            }

            return (
              <Tab.Panel key={stat.key}>
                <LabelStatComponent
                  distribution={labels}
                  handleLabelFiltersUpdate={handleLabelFiltersUpdate}
                  isEditable={isEditable}
                  selectedLabelIds={selectedLabelIds}
                />
              </Tab.Panel>
            );
          })}
        </Tab.Panels>
      </Tab.Group>
    </div>
  );
};
