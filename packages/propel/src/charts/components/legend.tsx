/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";
import { Legend } from "recharts";
import type { LegendProps } from "recharts";
import type { LegendPayload } from "recharts/types/component/DefaultLegendContent";
// plane imports
import type { TChartLegend } from "@plane/types";
import { cn } from "@plane/utils/common";

export function ChartLegend({
  options,
  labels,
  onActiveChange,
}: {
  options: TChartLegend;
  labels: Record<string, string>;
  onActiveChange: (key: string | null) => void;
}) {
  return (
    <Legend
      formatter={(value) => labels[value]}
      {...getLegendProps(options, (payload) => onActiveChange(payload?.value ?? null))}
    />
  );
}

export function KeyedChartLegend({
  options,
  onActiveChange,
}: {
  options: TChartLegend;
  onActiveChange: (key: string | null) => void;
}) {
  return (
    <Legend
      {...getLegendProps(options, (item) => {
        const value = item?.payload;
        onActiveChange(value && "key" in value ? String(value.key) : null);
      })}
    />
  );
}

const getLegendProps = (args: TChartLegend, onActiveChange?: (item: LegendPayload | null) => void): LegendProps => {
  const { align, layout, verticalAlign } = args;
  return {
    layout,
    align,
    verticalAlign,
    wrapperStyle: {
      display: "flex",
      overflow: "hidden",
      ...(layout === "vertical"
        ? {
            top: 0,
            alignItems: "center",
            height: "100%",
          }
        : {
            left: 0,
            bottom: 0,
            width: "100%",
            justifyContent: "center",
          }),
      ...args.wrapperStyles,
    },
    content: <CustomLegend {...args} onActiveChange={onActiveChange} />,
  };
};

const CustomLegend = React.forwardRef(function CustomLegend(
  props: Pick<LegendProps, "formatter" | "onClick" | "onMouseEnter" | "onMouseLeave"> &
    TChartLegend & { payload?: LegendPayload[]; onActiveChange?: (item: LegendPayload | null) => void },
  ref: React.ForwardedRef<HTMLDivElement>
) {
  const { formatter, layout, onClick, onMouseEnter, onMouseLeave, payload, onActiveChange } = props;

  if (!payload?.length) return null;

  return (
    <div
      ref={ref}
      className={cn("vertical-scrollbar flex scrollbar-sm items-center overflow-scroll px-4", {
        "max-h-full flex-col items-start py-4": layout === "vertical",
      })}
    >
      {payload.map((item, index) => {
        const className = cn(
          "flex items-center gap-1.5 text-13 font-medium whitespace-nowrap text-tertiary focus-visible:outline-2 focus-visible:outline-accent-strong",
          {
            "px-2": layout === "horizontal",
            "py-2": layout === "vertical",
            "pt-0 pl-0": index === 0,
            "pr-0 pb-0": index === payload.length - 1,
            "cursor-pointer": !!onClick,
          }
        );
        const label = (
          <>
            <span className="size-2 flex-shrink-0 rounded-xs" style={{ backgroundColor: item.color }} />
            {formatter?.(item.value, item, index) ??
              (item.payload && "name" in item.payload ? String(item.payload.name) : undefined)}
          </>
        );
        if (!onClick && !onActiveChange)
          return (
            <span key={item.value} className={className}>
              {label}
            </span>
          );
        return (
          <button
            key={item.value}
            type="button"
            className={className}
            onClick={(event) => onClick?.(item, index, event)}
            onFocus={() => onActiveChange?.(item)}
            onBlur={() => onActiveChange?.(null)}
            onMouseEnter={(event) => {
              onActiveChange?.(item);
              onMouseEnter?.(item, index, event);
            }}
            onMouseLeave={(event) => {
              onActiveChange?.(null);
              onMouseLeave?.(item, index, event);
            }}
          >
            {label}
          </button>
        );
      })}
    </div>
  );
});
CustomLegend.displayName = "CustomLegend";
