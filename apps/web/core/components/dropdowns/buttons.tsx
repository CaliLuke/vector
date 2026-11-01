/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";
// helpers
import { getButtonStyling } from "@plane/propel/button";
import { Tooltip } from "@plane/propel/tooltip";
import { cn } from "@plane/utils/common";
// types
import { usePlatformOS } from "@/hooks/use-platform-os";
import { BACKGROUND_BUTTON_VARIANTS, BORDER_BUTTON_VARIANTS } from "./constants";
import type { TButtonVariants } from "./types";

export type DropdownButtonProps = {
  children: React.ReactNode;
  className?: string;
  isActive: boolean;
  tooltipContent?: string | React.ReactNode | null;
  tooltipHeading: string;
  showTooltip: boolean;
  variant: TButtonVariants;
  renderToolTipByDefault?: boolean;
};

// Each dropdown owns the native trigger; this component supplies its appearance
// and tooltip without adding a second focus target.
export function DropdownButton({
  children,
  className,
  isActive,
  tooltipContent,
  tooltipHeading,
  showTooltip,
  variant,
  renderToolTipByDefault = true,
}: DropdownButtonProps) {
  const { isMobile } = usePlatformOS();
  const hasBorder = BORDER_BUTTON_VARIANTS.includes(variant);
  const hasBackground = BACKGROUND_BUTTON_VARIANTS.includes(variant);

  return (
    <Tooltip
      tooltipHeading={tooltipHeading}
      tooltipContent={<>{tooltipContent}</>}
      disabled={!showTooltip}
      isMobile={isMobile}
      renderByDefault={renderToolTipByDefault}
    >
      <span
        className={cn(
          getButtonStyling("ghost", "sm"),
          "flex h-full w-full items-center gap-1.5",
          hasBorder ? "justify-start border-[0.5px] border-strong" : "justify-between",
          hasBackground && "bg-layer-3 hover:bg-layer-1-hover",
          !hasBackground && isActive && "bg-layer-transparent-active",
          className
        )}
      >
        {children}
      </span>
    </Tooltip>
  );
}

type TriggerAppearance = Pick<
  import("./types").TDropdownProps,
  "buttonVariant" | "buttonClassName" | "buttonContainerClassName" | "disabled" | "showTooltip"
> & { button?: React.ReactNode; renderByDefault?: boolean };
export const DropdownTrigger = React.forwardRef<
  HTMLButtonElement,
  {
    appearance: TriggerAppearance;
    isActive: boolean;
    children: React.ReactNode;
    tooltipHeading: string;
    tooltipContent?: React.ReactNode;
    tabIndex?: number;
    hoverBackground?: boolean;
  }
>(function DropdownTrigger(
  { appearance, children, hoverBackground, isActive, tooltipHeading, tooltipContent, ...props },
  ref
) {
  const {
    disabled,
    buttonClassName,
    buttonContainerClassName,
    buttonVariant,
    button: customContent,
    showTooltip = false,
    renderByDefault = true,
  } = appearance;
  return (
    <button
      {...props}
      ref={ref}
      type="button"
      disabled={disabled}
      className={cn(
        "clickable block h-full outline-none",
        customContent ? "w-full" : "max-w-full",
        !customContent && (disabled ? "cursor-not-allowed text-secondary" : "cursor-pointer"),
        hoverBackground && "hover:bg-layer-1",
        buttonContainerClassName
      )}
    >
      {customContent || (
        <DropdownButton
          className={buttonClassName}
          variant={buttonVariant}
          isActive={isActive}
          tooltipHeading={tooltipHeading}
          tooltipContent={tooltipContent}
          showTooltip={showTooltip}
          renderToolTipByDefault={renderByDefault}
        >
          {children}
        </DropdownButton>
      )}
    </button>
  );
});
