/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";
import { Tooltip } from "@plane/propel/tooltip";
import { cn } from "../utils";

type Props = {
  data: { id: string | number; name: string; value: number; color: string }[];
  noTooltip?: boolean;
  inPercentage?: boolean;
  size?: "sm" | "md" | "lg" | "xl";
  className?: string;
  barClassName?: string;
};

export function LinearProgressIndicator({
  data,
  noTooltip = false,
  inPercentage = false,
  size = "sm",
  className = "",
  barClassName = "",
}: Props) {
  const total = data.reduce((acc, cur) => acc + cur.value, 0);

  const bars = data.map((item) => {
    if (total <= 0 || item.value <= 0) return null;
    const width = `${(item.value / total) * 100}%`;
    const style = {
      width,
      backgroundColor: item.color,
    };
    if (noTooltip) return <div style={style} key={item.id} />;
    else
      return (
        <Tooltip key={item.id} tooltipContent={`${item.name} ${Math.round(item.value)}${inPercentage ? "%" : ""}`}>
          <div style={style} className={cn("first:rounded-l-xs last:rounded-r-xs", barClassName)} />
        </Tooltip>
      );
  });

  return (
    <div
      className={cn("flex w-full items-center justify-between gap-[1px] rounded-xs", {
        "h-2": size === "sm",
        "h-3": size === "md",
        "h-3.5": size === "lg",
        "h-[14px]": size === "xl",
      })}
    >
      <div className={cn("flex h-full w-full gap-[1.5px] rounded-xs bg-surface-2 p-[2px]", className)}>{bars}</div>
    </div>
  );
}
