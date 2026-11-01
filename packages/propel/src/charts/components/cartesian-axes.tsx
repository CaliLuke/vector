/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { XAxis, YAxis } from "recharts";
import { AXIS_LABEL_CLASSNAME } from "@plane/constants";
import type { TAxisChartProps } from "@plane/types";
import { CustomXAxisTick, CustomYAxisTick } from "./tick";

type AxisOptions = TAxisChartProps<string, string>;
type Tick = NonNullable<AxisOptions["customTicks"]>["x"];

export function ChartXAxis({
  axis,
  tick,
  tickCount,
  showEmptyLabel = false,
}: {
  axis: AxisOptions["xAxis"];
  tick?: Tick;
  tickCount?: number;
  showEmptyLabel?: boolean;
}) {
  const TickComponent = tick ?? CustomXAxisTick;
  return (
    <XAxis
      dataKey={axis.key}
      tick={<TickComponent />}
      tickLine={false}
      axisLine={false}
      tickCount={tickCount}
      label={
        showEmptyLabel || axis.label
          ? {
              value: axis.label,
              dy: axis.dy ?? 28,
              className: AXIS_LABEL_CLASSNAME,
            }
          : undefined
      }
    />
  );
}

export function ChartYAxis({
  axis,
  tick,
  tickCount,
  labelOffset,
  showEmptyLabel = false,
}: {
  axis: AxisOptions["yAxis"];
  tick?: Tick;
  tickCount?: number;
  labelOffset?: number;
  showEmptyLabel?: boolean;
}) {
  const TickComponent = tick ?? CustomYAxisTick;
  return (
    <YAxis
      domain={axis.domain}
      tick={<TickComponent />}
      tickLine={false}
      axisLine={false}
      tickCount={tickCount}
      allowDecimals={!!axis.allowDecimals}
      label={
        showEmptyLabel || axis.label
          ? {
              value: axis.label,
              angle: -90,
              position: "bottom",
              offset: labelOffset ?? axis.offset ?? -24,
              dx: axis.dx ?? -16,
              className: AXIS_LABEL_CLASSNAME,
            }
          : undefined
      }
    />
  );
}
