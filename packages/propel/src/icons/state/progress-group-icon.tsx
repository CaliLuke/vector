/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ISvgIcons } from "../type";
import { DashedCircle } from "./dashed-circle";
import { ProgressCircle } from "./progress-circle";

function ProgressGroupIcon({
  width = "20",
  height = "20",
  className,
  color = "#F59E0B",
  percentage = 100,
  showCenter = false,
}: ISvgIcons & { showCenter?: boolean }) {
  const normalized = Number.isFinite(percentage) ? (percentage <= 1 ? percentage * 100 : percentage) : 100;
  const progress = Math.max(0, Math.min(100, normalized));
  const circumference = 2 * Math.PI * 6;
  return (
    <svg width={width} height={height} viewBox="0 0 16 16" className={className}>
      <DashedCircle center={8} radius={6} color={color} percentage={progress} />
      {showCenter && (
        <circle
          cx={6}
          cy={6}
          r={3}
          strokeWidth={1.5}
          strokeLinecap="round"
          fill="none"
          transform="rotate(-90 8 6)"
          strokeDasharray={circumference}
          strokeDashoffset={0}
          stroke={color}
        />
      )}
      <ProgressCircle
        center={8}
        radius={6}
        color={color}
        strokeWidth={1.5}
        circumference={circumference}
        dashOffset={circumference * (1 - progress / 100)}
      />
    </svg>
  );
}

export function StartedGroupIcon(props: ISvgIcons) {
  return <ProgressGroupIcon {...props} showCenter />;
}

export function UnstartedGroupIcon(props: ISvgIcons) {
  return <ProgressGroupIcon {...props} />;
}
