/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";
import type { NameType, Payload, ValueType } from "recharts/types/component/DefaultTooltipContent";
import { Card, ECardSpacing } from "../../card";
import { cn } from "../../utils/classname";

type Props = {
  active: boolean | undefined;
  activeKey?: string | null;
  label: React.ReactNode;
  payload: readonly Payload<ValueType, NameType>[] | undefined;
  itemKeys: string[];
  itemLabels: Record<string, string>;
  itemDotColors: Record<string, string>;
};

export const CustomTooltip = React.memo(function CustomTooltip(props: Props) {
  const { active, activeKey, label, payload, itemKeys, itemLabels, itemDotColors } = props;
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
              {itemDotColors[key] && (
                <div
                  className="size-2 flex-shrink-0 rounded-xs"
                  style={{
                    backgroundColor: itemDotColors[key],
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
