/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useQuery } from "@tanstack/react-query";
import { useParams } from "react-router";

// plane ui
import { CycleIcon } from "@plane/propel/icons";
import { Tooltip } from "@plane/propel/tooltip";
// plane utils
import { cn } from "@plane/utils/common";
//hooks
import { cyclesOptions } from "@/queries/lookups";

type Props = {
  cycleId: string | undefined;
  shouldShowBorder?: boolean;
};

export const IssueBlockCycle = function IssueBlockCycle({ cycleId, shouldShowBorder = true }: Props) {
  const { data: cycles = [] } = useQuery(cyclesOptions(useParams().anchor));

  const cycle = cycles.find((item) => item.id === cycleId);

  return (
    <Tooltip tooltipHeading="Cycle" tooltipContent={cycle?.name ?? "No Cycle"}>
      <div
        className={cn(
          "flex h-full w-full items-center justify-between gap-1 rounded-sm px-2.5 py-1 text-11 duration-300 focus:outline-none",
          { "border-[0.5px] border-strong": shouldShowBorder }
        )}
      >
        <div className="flex w-full items-center gap-1.5 text-11">
          <CycleIcon className="h-3 w-3 flex-shrink-0" />
          <div className="max-w-40 truncate">{cycle?.name ?? "No Cycle"}</div>
        </div>
      </div>
    </Tooltip>
  );
};
