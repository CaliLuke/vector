/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { useMemo, useState } from "react";
import { Area, ResponsiveContainer, Line, ComposedChart, CartesianGrid } from "recharts";
// plane imports
import type { TAreaChartProps } from "@plane/types";
import { chartMargin } from "../components/layout";
// local components
import { ChartLegend } from "../components/legend";
import { ChartXAxis, ChartYAxis } from "../components/cartesian-axes";
import { ChartTooltip } from "../components/tooltip";

export const AreaChart = React.memo(function AreaChart<K extends string, T extends string>(
  props: TAreaChartProps<K, T>
) {
  const {
    data,
    areas,
    xAxis,
    yAxis,
    className,
    legend,
    margin,
    tickCount = {
      x: undefined,
      y: 10,
    },
    customTicks,
    showTooltip = true,
    comparisonLine,
  } = props;
  // states
  const [activeArea, setActiveArea] = useState<string | null>(null);
  const [activeLegend, setActiveLegend] = useState<string | null>(null);

  // derived values
  const { itemKeys, itemLabels, itemDotColors } = useMemo(() => {
    const keys: string[] = [];
    const labels: Record<string, string> = {};
    const colors: Record<string, string> = {};

    for (const area of areas) {
      keys.push(area.key);
      labels[area.key] = area.label;
      colors[area.key] = area.fill;
    }

    return { itemKeys: keys, itemLabels: labels, itemDotColors: colors };
  }, [areas]);

  const renderAreas = useMemo(
    () =>
      areas.map((area) => (
        <Area
          key={area.key}
          type={area.smoothCurves ? "monotone" : "linear"}
          dataKey={area.key}
          stackId={area.stackId}
          fill={area.fill}
          opacity={!!activeLegend && activeLegend !== area.key ? 0.1 : 1}
          fillOpacity={area.fillOpacity}
          strokeOpacity={area.strokeOpacity}
          stroke={area.strokeColor}
          strokeWidth={2}
          style={area.style}
          dot={
            area.showDot
              ? {
                  fill: area.fill,
                  fillOpacity: 1,
                }
              : false
          }
          activeDot={{
            stroke: area.fill,
          }}
          onMouseEnter={() => setActiveArea(area.key)}
          onMouseLeave={() => setActiveArea(null)}
          className="[&_path]:transition-opacity [&_path]:duration-200"
        />
      )),
    [activeLegend, areas]
  );

  // create comparison line data for straight line from origin to last point
  const comparisonLineData = useMemo(() => {
    if (!data || data.length === 0) return [];
    // get the last data point
    const lastPoint = data[data.length - 1];
    // for the y-value in the last point, use its yAxis key value
    const lastY = lastPoint[yAxis.key];
    const lastYValue = typeof lastY === "number" ? lastY : 0;
    // create data for a straight line that has points at each x-axis position
    return data.map((item, index) => {
      // calculate the y value for this point on the straight line
      // using linear interpolation between (0,0) and (last_x, last_y)
      const ratio = data.length > 1 ? index / (data.length - 1) : 0;
      const interpolatedValue = ratio * lastYValue;

      return {
        [xAxis.key]: item[xAxis.key],
        comparisonLine: interpolatedValue,
      };
    });
  }, [data, xAxis.key, yAxis.key]);
  return (
    <div className={className}>
      <ResponsiveContainer width="100%" height="100%">
        <ComposedChart data={data} margin={chartMargin(margin)}>
          <CartesianGrid stroke="var(--border-color-subtle)" vertical={false} />
          <ChartXAxis axis={{ ...xAxis, dy: 28 }} tick={customTicks?.x} tickCount={tickCount.x} />
          <ChartYAxis axis={yAxis} tick={customTicks?.y} tickCount={tickCount.y} />
          {legend && <ChartLegend options={legend} labels={itemLabels} onActiveChange={setActiveLegend} />}
          {showTooltip && (
            <ChartTooltip
              activeKey={activeArea}
              itemKeys={itemKeys}
              itemLabels={itemLabels}
              itemDotColors={itemDotColors}
            />
          )}
          {renderAreas}
          {comparisonLine && (
            <Line
              data={comparisonLineData}
              type="linear"
              dataKey="comparisonLine"
              stroke={comparisonLine.strokeColor}
              fill={comparisonLine.strokeColor}
              strokeWidth={2}
              strokeDasharray={comparisonLine.dashedLine ? "4 4" : "none"}
              activeDot={false}
              legendType="none"
              name="Comparison line"
            />
          )}
        </ComposedChart>
      </ResponsiveContainer>
    </div>
  );
});
AreaChart.displayName = "AreaChart";
