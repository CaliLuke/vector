/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
// plane package imports
import { useTranslation } from "@plane/i18n";
import { AreaChart } from "@plane/propel/charts/area-chart";
import { EmptyStateCompact } from "@plane/propel/empty-state";
import type { IChartResponse, TChartData } from "@plane/types";
import { renderFormattedDate } from "@plane/utils/datetime";
// hooks
import { useAnalyticsState } from "@/providers/analytics-state-provider";
import { analyticsChartOptions } from "@/queries/analytics";
// plane web components
import AnalyticsSectionWrapper from "../analytics-section-wrapper";
import { ChartLoader } from "../loaders";
import { AnalyticsAccessDenied, AnalyticsQueryError } from "../query-error";

const CreatedVsResolved = function CreatedVsResolved() {
  const { selectedDurationLabel, scope, requestFilters, isPeekView, isEpic } = useAnalyticsState();
  const { t } = useTranslation();
  const query = useQuery(analyticsChartOptions<IChartResponse>(scope, "work-items", requestFilters, isPeekView));
  const { data: createdVsResolvedData, isLoading: isCreatedVsResolvedLoading } = query;
  const canRead = isPeekView ? scope.hasProjectChartAccess : scope.hasWorkspaceAnalyticsAccess;
  const parsedData: TChartData<string, string>[] = useMemo(() => {
    if (!createdVsResolvedData?.data) return [];
    return createdVsResolvedData.data.map((datum) => ({
      ...datum,
      [datum.key]: datum.count,
      name: renderFormattedDate(datum.key) ?? datum.key,
    }));
  }, [createdVsResolvedData]);

  const areas = useMemo(
    () => [
      {
        key: "completed_issues",
        label: "Resolved",
        fill: "#19803833",
        fillOpacity: 1,
        stackId: "bar-one",
        showDot: false,
        smoothCurves: true,
        strokeColor: "#198038",
        strokeOpacity: 1,
      },
      {
        key: "created_issues",
        label: "Created",
        fill: "#1192E833",
        fillOpacity: 1,
        stackId: "bar-one",
        showDot: false,
        smoothCurves: true,
        strokeColor: "#1192E8",
        strokeOpacity: 1,
      },
    ],
    []
  );

  return (
    <AnalyticsSectionWrapper
      title={t("workspace_analytics.created_vs_resolved")}
      subtitle={selectedDurationLabel}
      className="col-span-1"
    >
      {!canRead ? (
        <AnalyticsAccessDenied />
      ) : query.isError ? (
        <AnalyticsQueryError onRetry={() => void query.refetch()} />
      ) : isCreatedVsResolvedLoading ? (
        <ChartLoader />
      ) : parsedData && parsedData.length > 0 ? (
        <AreaChart
          className="h-[350px] w-full"
          data={parsedData}
          areas={areas}
          xAxis={{
            key: "name",
            label: t("date"),
          }}
          yAxis={{
            key: "count",
            label: t("common.no_of", { entity: isEpic ? t("common.epics") : t("work_items") }),
            offset: -60,
            dx: -24,
          }}
          legend={{
            align: "left",
            verticalAlign: "bottom",
            layout: "horizontal",
            wrapperStyles: {
              justifyContent: "start",
              alignContent: "start",
              paddingLeft: "40px",
              paddingTop: "10px",
            },
          }}
        />
      ) : (
        <EmptyStateCompact
          assetKey="unknown"
          assetClassName="size-20"
          rootClassName="border border-subtle px-5 py-10 md:py-20 md:px-20"
          title={t("workspace_empty_state.analytics_work_items.title")}
        />
      )}
    </AnalyticsSectionWrapper>
  );
};

export default CreatedVsResolved;
