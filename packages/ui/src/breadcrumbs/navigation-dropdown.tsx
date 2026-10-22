/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { CheckIcon } from "lucide-react";
import * as React from "react";
// ui
import { Tooltip } from "@plane/propel/tooltip";
import type { TContextMenuItem } from "../dropdowns";
import { CustomMenu } from "../dropdowns";
import { cn } from "../utils";
import { Breadcrumbs } from "./breadcrumbs";

type TBreadcrumbNavigationDropdownProps = {
  selectedItemKey: string;
  navigationItems: TContextMenuItem[];
  navigationDisabled?: boolean;
  handleOnClick?: () => void;
  isLast?: boolean;
};

export function BreadcrumbNavigationDropdown(props: TBreadcrumbNavigationDropdownProps) {
  const { selectedItemKey, navigationItems, navigationDisabled = false, handleOnClick, isLast = false } = props;
  const [isOpen, setIsOpen] = React.useState(false);
  // derived values
  const selectedItem = navigationItems.find((item) => item.key === selectedItemKey);
  const selectedItemIcon = selectedItem?.icon ? (
    <selectedItem.icon className={cn("size-4", selectedItem.iconClassName)} />
  ) : undefined;
  const label = (
    <Tooltip tooltipContent={selectedItem?.title} position="bottom" disabled={isOpen}>
      <span className="flex items-center gap-2">
        <div className="flex text-tertiary @4xl:hidden">...</div>
        <div className="hidden items-center gap-2 @4xl:flex">
          {selectedItemIcon && <Breadcrumbs.Icon>{selectedItemIcon}</Breadcrumbs.Icon>}
          <Breadcrumbs.Label>{selectedItem?.title}</Breadcrumbs.Label>
        </div>
      </span>
    </Tooltip>
  );
  const separator = (
    <Breadcrumbs.Separator
      className={cn("rounded-r-sm", {
        "bg-layer-1": isOpen && !isLast,
        "hover:bg-layer-1": !isLast,
      })}
      containerClassName="p-0"
      iconClassName={cn("group-hover:rotate-90 hover:text-primary", {
        "text-primary": isOpen,
        "rotate-90": isOpen || isLast,
      })}
      showDivider={!isLast}
    />
  );

  // if no selected item, return null
  if (!selectedItem) return null;

  function NavigationButton() {
    return (
      <button
        type="button"
        onClick={(e) => {
          e.preventDefault();
          e.stopPropagation();
          if (!isLast) handleOnClick?.();
        }}
        className="group flex h-full cursor-pointer items-center gap-2 rounded-sm rounded-r-none px-1.5 py-1 text-13 font-medium text-tertiary hover:bg-layer-1 hover:text-primary"
      >
        {label}
      </button>
    );
  }

  if (navigationDisabled) {
    return <NavigationButton />;
  }

  const trigger = isLast ? (
    <button type="button" className="group flex h-full items-center gap-0.5 rounded-sm outline-none">
      {label}
      {separator}
    </button>
  ) : (
    <button type="button" aria-label="Open breadcrumb menu" className="group flex h-full items-center outline-none">
      {separator}
    </button>
  );

  return (
    <div className="flex h-full items-center gap-0.5 rounded-sm hover:bg-surface-2">
      {!isLast && <NavigationButton />}
      <CustomMenu
        customButton={trigger}
        customButtonAsChild
        placement="bottom-start"
        className="h-full rounded-sm"
        customButtonClassName={cn("group flex h-full cursor-pointer items-center rounded-sm outline-none", {
          "bg-surface-2": isOpen,
        })}
        closeOnSelect
        menuButtonOnClick={() => {
          setIsOpen(!isOpen);
        }}
        onMenuClose={() => {
          setIsOpen(false);
        }}
      >
        {navigationItems.map((item) => {
          if (item.shouldRender === false) return null;
          return (
            <CustomMenu.MenuItem
              key={item.key}
              onClick={(e) => {
                e.preventDefault();
                e.stopPropagation();
                if (item.key === selectedItemKey) return;
                item.action();
              }}
              className={cn(
                "flex items-center gap-2",
                {
                  "text-placeholder": item.disabled,
                },
                item.className
              )}
              disabled={item.disabled}
            >
              {item.icon && <item.icon className={cn("size-4 flex-shrink-0", item.iconClassName)} />}
              <div className="w-full">
                <h5>{item.title}</h5>
                {item.description && (
                  <p
                    className={cn("whitespace-pre-line text-tertiary", {
                      "text-placeholder": item.disabled,
                    })}
                  >
                    {item.description}
                  </p>
                )}
              </div>
              {item.key === selectedItemKey && <CheckIcon className="size-3.5 flex-shrink-0" />}
            </CustomMenu.MenuItem>
          );
        })}
      </CustomMenu>
    </div>
  );
}
