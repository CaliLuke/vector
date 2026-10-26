/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useMemo } from "react";
import { SlidersHorizontal } from "lucide-react";
// plane package imports
import { ANALYTICS_X_AXIS_VALUES, ANALYTICS_Y_AXIS_VALUES } from "@plane/constants";
import { CalendarLayoutIcon } from "@plane/propel/icons";
import type { IAnalyticsParams } from "@plane/types";
import { ChartYAxisMetric } from "@plane/types";
import { cn } from "@plane/utils/common";
// plane web components
import { SelectXAxis } from "./select-x-axis";
import { SelectYAxis } from "./select-y-axis";

type Props = {
  onChange: (value: Partial<IAnalyticsParams>) => void;
  params: IAnalyticsParams;
  classNames?: string;
  isEpic?: boolean;
};

export const AnalyticsSelectParams = function AnalyticsSelectParams(props: Props) {
  const { onChange, params, classNames, isEpic } = props;
  const xAxisOptions = useMemo(
    () => ANALYTICS_X_AXIS_VALUES.filter((option) => option.value !== params.group_by),
    [params.group_by]
  );
  const groupByOptions = useMemo(
    () => ANALYTICS_X_AXIS_VALUES.filter((option) => option.value !== params.x_axis),
    [params.x_axis]
  );

  return (
    <div className={cn("flex w-full justify-between", classNames)}>
      <div className={`flex items-center gap-2`}>
        <SelectYAxis
          value={params.y_axis}
          onChange={(val: ChartYAxisMetric | null) => {
            if (val !== null) onChange({ y_axis: val });
          }}
          options={ANALYTICS_Y_AXIS_VALUES}
          hiddenOptions={[
            ChartYAxisMetric.ESTIMATE_POINT_COUNT,
            isEpic ? ChartYAxisMetric.WORK_ITEM_COUNT : ChartYAxisMetric.EPIC_WORK_ITEM_COUNT,
          ]}
        />
        <SelectXAxis
          value={params.x_axis}
          onChange={(val) => {
            if (val !== null) onChange({ x_axis: val });
          }}
          label={
            <div className="flex items-center gap-2">
              <CalendarLayoutIcon className="h-3 w-3" />
              <span className={cn("text-secondary", params.x_axis && "text-primary")}>
                {xAxisOptions.find((v) => v.value === params.x_axis)?.label || "Add Property"}
              </span>
            </div>
          }
          options={xAxisOptions}
        />
        <SelectXAxis
          value={params.group_by}
          onChange={(val) => {
            onChange({ group_by: val ?? undefined });
          }}
          label={
            <div className="flex items-center gap-2">
              <SlidersHorizontal className="h-3 w-3" />
              <span className={cn("text-secondary", params.group_by && "text-primary")}>
                {groupByOptions.find((v) => v.value === params.group_by)?.label || "Add Property"}
              </span>
            </div>
          }
          options={groupByOptions}
          placeholder="Group By"
          allowNoValue
        />
      </div>
    </div>
  );
};
