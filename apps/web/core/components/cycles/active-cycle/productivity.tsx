/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Link } from "@tanstack/react-router";

import { useTheme } from "next-themes";
// plane imports
import { useTranslation } from "@plane/i18n";
import type { ICycle, TCycleEstimateType } from "@plane/types";
import { Loader } from "@plane/ui";
// assets
import darkChartAsset from "@/app/assets/empty-state/active-cycle/chart-dark.webp?url";
import lightChartAsset from "@/app/assets/empty-state/active-cycle/chart-light.webp?url";
// components
import ProgressChart from "@/components/core/sidebar/progress-chart";
import { SimpleEmptyState } from "@/components/empty-state/simple-empty-state-root";
// hooks
import { progressDistribution } from "@/components/estimates/progress-distribution";
import { useEstimateStatistics } from "@/hooks/use-estimate-statistics";
import { EstimateStatisticsBoundary } from "@/components/estimates/statistics-boundary";
import { useParentChartPreferences } from "@/providers/cycle-module-preferences";
import { EstimateTypeDropdown } from "../dropdowns/estimate-type-dropdown";

export type ActiveCycleProductivityProps = {
  workspaceSlug: string;
  projectId: string;
  cycle: ICycle | null;
};

type ProductivityContentProps = {
  cycle: ICycle;
  estimateType: TCycleEstimateType;
  distribution: NonNullable<ReturnType<typeof progressDistribution>>["completion_chart"];
  onChange: (value: TCycleEstimateType) => Promise<void>;
  assetPath: string;
  translate: ReturnType<typeof useTranslation>["t"];
  projectId: string;
  workspaceSlug: string;
};

function CycleProductivityContent(props: ProductivityContentProps) {
  const { cycle, estimateType, distribution, onChange, assetPath, translate, projectId, workspaceSlug } = props;
  const isPoints = estimateType === "points";
  const totalIssues = isPoints ? cycle.total_estimate_points || 0 : cycle.total_issues || 0;
  const pendingCount = isPoints
    ? cycle.backlog_estimate_points + cycle.unstarted_estimate_points + cycle.started_estimate_points
    : cycle.backlog_issues + cycle.unstarted_issues + cycle.started_issues;

  return (
    <div className="flex min-h-[17rem] flex-col gap-5 rounded-lg border border-subtle bg-surface-1 px-3.5 py-4">
      <div className="relative flex items-center justify-between gap-4">
        <Link
          to="/$workspaceSlug/projects/$projectId/cycles/$cycleId/"
          params={{ workspaceSlug, projectId, cycleId: cycle.id }}
        >
          <h3 className="text-14 font-semibold text-tertiary">
            {translate("project_cycles.active_cycle.issue_burndown")}
          </h3>
        </Link>
        <EstimateTypeDropdown value={estimateType} onChange={onChange} cycleId={cycle.id} projectId={projectId} />
      </div>
      <Link
        to="/$workspaceSlug/projects/$projectId/cycles/$cycleId/"
        params={{ workspaceSlug, projectId, cycleId: cycle.id }}
      >
        {cycle.total_issues > 0 ? (
          <div className="h-full w-full px-2">
            <div className="flex items-center justify-end gap-4 py-1 text-11 text-tertiary">
              <span>{isPoints ? `Pending points - ${pendingCount}` : `Pending work items - ${pendingCount}`}</span>
            </div>
            <ProgressChart
              distribution={distribution}
              totalIssues={totalIssues}
              plotTitle={isPoints ? "points" : "work items"}
            />
          </div>
        ) : (
          <div className="flex h-full w-full items-center justify-center">
            <SimpleEmptyState title={translate("active_cycle.empty_state.chart.title")} assetPath={assetPath} />
          </div>
        )}
      </Link>
    </div>
  );
}

export const ActiveCycleProductivity = function ActiveCycleProductivity(props: ActiveCycleProductivityProps) {
  const { workspaceSlug, projectId, cycle } = props;
  const estimateStats = useEstimateStatistics("cycle", workspaceSlug, projectId, cycle?.id);
  const { isEnabled: estimatesEnabled } = estimateStats;
  // theme hook
  const { resolvedTheme } = useTheme();
  // plane hooks
  const { t } = useTranslation();
  // hooks
  const chart = useParentChartPreferences("cycle", workspaceSlug, projectId, cycle?.id);
  // derived values
  const estimateType: TCycleEstimateType = estimatesEnabled && cycle ? chart.measure : "issues";
  const resolvedPath = resolvedTheme === "light" ? lightChartAsset : darkChartAsset;

  const onChange = async (value: TCycleEstimateType) => {
    if (!cycle?.id) return;
    chart.update({ measure: value });
  };

  const chartDistributionData = progressDistribution(cycle, estimateType);
  const completionChartDistributionData = chartDistributionData?.completion_chart || undefined;

  const content =
    cycle && completionChartDistributionData ? (
      <CycleProductivityContent
        cycle={cycle}
        estimateType={estimateType}
        distribution={completionChartDistributionData}
        onChange={onChange}
        assetPath={resolvedPath}
        translate={t}
        projectId={projectId}
        workspaceSlug={workspaceSlug}
      />
    ) : (
      <Loader className="flex min-h-[17rem] flex-col gap-5 rounded-lg border border-subtle bg-surface-1">
        <Loader.Item width="100%" height="100%" />
      </Loader>
    );
  return <EstimateStatisticsBoundary state={estimateStats.statistics}>{content}</EstimateStatisticsBoundary>;
};
