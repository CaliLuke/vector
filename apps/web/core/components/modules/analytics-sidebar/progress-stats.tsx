/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TWorkItemFilterCondition } from "@plane/shared-state";
import type { TModuleDistribution, TModuleEstimateDistribution, TModulePlotType } from "@plane/types";
import type { TSelectedFilterProgressStats } from "@/components/core/sidebar/progress-stats/shared";
import { ProgressStats } from "@/components/core/sidebar/progress-stats/progress-stats";

type TModuleProgressStats = {
  distribution: TModuleDistribution | TModuleEstimateDistribution | undefined;
  groupedIssues: Record<string, number>;
  handleFiltersUpdate: (condition: TWorkItemFilterCondition) => void;
  isEditable?: boolean;
  moduleId: string;
  noBackground?: boolean;
  plotType: TModulePlotType;
  roundedTab?: boolean;
  selectedFilters: TSelectedFilterProgressStats;
  size?: "xs" | "sm";
  totalIssuesCount: number;
};

export const ModuleProgressStats = function ModuleProgressStats(props: TModuleProgressStats) {
  const {
    distribution,
    groupedIssues,
    handleFiltersUpdate,
    isEditable = false,
    moduleId,
    noBackground = false,
    roundedTab = false,
    selectedFilters,
    size = "sm",
    totalIssuesCount,
  } = props;

  return (
    <ProgressStats
      distribution={distribution}
      groupedIssues={groupedIssues}
      handleFiltersUpdate={handleFiltersUpdate}
      isEditable={isEditable}
      noBackground={noBackground}
      roundedTab={roundedTab}
      selectedFilters={selectedFilters}
      size={size}
      storageKey={`module-analytics-tab-${moduleId}`}
      totalIssuesCount={totalIssuesCount}
    />
  );
};
