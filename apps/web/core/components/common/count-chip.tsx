/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

//
import { forwardRef, type ComponentPropsWithoutRef } from "react";
import { cn } from "@plane/utils";

type TCountChip = Omit<ComponentPropsWithoutRef<"div">, "children" | "className"> & {
  count: string | number;
  className?: string;
};

export const CountChip = forwardRef<HTMLDivElement, TCountChip>(function CountChip(props, ref) {
  const { count, className = "", ...divProps } = props;

  return (
    <div
      {...divProps}
      ref={ref}
      className={cn(
        "relative flex flex-shrink-0 items-center justify-center rounded-xl bg-accent-primary/20 px-2.5 py-0.5 text-caption-sm-semibold text-accent-primary",
        className
      )}
    >
      {count}
    </div>
  );
});
