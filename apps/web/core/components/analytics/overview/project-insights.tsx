/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { lazy, Suspense } from "react";

import { useQuery } from "@tanstack/react-query";
// plane package imports
import { useTranslation } from "@plane/i18n";
import { EmptyStateCompact } from "@plane/propel/empty-state";
import type { TChartData } from "@plane/types";
import { useAnalyticsState } from "@/providers/analytics-state-provider";
import { analyticsChartOptions } from "@/queries/analytics";
// plane web components
import AnalyticsSectionWrapper from "../analytics-section-wrapper";
import { ProjectInsightsLoader } from "../loaders";
import { AnalyticsAccessDenied, AnalyticsQueryError } from "../query-error";

const RadarChart = lazy(function RadarChart() {
  return import("@plane/propel/charts/radar-chart").then((mod) => ({
    default: mod.RadarChart,
  }));
});

const ProjectInsights = function ProjectInsights() {
  const { t } = useTranslation();
  const { scope, requestFilters, selectedDurationLabel, isPeekView } = useAnalyticsState();
  const query = useQuery(
    analyticsChartOptions<TChartData<string, string>[]>(scope, "projects", requestFilters, isPeekView)
  );
  const { data: projectInsightsData, isLoading: isLoadingProjectInsight } = query;
  const canRead = isPeekView ? scope.hasProjectChartAccess : scope.hasWorkspaceAnalyticsAccess;

  return (
    <AnalyticsSectionWrapper
      title={`${t("workspace_analytics.project_insights")}`}
      subtitle={selectedDurationLabel}
      className="md:col-span-3"
    >
      {!canRead ? (
        <AnalyticsAccessDenied />
      ) : query.isError ? (
        <AnalyticsQueryError onRetry={() => void query.refetch()} />
      ) : isLoadingProjectInsight ? (
        <ProjectInsightsLoader />
      ) : projectInsightsData && projectInsightsData?.length == 0 ? (
        <EmptyStateCompact
          assetKey="unknown"
          assetClassName="size-20"
          rootClassName="border border-subtle px-5 py-10 md:py-20 md:px-20"
          title={t("workspace_empty_state.analytics_work_items.title")}
        />
      ) : (
        <div className="gap-8 lg:flex">
          {projectInsightsData && (
            <Suspense fallback={<ProjectInsightsLoader />}>
              <RadarChart
                className="h-[350px] w-full text-accent-primary lg:w-3/5"
                data={projectInsightsData}
                dataKey="key"
                radars={[
                  {
                    key: "count",
                    name: "Count",
                    fill: "var(--text-color-accent-primary)",
                    stroke: "var(--text-color-accent-primary)",
                    fillOpacity: 0.6,
                    dot: {
                      r: 4,
                      fillOpacity: 1,
                    },
                  },
                ]}
                margin={{ top: 0, right: 40, bottom: 10, left: 40 }}
                showTooltip
                angleAxis={{
                  key: "name",
                }}
              />
            </Suspense>
          )}
          <div className="w-full lg:w-2/5">
            <div className="text-13 text-tertiary">{t("workspace_analytics.summary_of_projects")}</div>
            <div className="mb-3 border-b border-subtle py-2">{t("workspace_analytics.all_projects")}</div>
            <div className="flex flex-col gap-4">
              <div className="flex items-center justify-between text-13 text-tertiary">
                <div>{t("workspace_analytics.trend_on_charts")}</div>
                <div>{t("common.work_items")}</div>
              </div>
              {projectInsightsData?.map((item) => (
                <div key={item.key} className="flex items-center justify-between text-13 text-primary">
                  <div>{item.name}</div>
                  <div className="flex items-center gap-1">
                    <div className="text-secondary">{item.count}</div>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}
    </AnalyticsSectionWrapper>
  );
};

export default ProjectInsights;
