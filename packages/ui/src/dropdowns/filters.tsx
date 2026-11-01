/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { useState } from "react";
import type { Placement } from "@popperjs/core";
import { usePlatformOS } from "@plane/hooks";
import { Button } from "@plane/propel/button";
import { Popover } from "@plane/propel/popover";
import { cn } from "@plane/utils/common";

type Props = {
  children: React.ReactNode;
  icon?: React.ComponentProps<typeof Button>["prependIcon"];
  miniIcon?: React.ReactNode;
  title?: string;
  placement?: Placement;
  disabled?: boolean;
  tabIndex?: number;
  menuButton?: React.ReactNode;
  isFiltersApplied?: boolean;
  responsive?: boolean;
};

/** The popup owns initial and return focus; filter contents keep normal tab order. */
export function FiltersDropdown({
  children,
  icon,
  miniIcon,
  title = "Dropdown",
  placement = "auto",
  disabled = false,
  tabIndex,
  menuButton,
  isFiltersApplied = false,
  responsive = false,
}: Props) {
  const [open, setOpen] = useState(false);
  const { isMobile } = usePlatformOS();
  const trigger = menuButton ? (
    <button type="button" tabIndex={tabIndex} disabled={disabled}>
      {menuButton}
    </button>
  ) : (
    <Button
      variant="secondary"
      size={responsive ? "lg" : undefined}
      disabled={disabled}
      tabIndex={tabIndex}
      className={cn("relative", open ? "text-primary" : "text-secondary")}
    >
      <span className={cn("items-center gap-2", responsive ? "hidden @4xl:flex" : "flex")}>
        {icon && React.cloneElement(icon, { className: "size-4 shrink-0", strokeWidth: 2 })}
        <span>{title}</span>
      </span>
      {responsive && <span className="flex @4xl:hidden">{miniIcon || title}</span>}
      {isFiltersApplied && <span className="absolute -top-0.5 -right-0.5 size-2 rounded-full bg-accent-primary" />}
    </Button>
  );
  return (
    <Popover open={open} onOpenChange={setOpen}>
      <Popover.Button render={trigger} disabled={disabled} />
      <Popover.Panel
        placement={placement}
        sideOffset={4}
        initialFocus={!isMobile}
        positionerClassName="z-10"
        className="overflow-hidden rounded-sm border border-subtle bg-surface-1 shadow-raised-200"
      >
        <div className="flex max-h-[30rem] w-[18.75rem] flex-col overflow-hidden lg:max-h-[37.5rem]">{children}</div>
      </Popover.Panel>
    </Popover>
  );
}
