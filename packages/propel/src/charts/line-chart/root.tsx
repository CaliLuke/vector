/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { useMemo, useState } from "react";
import { CartesianGrid, LineChart as CoreLineChart, Line, ResponsiveContainer } from "recharts";
// plane imports
import type { TLineChartProps } from "@plane/types";
import { chartMargin } from "../components/layout";
// local components
import { ChartLegend } from "../components/legend";
import { ChartXAxis, ChartYAxis } from "../components/cartesian-axes";
import { ChartTooltip } from "../components/tooltip";

export const LineChart = React.memo(function LineChart<K extends string, T extends string>(
  props: TLineChartProps<K, T>
) {
  const {
    data,
    lines,
    margin,
    xAxis,
    yAxis,
    className,
    tickCount = {
      x: undefined,
      y: 10,
    },
    customTicks,
    legend,
    showTooltip = true,
    customTooltipContent,
  } = props;
  // states
  const [activeLine, setActiveLine] = useState<string | null>(null);
  const [activeLegend, setActiveLegend] = useState<string | null>(null);

  // derived values
  const { itemKeys, itemLabels, itemDotColors } = useMemo(() => {
    const keys: string[] = [];
    const labels: Record<string, string> = {};
    const colors: Record<string, string> = {};

    for (const line of lines) {
      keys.push(line.key);
      labels[line.key] = line.label;
      colors[line.key] = line.stroke;
    }

    return { itemKeys: keys, itemLabels: labels, itemDotColors: colors };
  }, [lines]);

  const renderLines = useMemo(
    () =>
      lines.map((line) => (
        <Line
          key={line.key}
          dataKey={line.key}
          type={line.smoothCurves ? "monotone" : "linear"}
          className="[&_path]:transition-opacity [&_path]:duration-200"
          opacity={!!activeLegend && activeLegend !== line.key ? 0.1 : 1}
          fill={line.fill}
          stroke={line.stroke}
          strokeWidth={2}
          strokeDasharray={line.dashedLine ? "4 4" : "none"}
          dot={
            line.showDot
              ? {
                  fill: line.fill,
                  fillOpacity: 1,
                }
              : false
          }
          activeDot={{
            stroke: line.fill,
          }}
          onMouseEnter={() => setActiveLine(line.key)}
          onMouseLeave={() => setActiveLine(null)}
        />
      )),
    [activeLegend, lines]
  );

  return (
    <div className={className}>
      <ResponsiveContainer width="100%" height="100%">
        <CoreLineChart data={data} margin={chartMargin(margin)}>
          <CartesianGrid stroke="var(--border-color-subtle)" vertical={false} />
          <ChartXAxis axis={{ ...xAxis, dy: 28 }} tick={customTicks?.x} tickCount={tickCount.x} />
          <ChartYAxis axis={yAxis} tick={customTicks?.y} tickCount={tickCount.y} labelOffset={-24} />
          {legend && <ChartLegend options={legend} labels={itemLabels} onActiveChange={setActiveLegend} />}
          {showTooltip && (
            <ChartTooltip
              activeKey={activeLine}
              itemKeys={itemKeys}
              itemLabels={itemLabels}
              itemDotColors={itemDotColors}
              renderContent={customTooltipContent}
            />
          )}
          {renderLines}
        </CoreLineChart>
      </ResponsiveContainer>
    </div>
  );
});
LineChart.displayName = "LineChart";
