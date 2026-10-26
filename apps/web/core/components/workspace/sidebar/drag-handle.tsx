/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { forwardRef } from "react";
import type { ComponentProps } from "react";
import { Tooltip } from "@plane/propel/tooltip";
import { DragHandle } from "@plane/ui";
import { cn } from "@plane/utils/common";

type TooltipProps = ComponentProps<typeof Tooltip>;

type SidebarDragHandleProps = Omit<TooltipProps, "children" | "className"> & {
  handleClassName?: string;
};

export const SidebarDragHandle = forwardRef<HTMLButtonElement, SidebarDragHandleProps>(function SidebarDragHandle(
  { handleClassName, ...tooltipProps },
  ref
) {
  return (
    <Tooltip {...tooltipProps}>
      <DragHandle className={cn("bg-transparent", handleClassName)} ref={ref} />
    </Tooltip>
  );
});
