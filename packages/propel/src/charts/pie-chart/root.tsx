/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { useMemo, useState } from "react";
import { Cell, PieChart as CorePieChart, Label, Legend, Pie, ResponsiveContainer, Tooltip } from "recharts";
// plane imports
import type { TPieChartProps } from "@plane/types";
// local components
import { getLegendProps } from "../components/legend";
import { CustomActiveShape } from "./active-shape";
import { CustomPieChartTooltip } from "./tooltip";

const renderActiveShape = (shapeProps: React.ComponentProps<typeof CustomActiveShape>) => (
  <CustomActiveShape {...shapeProps} />
);

export const PieChart = React.memo(function PieChart<K extends string, T extends string>(props: TPieChartProps<K, T>) {
  const {
    data,
    dataKey,
    cells,
    className,
    innerRadius,
    legend,
    margin,
    outerRadius,
    showTooltip = true,
    showLabel,
    customLabel,
    centerLabel,
    cornerRadius,
    paddingAngle,
    tooltipLabel,
  } = props;
  // states
  const [activeLegend, setActiveLegend] = useState<string | null>(null);

  const renderCells = useMemo(
    () =>
      cells.map((cell) => (
        <Cell
          key={cell.key}
          className="transition-opacity duration-200"
          fill={cell.fill}
          opacity={!!activeLegend && activeLegend !== cell.key ? 0.1 : 1}
          style={{
            outline: "none",
          }}
        />
      )),
    [activeLegend, cells]
  );

  return (
    <div className={className}>
      <ResponsiveContainer width="100%" height="100%">
        <CorePieChart
          data={data}
          margin={{
            top: margin?.top === undefined ? 5 : margin.top,
            right: margin?.right === undefined ? 30 : margin.right,
            bottom: margin?.bottom === undefined ? 5 : margin.bottom,
            left: margin?.left === undefined ? 20 : margin.left,
          }}
        >
          <Pie
            data={data}
            dataKey={dataKey}
            cx="50%"
            cy="50%"
            stroke="none"
            activeShape={renderActiveShape}
            innerRadius={innerRadius}
            outerRadius={outerRadius}
            cornerRadius={cornerRadius}
            paddingAngle={paddingAngle}
            labelLine={false}
            label={
              showLabel
                ? ({ payload, ...props }) => (
                    <text
                      className="text-sm font-medium transition-opacity duration-200"
                      cx={props.cx}
                      cy={props.cy}
                      x={props.x}
                      y={props.y}
                      textAnchor={props.textAnchor}
                      dominantBaseline={props.dominantBaseline}
                      fill="var(--text-color-secondary)"
                      opacity={!!activeLegend && activeLegend !== payload.key ? 0.1 : 1}
                    >
                      {customLabel?.(payload.count) ?? payload.count}
                    </text>
                  )
                : undefined
            }
          >
            {renderCells}
            {centerLabel && (
              <Label
                value={centerLabel.text}
                fill={centerLabel.fill}
                position="center"
                opacity={activeLegend ? 0.1 : 1}
                style={centerLabel.style}
                className={centerLabel.className}
              />
            )}
          </Pie>
          {legend && (
            <Legend
              onMouseEnter={(payload) => {
                const key =
                  typeof payload.payload === "object" && payload.payload !== null && "key" in payload.payload
                    ? String(payload.payload.key)
                    : undefined;
                if (!key) return;
                setActiveLegend(key);
              }}
              onMouseLeave={() => setActiveLegend(null)}
              {...getLegendProps(legend)}
            />
          )}
          <Tooltip
            cursor={
              showTooltip
                ? {
                    fill: "currentColor",
                    className: "bg-layer-1-hover cursor-pointer",
                  }
                : false
            }
            wrapperStyle={{
              pointerEvents: "none",
            }}
            content={({ active, payload }) => {
              if (!showTooltip || !active || !payload || !payload.length) return null;
              const cellData = cells.find((c) => c.key === payload[0].payload.key);
              if (!cellData) return null;
              const label = tooltipLabel
                ? typeof tooltipLabel === "function"
                  ? tooltipLabel(payload[0]?.payload?.payload)
                  : tooltipLabel
                : dataKey;
              return <CustomPieChartTooltip dotColor={cellData.fill} label={String(label)} payload={payload} />;
            }}
          />
        </CorePieChart>
      </ResponsiveContainer>
    </div>
  );
});
PieChart.displayName = "PieChart";
