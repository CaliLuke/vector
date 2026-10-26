/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TWorkItemFilterCondition } from "@plane/shared-state";
import type { TCycleDistribution, TCycleEstimateDistribution, TCyclePlotType } from "@plane/types";
import type { TSelectedFilterProgressStats } from "@/components/core/sidebar/progress-stats/shared";
import { ProgressStats } from "@/components/core/sidebar/progress-stats/progress-stats";

type TCycleProgressStats = {
  cycleId: string;
  distribution: TCycleDistribution | TCycleEstimateDistribution | undefined;
  groupedIssues: Record<string, number>;
  handleFiltersUpdate: (condition: TWorkItemFilterCondition) => void;
  isEditable?: boolean;
  noBackground?: boolean;
  plotType: TCyclePlotType;
  roundedTab?: boolean;
  selectedFilters: TSelectedFilterProgressStats;
  size?: "xs" | "sm";
  totalIssuesCount: number;
};

export const CycleProgressStats = function CycleProgressStats(props: TCycleProgressStats) {
  const {
    cycleId,
    distribution,
    groupedIssues,
    handleFiltersUpdate,
    isEditable = false,
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
      storageKey={`cycle-analytics-tab-${cycleId}`}
      totalIssuesCount={totalIssuesCount}
    />
  );
};
