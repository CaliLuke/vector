/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import * as React from "react";
import { Separator as SeparatorPrimitive } from "@base-ui/react/separator";
import { cn } from "@plane/utils/common";

const Separator = React.forwardRef(function Separator(
  { orientation = "horizontal", className, ...props }: React.ComponentProps<typeof SeparatorPrimitive>,
  ref: React.ForwardedRef<React.ElementRef<typeof SeparatorPrimitive>>
) {
  return (
    <SeparatorPrimitive
      ref={ref}
      orientation={orientation}
      data-slot="separator"
      data-orientation={orientation}
      {...props}
      className={cn("bg-subtle-1", "shrink-0", orientation === "horizontal" ? "h-px w-full" : "h-full w-px", className)}
    />
  );
});

Separator.displayName = "Separator";

export { Separator };
