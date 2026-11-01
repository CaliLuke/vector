/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";

type TickProps = {
  x?: number;
  y?: number;
  payload?: { value: string | number };
  getLabel?: (value: string | number) => React.ReactNode;
};

// Common classnames
const AXIS_TICK_CLASSNAME = "fill-tertiary text-13";

export const CustomXAxisTick = React.memo(function CustomXAxisTick({ x = 0, y = 0, payload, getLabel }: TickProps) {
  return (
    <g transform={`translate(${x},${y})`}>
      <text y={0} dy={16} textAnchor="middle" className={AXIS_TICK_CLASSNAME}>
        {getLabel && payload ? getLabel(payload.value) : payload?.value}
      </text>
    </g>
  );
});
CustomXAxisTick.displayName = "CustomXAxisTick";

export const CustomYAxisTick = React.memo(function CustomYAxisTick({ x = 0, y = 0, payload }: TickProps) {
  return (
    <g transform={`translate(${x},${y})`}>
      <text dx={-10} textAnchor="middle" className={AXIS_TICK_CLASSNAME}>
        {payload?.value}
      </text>
    </g>
  );
});

CustomYAxisTick.displayName = "CustomYAxisTick";

export const CustomRadarAxisTick = React.memo(function CustomRadarAxisTick({
  x = 0,
  y = 0,
  payload,
  getLabel,
  cx = 0,
  cy = 0,
  offset = 16,
}: TickProps & { cx?: number; cy?: number; offset?: number }) {
  // Calculate direction vector from center to tick
  const dx = x - cx;
  const dy = y - cy;
  // Normalize and apply offset
  const length = Math.sqrt(dx * dx + dy * dy);
  const normX = length === 0 ? 0 : dx / length;
  const normY = length === 0 ? 0 : dy / length;
  const labelX = x + normX * offset;
  const labelY = y + normY * offset;

  return (
    <g transform={`translate(${labelX},${labelY})`}>
      <text y={0} textAnchor="middle" className={AXIS_TICK_CLASSNAME}>
        {getLabel && payload ? getLabel(payload.value) : payload?.value}
      </text>
    </g>
  );
});
CustomRadarAxisTick.displayName = "CustomRadarAxisTick";
