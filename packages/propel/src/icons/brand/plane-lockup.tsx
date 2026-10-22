/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import * as React from "react";
import type { ISvgIcons } from "../type";

export function PlaneLockup({ width = "253", height = "53", className, color = "currentColor" }: ISvgIcons) {
  return (
    <svg
      width={width}
      height={height}
      viewBox="0 0 160 44"
      fill={color}
      xmlns="http://www.w3.org/2000/svg"
      className={className}
      aria-label="Vector"
    >
      <text x="0" y="34" fontFamily="Inter, system-ui, sans-serif" fontSize="40" fontWeight="650" letterSpacing="-2">
        Vector
      </text>
    </svg>
  );
}
