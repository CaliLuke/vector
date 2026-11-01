/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";
import { Tooltip } from "recharts";
import type { TBaseChartProps } from "@plane/types";
import type { Payload } from "recharts/types/component/DefaultTooltipContent";
import { Card, ECardSpacing } from "../../card";
import { cn } from "@plane/utils/common";

export type ChartTooltipPayload = readonly Payload[];

type Props = {
  active?: boolean;
  activeKey?: string | null;
  label?: React.ReactNode;
  payload?: ChartTooltipPayload;
  itemKeys: string[];
  itemLabels: Record<string, string>;
  itemDotColors: Record<string, string> | ((payload: ChartTooltipPayload) => Record<string, string>);
  renderContent?: TBaseChartProps<string, string>["customTooltipContent"];
};

const DEFAULT_CURSOR = { stroke: "var(--text-color-tertiary)", strokeDasharray: "4 4" };

export function ChartTooltip({
  cursor = DEFAULT_CURSOR,
  ...content
}: Pick<Props, "activeKey" | "itemKeys" | "itemLabels" | "itemDotColors" | "renderContent"> & {
  cursor?: React.ComponentProps<typeof Tooltip>["cursor"];
}) {
  return <Tooltip cursor={cursor} wrapperStyle={{ pointerEvents: "auto" }} content={<CustomTooltip {...content} />} />;
}

export const CustomTooltip = React.memo(function CustomTooltip(props: Props) {
  const { active, activeKey, label, payload, itemKeys, itemLabels, itemDotColors, renderContent } = props;
  if (renderContent)
    return renderContent({
      active,
      label: typeof label === "string" || typeof label === "number" ? String(label) : "",
      payload,
    });
  const dotColors = typeof itemDotColors === "function" ? itemDotColors(payload ?? []) : itemDotColors;
  // derived values
  const filteredPayload = payload?.filter((item) => {
    const key = typeof item.dataKey === "string" || typeof item.dataKey === "number" ? String(item.dataKey) : null;
    return key !== null && itemKeys.includes(key);
  });

  if (!active || !filteredPayload || !filteredPayload.length) return null;

  return (
    <Card
      className="vertical-scrollbar flex scrollbar-sm max-h-[40vh] w-[12rem] flex-col overflow-y-scroll"
      spacing={ECardSpacing.SM}
    >
      <p className="flex-shrink-0 truncate border-b border-subtle pb-2 text-11 font-medium text-primary">{label}</p>
      {filteredPayload.map((item) => {
        const key = typeof item.dataKey === "string" || typeof item.dataKey === "number" ? String(item.dataKey) : null;
        if (key === null) return null;

        return (
          <div
            key={key}
            className={cn("flex items-center gap-2 text-11 transition-opacity", {
              "opacity-20": activeKey && key !== activeKey,
            })}
          >
            <div className="flex items-center gap-2 truncate">
              {dotColors[key] && (
                <div
                  className="size-2 flex-shrink-0 rounded-xs"
                  style={{
                    backgroundColor: dotColors[key],
                  }}
                />
              )}
              <span className="truncate text-tertiary">{itemLabels[key]}:</span>
            </div>
            <span className="flex-shrink-0 font-medium text-secondary">{item?.value}</span>
          </div>
        );
      })}
    </Card>
  );
});
CustomTooltip.displayName = "CustomTooltip";
