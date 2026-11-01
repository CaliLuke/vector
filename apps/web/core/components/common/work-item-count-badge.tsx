/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Tooltip } from "@plane/propel/tooltip";

type Props = { count: number | undefined; parent: "cycle" | "module"; isMobile: boolean };

export function WorkItemCountBadge({ count, parent, isMobile }: Props) {
  return count && count > 0 ? (
    <Tooltip
      isMobile={isMobile}
      tooltipContent={`There are ${count} ${count > 1 ? "work items" : "work item"} in this ${parent}`}
      position="bottom"
    >
      <span className="flex flex-shrink-0 cursor-default items-center justify-center rounded-xl bg-accent-primary/20 px-2 text-center text-11 font-semibold text-accent-primary">
        {count}
      </span>
    </Tooltip>
  ) : null;
}
