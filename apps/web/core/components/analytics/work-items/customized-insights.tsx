/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
// plane package imports
import { useTranslation } from "@plane/i18n";
import type { IAnalyticsParams } from "@plane/types";
import { ChartXAxisProperty, ChartYAxisMetric } from "@plane/types";
import { cn } from "@plane/utils/common";
// plane web components
import AnalyticsSectionWrapper from "../analytics-section-wrapper";
import { AnalyticsSelectParams } from "../select/analytics-params";
import PriorityChart from "./priority-chart";

const CustomizedInsights = function CustomizedInsights({ peekView, isEpic }: { peekView?: boolean; isEpic?: boolean }) {
  const { t } = useTranslation();
  const [params, setParams] = useState<IAnalyticsParams>({
    x_axis: ChartXAxisProperty.PRIORITY,
    y_axis: isEpic ? ChartYAxisMetric.EPIC_WORK_ITEM_COUNT : ChartYAxisMetric.WORK_ITEM_COUNT,
  });

  return (
    <AnalyticsSectionWrapper
      title={t("workspace_analytics.customized_insights")}
      className="col-span-1"
      headerClassName={cn(peekView ? "flex-col items-start" : "")}
      actions={
        <AnalyticsSelectParams
          onChange={(value) => setParams((current) => ({ ...current, ...value }))}
          params={params}
          isEpic={isEpic}
        />
      }
    >
      <PriorityChart x_axis={params.x_axis} y_axis={params.y_axis} group_by={params.group_by} />
    </AnalyticsSectionWrapper>
  );
};

export default CustomizedInsights;
