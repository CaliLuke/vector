/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import * as React from "react";
import { useState } from "react";
import type { ICustomSearchSelectOption } from "@plane/types";
import { CustomSearchSelect } from "../dropdowns";
import { cn } from "../utils";
import { Breadcrumbs } from "./breadcrumbs";

type TBreadcrumbNavigationSearchDropdownProps = {
  icon?: React.ReactNode;
  title?: string;
  selectedItem: string;
  navigationItems: ICustomSearchSelectOption[];
  onChange?: (value: string) => void;
  navigationDisabled?: boolean;
  isLast?: boolean;
  handleOnClick?: () => void;
  disableRootHover?: boolean;
  shouldTruncate?: boolean;
};

export function BreadcrumbNavigationSearchDropdown(props: TBreadcrumbNavigationSearchDropdownProps) {
  const {
    icon,
    title,
    selectedItem,
    navigationItems,
    onChange,
    navigationDisabled = false,
    isLast = false,
    handleOnClick,
    shouldTruncate = false,
  } = props;
  const [isDropdownOpen, setIsDropdownOpen] = useState(false);

  const label = (
    <Breadcrumbs.ItemWrapper label={title} isLast={isLast}>
      <span className="flex min-w-0 items-center gap-2">
        {shouldTruncate && <span className="flex text-tertiary @4xl:hidden">...</span>}
        <span
          className={cn("flex min-w-0 items-center gap-2", {
            "hidden @4xl:flex": shouldTruncate,
          })}
        >
          {icon && <Breadcrumbs.Icon>{icon}</Breadcrumbs.Icon>}
          <Breadcrumbs.Label>{title}</Breadcrumbs.Label>
        </span>
      </span>
    </Breadcrumbs.ItemWrapper>
  );

  const separator = (
    <Breadcrumbs.Separator
      className={cn("rounded-r-sm", {
        "bg-layer-1": isDropdownOpen && !isLast,
        "hover:bg-layer-1": !isLast,
      })}
      containerClassName="p-0"
      iconClassName={cn("group-hover:rotate-90 hover:text-primary", {
        "text-primary": isDropdownOpen,
        "rotate-90": isDropdownOpen || isLast,
      })}
      showDivider={!isLast}
    />
  );

  const navigationLabel = (
    <button
      type="button"
      onClick={(event) => {
        event.preventDefault();
        event.stopPropagation();
        handleOnClick?.();
      }}
      className={cn("group flex h-full cursor-pointer items-center gap-2 rounded-sm rounded-r-none text-tertiary", {
        "hover:bg-layer-1 hover:text-primary": !isLast,
      })}
    >
      {label}
    </button>
  );

  const trigger = isLast ? (
    <button
      type="button"
      onClick={() => setIsDropdownOpen((open) => !open)}
      className="group flex h-full items-center gap-0.5 rounded-sm outline-none"
    >
      {label}
      {separator}
    </button>
  ) : (
    <button
      type="button"
      aria-label="Open breadcrumb menu"
      onClick={() => setIsDropdownOpen((open) => !open)}
      className="group flex h-full items-center rounded-sm outline-none"
    >
      {separator}
    </button>
  );

  return (
    <div className="group flex h-full items-center gap-0.5 rounded-sm hover:bg-surface-2">
      {!isLast && navigationLabel}
      <CustomSearchSelect
        customButton={trigger}
        customButtonAsChild
        customButtonClassName={cn("group flex h-full cursor-pointer items-center outline-none", {
          "bg-surface-2": isDropdownOpen,
        })}
        onClose={() => setIsDropdownOpen(false)}
        options={navigationItems}
        value={selectedItem}
        onChange={(value: string) => {
          if (value !== selectedItem) onChange?.(value);
        }}
        disabled={navigationDisabled}
        placement="bottom-start"
        className="h-full rounded-sm"
      />
    </div>
  );
}
