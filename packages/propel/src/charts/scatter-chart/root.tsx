/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { useMemo, useState } from "react";
import { CartesianGrid, ScatterChart as CoreScatterChart, Scatter, ResponsiveContainer } from "recharts";
// plane imports
import type { TScatterChartProps } from "@plane/types";
import { chartMargin } from "../components/layout";
// local components
import { ChartLegend } from "../components/legend";
import { ChartXAxis, ChartYAxis } from "../components/cartesian-axes";
import { ChartTooltip } from "../components/tooltip";

export const ScatterChart = React.memo(function ScatterChart<K extends string, T extends string>(
  props: TScatterChartProps<K, T>
) {
  const {
    data,
    scatterPoints,
    margin,
    xAxis,
    yAxis,
    className,
    customTicks,
    tickCount = {
      x: undefined,
      y: 10,
    },
    legend,
    showTooltip = true,
    customTooltipContent,
  } = props;
  // states
  const [activePoint, setActivePoint] = useState<string | null>(null);
  const [activeLegend, setActiveLegend] = useState<string | null>(null);

  //derived values
  const { itemKeys, itemLabels, itemDotColors } = useMemo(() => {
    const keys: string[] = [];
    const labels: Record<string, string> = {};
    const colors: Record<string, string> = {};

    for (const point of scatterPoints) {
      keys.push(point.key);
      labels[point.key] = point.label;
      colors[point.key] = point.fill;
    }
    return { itemKeys: keys, itemLabels: labels, itemDotColors: colors };
  }, [scatterPoints]);

  const renderPoints = useMemo(
    () =>
      scatterPoints.map((point) => (
        <Scatter
          key={point.key}
          dataKey={point.key}
          fill={point.fill}
          stroke={point.stroke}
          opacity={!!activeLegend && activeLegend !== point.key ? 0.1 : 1}
          onMouseEnter={() => setActivePoint(point.key)}
          onMouseLeave={() => setActivePoint(null)}
        />
      )),
    [activeLegend, scatterPoints]
  );

  return (
    <div className={className}>
      <ResponsiveContainer width="100%" height="100%">
        <CoreScatterChart data={data} margin={chartMargin(margin)}>
          <CartesianGrid stroke="var(--border-color-subtle)" vertical={false} />
          <ChartXAxis axis={{ ...xAxis, dy: 28 }} tick={customTicks?.x} tickCount={tickCount.x} />
          <ChartYAxis axis={yAxis} tick={customTicks?.y} tickCount={tickCount.y} labelOffset={-24} />
          {legend && <ChartLegend options={legend} labels={itemLabels} onActiveChange={setActiveLegend} />}
          {showTooltip && (
            <ChartTooltip
              activeKey={activePoint}
              itemKeys={itemKeys}
              itemLabels={itemLabels}
              itemDotColors={itemDotColors}
              renderContent={customTooltipContent}
            />
          )}
          {renderPoints}
        </CoreScatterChart>
      </ResponsiveContainer>
    </div>
  );
});
ScatterChart.displayName = "ScatterChart";
