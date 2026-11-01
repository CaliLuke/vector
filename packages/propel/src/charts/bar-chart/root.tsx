import { isRecord } from "@plane/utils/common";
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { useCallback, useMemo, useState } from "react";
import { BarChart as CoreBarChart, Bar, ResponsiveContainer, CartesianGrid } from "recharts";
// plane imports
import type { TBarChartProps, TBarItem } from "@plane/types";
import { chartMargin } from "../components/layout";
// local components
import { ChartLegend } from "../components/legend";
import { ChartXAxis, ChartYAxis } from "../components/cartesian-axes";
import { ChartTooltip, type ChartTooltipPayload } from "../components/tooltip";
import { barShapeVariants, DEFAULT_BAR_FILL_COLOR } from "./bar";

export const BarChart = React.memo(function BarChart<K extends string, T extends string>(props: TBarChartProps<K, T>) {
  const {
    data,
    bars,
    xAxis,
    yAxis,
    barSize = 40,
    className,
    legend,
    margin,
    tickCount = {
      x: undefined,
      y: 10,
    },
    customTicks,
    showTooltip = true,
    customTooltipContent,
  } = props;
  // states
  const [activeBar, setActiveBar] = useState<string | null>(null);
  const [activeLegend, setActiveLegend] = useState<string | null>(null);

  // derived values
  const { stackKeys, stackLabels } = useMemo(() => {
    const keys: string[] = [];
    const labels: Record<string, string> = {};

    for (const bar of bars) {
      keys.push(bar.key);
      labels[bar.key] = bar.label;
    }

    return { stackKeys: keys, stackLabels: labels };
  }, [bars]);

  // get bar color dynamically based on payload
  const getBarColor = useCallback(
    (payload: readonly unknown[], barKey: string) => {
      const bar = bars.find((b) => b.key === barKey);
      if (!bar) return DEFAULT_BAR_FILL_COLOR;

      if (typeof bar.fill === "function") {
        const payloadItem = payload.find(
          (item) => typeof item === "object" && item !== null && "dataKey" in item && item.dataKey === barKey
        );
        if (
          typeof payloadItem === "object" &&
          payloadItem !== null &&
          "payload" in payloadItem &&
          isRecord(payloadItem.payload)
        ) {
          try {
            return bar.fill(payloadItem.payload);
          } catch (error) {
            console.error(error);
            return DEFAULT_BAR_FILL_COLOR;
          }
        } else {
          return DEFAULT_BAR_FILL_COLOR; // fallback color when no payload data
        }
      } else {
        return bar.fill;
      }
    },
    [bars]
  );

  // get all bar colors
  const getAllBarColors = useCallback(
    (payload: ChartTooltipPayload) => {
      const colors: Record<string, string> = {};
      for (const bar of bars) {
        colors[bar.key] = getBarColor(payload, bar.key);
      }
      return colors;
    },
    [bars, getBarColor]
  );

  const renderBars = useMemo(
    () =>
      bars.map((bar) => (
        <Bar
          key={bar.key}
          dataKey={bar.key}
          stackId={bar.stackId}
          opacity={!!activeLegend && activeLegend !== bar.key ? 0.1 : 1}
          shape={<BarShape bar={bar} stackKeys={stackKeys} />}
          className="[&_path]:transition-opacity [&_path]:duration-200"
          onMouseEnter={() => setActiveBar(bar.key)}
          onMouseLeave={() => setActiveBar(null)}
          fill={getBarColor(data, bar.key)}
        />
      )),
    [activeLegend, stackKeys, bars, getBarColor, data]
  );

  return (
    <div className={className}>
      <ResponsiveContainer width="100%" height="100%">
        <CoreBarChart data={data} margin={chartMargin(margin)} barSize={barSize} className="recharts-wrapper">
          <CartesianGrid stroke="var(--border-color-subtle)" vertical={false} />
          <ChartXAxis axis={xAxis} tick={customTicks?.x} tickCount={tickCount.x} showEmptyLabel />
          <ChartYAxis axis={yAxis} tick={customTicks?.y} tickCount={tickCount.y} showEmptyLabel />
          {legend && <ChartLegend options={legend} labels={stackLabels} onActiveChange={setActiveLegend} />}
          {showTooltip && (
            <ChartTooltip
              cursor={{ fill: "var(--alpha-black-300)", className: "bg-layer-1 cursor-pointer" }}
              activeKey={activeBar}
              itemKeys={stackKeys}
              itemLabels={stackLabels}
              itemDotColors={getAllBarColors}
              renderContent={customTooltipContent}
            />
          )}
          {renderBars}
        </CoreBarChart>
      </ResponsiveContainer>
    </div>
  );
});
BarChart.displayName = "BarChart";

function BarShape({
  bar,
  stackKeys,
  ...shape
}: {
  bar: TBarItem<string>;
  stackKeys: string[];
  payload?: unknown;
  x?: unknown;
  y?: unknown;
  width?: unknown;
  height?: unknown;
  opacity?: unknown;
}) {
  if (!isRecord(shape) || !isRecord(shape.payload)) return null;
  const { x, y, width, height, opacity } = shape;
  if (typeof x !== "number" || typeof y !== "number" || typeof width !== "number" || typeof height !== "number")
    return null;
  return barShapeVariants[bar.shapeVariant ?? "bar"](
    {
      x,
      y,
      width,
      height,
      dataKey: bar.key,
      payload: shape.payload,
      opacity: typeof opacity === "number" ? opacity : undefined,
    },
    bar,
    stackKeys
  );
}
