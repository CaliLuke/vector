/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { useMemo, useState } from "react";
import type { PieLabelRenderProps } from "recharts";
import { isRecord } from "@plane/utils/common";
import { Cell, PieChart as CorePieChart, Label, Pie, ResponsiveContainer, Tooltip } from "recharts";
// plane imports
import type { TPieChartProps } from "@plane/types";
import { chartMargin } from "../components/layout";
// local components
import { KeyedChartLegend } from "../components/legend";
import { CustomActiveShape } from "./active-shape";
import { PieChartTooltip } from "./tooltip";

const renderActiveShape = ({ key, ...shapeProps }: React.ComponentProps<typeof CustomActiveShape>) => (
  <CustomActiveShape key={key} {...shapeProps} />
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
        <CorePieChart data={data} margin={chartMargin(margin)}>
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
            label={showLabel ? <PieValueLabel activeLegend={activeLegend} customLabel={customLabel} /> : undefined}
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
          {legend && <KeyedChartLegend options={legend} onActiveChange={setActiveLegend} />}
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
            content={
              <PieChartTooltip cells={cells} showTooltip={showTooltip} dataKey={dataKey} tooltipLabel={tooltipLabel} />
            }
          />
        </CorePieChart>
      </ResponsiveContainer>
    </div>
  );
});
PieChart.displayName = "PieChart";

function PieValueLabel({
  payload,
  activeLegend,
  customLabel,
  ...position
}: Omit<Partial<PieLabelRenderProps>, "payload"> & {
  payload?: unknown;
  activeLegend: string | null;
  customLabel?: (value: string | number) => string;
}) {
  if (!isRecord(payload) || (typeof payload.count !== "number" && typeof payload.count !== "string")) return null;
  return (
    <text
      className="text-sm font-medium transition-opacity duration-200"
      cx={position.cx}
      cy={position.cy}
      x={position.x}
      y={position.y}
      textAnchor={position.textAnchor}
      dominantBaseline={position.dominantBaseline}
      fill="var(--text-color-secondary)"
      opacity={activeLegend && activeLegend !== payload.key ? 0.1 : 1}
    >
      {customLabel?.(payload.count) ?? payload.count}
    </text>
  );
}
