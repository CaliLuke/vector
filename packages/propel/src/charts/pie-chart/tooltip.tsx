/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";
import type { TPieChartProps } from "@plane/types";
import { isRecord } from "@plane/utils/common";
import type { Payload } from "recharts/types/component/DefaultTooltipContent";
// plane imports
import { Card, ECardSpacing } from "../../card";

type Props = {
  dotColor?: string;
  label: string;
  payload: readonly Payload[];
};

const CustomPieChartTooltip = React.memo(function CustomPieChartTooltip(props: Props) {
  const { dotColor, label, payload } = props;

  return (
    <Card
      className="vertical-scrollbar flex scrollbar-sm max-h-[40vh] w-[12rem] flex-col overflow-y-scroll"
      spacing={ECardSpacing.SM}
    >
      <p className="flex-shrink-0 truncate border-b border-subtle pb-2 text-11 font-medium text-primary">{label}</p>
      {payload?.map((item) => (
        <div key={String(item?.dataKey)} className="flex items-center gap-2 text-11 capitalize">
          <div className="flex items-center gap-2 truncate">
            <div
              className="size-2 flex-shrink-0 rounded-xs"
              style={{
                backgroundColor: dotColor,
              }}
            />
            <span className="truncate text-tertiary">{item?.name}:</span>
          </div>
          <span className="flex-shrink-0 font-medium text-secondary">{item?.value}</span>
        </div>
      ))}
    </Card>
  );
});
CustomPieChartTooltip.displayName = "CustomPieChartTooltip";

export function PieChartTooltip({
  active,
  payload,
  cells,
  showTooltip,
  dataKey,
  tooltipLabel,
}: Pick<TPieChartProps<string, string>, "cells" | "showTooltip" | "dataKey" | "tooltipLabel"> & {
  active?: boolean;
  payload?: readonly Payload[];
}) {
  if (!showTooltip || !active || !payload?.length) return null;
  const entry: unknown = payload[0].payload;
  if (!isRecord(entry)) return null;
  const cell = cells.find((candidate) => candidate.key === entry.key);
  if (!cell) return null;
  const label =
    typeof tooltipLabel === "function"
      ? tooltipLabel(isRecord(entry.payload) ? entry.payload : {})
      : tooltipLabel || dataKey;
  return <CustomPieChartTooltip dotColor={cell.fill} label={label} payload={payload} />;
}
