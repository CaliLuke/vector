/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// plane imports
import { useTranslation } from "@plane/i18n";
import type { ICycle, TCycleEstimateType } from "@plane/types";
import { Loader } from "@plane/ui";
import { getDate } from "@plane/utils/datetime";
// components
import ProgressChart from "@/components/core/sidebar/progress-chart";
import { validateCycleSnapshot } from "@/components/cycles/analytics-sidebar/issue-progress";
import { EstimateTypeDropdown } from "@/components/cycles/dropdowns";
// hooks
import { progressDistribution } from "@/components/estimates/progress-distribution";
import { useProjectEstimates } from "@/hooks/use-estimates";
import { useCycleDetails } from "@/hooks/use-cycles";
import { useParentChartPreferences } from "@/providers/cycle-module-preferences";

type ProgressChartProps = {
  workspaceSlug: string;
  projectId: string;
  cycleId: string;
  isArchived?: boolean;
};

type CycleProgressChartProps = {
  cycle: ICycle | null;
  estimateType: TCycleEstimateType;
  translate: ReturnType<typeof useTranslation>["t"];
};

function CycleProgressChartContent(props: CycleProgressChartProps) {
  const { cycle, estimateType, translate } = props;
  const cycleDetails = validateCycleSnapshot(cycle);
  const startDate = getDate(cycleDetails?.start_date);
  const endDate = getDate(cycleDetails?.end_date);
  const distribution = progressDistribution(cycleDetails, estimateType)?.completion_chart;

  if (!cycleDetails || !startDate || !endDate || !distribution)
    return (
      <Loader className="mt-4 h-[160px] w-full">
        <Loader.Item width="100%" height="100%" />
      </Loader>
    );

  const total = estimateType === "points" ? cycleDetails.total_estimate_points || 0 : cycleDetails.total_issues || 0;
  return (
    <ProgressChart
      distribution={distribution}
      totalIssues={total}
      plotTitle={estimateType === "points" ? translate("points") : translate("work_items")}
    />
  );
}

export const SidebarChart = function SidebarChart(props: ProgressChartProps) {
  const { workspaceSlug, projectId, cycleId, isArchived = false } = props;
  const { isEnabled: estimatesEnabled } = useProjectEstimates(workspaceSlug, projectId);

  // hooks
  const { cycle } = useCycleDetails(workspaceSlug, projectId, cycleId, isArchived);
  const chart = useParentChartPreferences("cycle", workspaceSlug, projectId, cycleId);
  const { t } = useTranslation();

  const estimateType = estimatesEnabled ? chart.measure : "issues";

  if (!workspaceSlug || !projectId || !cycleId) return null;

  // handlers
  const onChange = async (value: TCycleEstimateType) => {
    chart.update({ measure: value });
  };
  return (
    <div>
      <div className="relative flex items-center justify-between gap-2 pt-4">
        <EstimateTypeDropdown
          value={estimateType}
          onChange={onChange}
          cycleId={cycleId}
          projectId={projectId}
          isArchived={isArchived}
        />
      </div>
      <div className="py-4">
        <CycleProgressChartContent cycle={cycle ?? null} estimateType={estimateType} translate={t} />
      </div>
    </div>
  );
};
