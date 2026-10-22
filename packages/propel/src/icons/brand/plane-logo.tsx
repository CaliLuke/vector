/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import * as React from "react";

import type { ISvgIcons } from "../type";

export function PlaneLogo({ width = "85", height = "52", className, color = "currentColor" }: ISvgIcons) {
  return (
    <svg
      width={width}
      height={height}
      viewBox="0 0 85 52"
      fill={color}
      xmlns="http://www.w3.org/2000/svg"
      className={className}
    >
      <path d="M9 2h13l20 36L62 2h14L48 51H36z" fill={color} />
    </svg>
  );
}
