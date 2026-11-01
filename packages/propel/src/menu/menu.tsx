/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import * as React from "react";
import { Menu as BaseMenu } from "@base-ui/react/menu";
import { MoreHorizontal } from "lucide-react";
import { ChevronDownIcon, ChevronRightIcon } from "../icons";
import { cn } from "@plane/utils/common";
import type { TMenuProps, TSubMenuProps, TMenuItemProps } from "./types";

// SubMenu implementation
function SubMenu(props: TSubMenuProps) {
  const { children, trigger, disabled = false, className = "" } = props;

  return (
    <BaseMenu.SubmenuRoot disabled={disabled}>
      <BaseMenu.SubmenuTrigger className={""}>
        <span className="flex-1">{trigger}</span>
        <ChevronRightIcon />
      </BaseMenu.SubmenuTrigger>
      <BaseMenu.Portal>
        <BaseMenu.Positioner className={""} alignOffset={-4} sideOffset={-4}>
          <BaseMenu.Popup className={className}>{children} </BaseMenu.Popup>
        </BaseMenu.Positioner>
      </BaseMenu.Portal>
    </BaseMenu.SubmenuRoot>
  );
}

function MenuItem(props: TMenuItemProps) {
  const { children, disabled = false, onClick, className } = props;

  return (
    <BaseMenu.Item
      disabled={disabled}
      className={cn(
        "w-full cursor-pointer truncate rounded-sm px-1 py-1.5 text-left text-secondary outline-none select-none hover:bg-layer-1",
        {
          "text-placeholder": disabled,
        },
        className
      )}
      onClick={(e) => {
        onClick?.(e);
      }}
    >
      {children}
    </BaseMenu.Item>
  );
}

function Menu(props: TMenuProps) {
  const [isOpen, setIsOpen] = React.useState(false);
  return (
    <BaseMenu.Root
      open={isOpen}
      onOpenChange={(open) => {
        setIsOpen(open);
        if (!open && isOpen) props.onMenuClose?.();
        props.handleOpenChange?.(open);
      }}
    >
      <MenuTrigger appearance={props} isOpen={isOpen} />
      <BaseMenu.Portal>
        <BaseMenu.Positioner align="start" className={cn("fixed z-30 translate-y-0", props.menuItemsClassName)}>
          <MenuPopup appearance={props}>{props.children}</MenuPopup>
        </BaseMenu.Positioner>
      </BaseMenu.Portal>
    </BaseMenu.Root>
  );
}

function MenuTrigger({ appearance, isOpen }: { appearance: TMenuProps; isOpen: boolean }) {
  return (
    <BaseMenu.Trigger
      type="button"
      onClick={(event) => {
        event.stopPropagation();
        appearance.menuButtonOnClick?.();
      }}
      className={menuTriggerClassName(appearance, isOpen)}
      tabIndex={appearance.customButtonTabIndex ?? 0}
      disabled={appearance.disabled}
      aria-label={appearance.ariaLabel}
      openOnHover={appearance.openOnHover}
    >
      <MenuTriggerContent appearance={appearance} />
    </BaseMenu.Trigger>
  );
}

function MenuTriggerContent({ appearance }: { appearance: TMenuProps }) {
  if (appearance.customButton) return <>{appearance.customButton}</>;
  if (appearance.ellipsis || appearance.verticalEllipsis)
    return <MoreHorizontal className={cn("h-3.5 w-3.5", { "rotate-90": appearance.verticalEllipsis })} />;
  return (
    <>
      {appearance.label}
      {!appearance.noChevron && <ChevronDownIcon className="h-3.5 w-3.5" />}
    </>
  );
}

function menuTriggerClassName(appearance: TMenuProps, isOpen: boolean) {
  const { disabled, buttonClassName, noBorder } = appearance;
  if (appearance.customButton) return cn(appearance.customButtonClassName, "outline-none");
  if (appearance.ellipsis || appearance.verticalEllipsis)
    return cn(
      "relative grid place-items-center rounded-sm p-1 text-secondary outline-none hover:text-primary",
      disabled ? "cursor-not-allowed" : "cursor-pointer hover:bg-layer-1",
      buttonClassName
    );
  return cn(
    "flex items-center justify-between gap-1 rounded-md px-2.5 py-1 text-11 whitespace-nowrap duration-300 outline-none",
    isOpen ? "bg-surface-2 text-primary" : "text-secondary",
    !noBorder && "shadow-sm border border-strong focus:outline-none",
    disabled ? "cursor-not-allowed text-secondary" : "cursor-pointer hover:bg-layer-1",
    buttonClassName
  );
}

function MenuPopup({ appearance, children }: { appearance: TMenuProps; children: React.ReactNode }) {
  const { maxHeight = "md", tabIndex, optionsClassName } = appearance;
  return (
    <BaseMenu.Popup
      tabIndex={tabIndex}
      data-main-menu="true"
      className={cn(
        "my-1 min-w-[12rem] overflow-y-scroll rounded-md border-[0.5px] border-strong bg-surface-1 px-2 py-2.5 text-11 whitespace-nowrap shadow-raised-200 focus:outline-none",
        {
          "max-h-60": maxHeight === "lg",
          "max-h-48": maxHeight === "md",
          "max-h-36": maxHeight === "rg",
          "max-h-28": maxHeight === "sm",
        },
        optionsClassName
      )}
    >
      {children}
    </BaseMenu.Popup>
  );
}

Menu.MenuItem = MenuItem;
Menu.SubMenu = SubMenu;

export { Menu };
