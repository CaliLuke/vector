/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// plane package imports
import { useQuery } from "@tanstack/react-query";
import type { IInsightField } from "@plane/constants";
import { ANALYTICS_INSIGHTS_FIELDS } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import type { TAnalyticsTabsBase } from "@plane/types";
import { cn } from "@plane/utils/common";
// hooks
import { useAnalyticsState } from "@/providers/analytics-state-provider";
import { analyticsInsightsOptions } from "@/queries/analytics";
import { AnalyticsAccessDenied, AnalyticsQueryError } from "./query-error";
// local imports
import InsightCard from "./insight-card";

const getInsightLabel = (
  analyticsType: TAnalyticsTabsBase,
  item: IInsightField,
  isEpic: boolean | undefined,
  t: (key: string, params?: Record<string, unknown>) => string
) => {
  if (analyticsType === "work-items") {
    return isEpic
      ? t(item.i18nKey, { entity: t("common.epics") })
      : t(item.i18nKey, { entity: t("common.work_items") });
  }

  // Get the base translation with entity
  const baseTranslation = t(item.i18nKey, {
    ...item.i18nProps,
    entity: item.i18nProps?.entity && t(item.i18nProps?.entity),
  });

  // Add prefix if available
  const prefix = item.i18nProps?.prefix ? `${t(item.i18nProps.prefix)} ` : "";

  // Add suffix if available
  const suffix = item.i18nProps?.suffix ? ` ${t(item.i18nProps.suffix)}` : "";

  // Combine prefix, base translation, and suffix
  return `${prefix}${baseTranslation}${suffix}`;
};

const TotalInsights = function TotalInsights({
  analyticsType,
  peekView,
}: {
  analyticsType: TAnalyticsTabsBase;
  peekView?: boolean;
}) {
  const { t } = useTranslation();
  const { scope, requestFilters, isPeekView, isEpic } = useAnalyticsState();
  const query = useQuery(analyticsInsightsOptions(scope, analyticsType, requestFilters, isPeekView));
  const { data: totalInsightsData, isLoading } = query;
  const canRead = isPeekView ? scope.hasProjectAnalyticsAccess : scope.hasWorkspaceAnalyticsAccess;
  return (
    <div
      className={cn(
        "grid grid-cols-1 gap-8 sm:grid-cols-2 md:gap-10",
        !peekView
          ? ANALYTICS_INSIGHTS_FIELDS[analyticsType]?.length % 5 === 0
            ? "gap-10 lg:grid-cols-5"
            : "gap-8 lg:grid-cols-4"
          : "grid-cols-2"
      )}
    >
      {!canRead ? (
        <AnalyticsAccessDenied />
      ) : query.isError ? (
        <AnalyticsQueryError onRetry={() => void query.refetch()} />
      ) : null}
      {canRead &&
        !query.isError &&
        ANALYTICS_INSIGHTS_FIELDS[analyticsType]?.map((item) => (
          <InsightCard
            key={`${analyticsType}-${item.key}`}
            isLoading={isLoading}
            data={totalInsightsData?.[item.key]}
            label={getInsightLabel(analyticsType, item, isEpic, t)}
          />
        ))}
    </div>
  );
};

export default TotalInsights;
