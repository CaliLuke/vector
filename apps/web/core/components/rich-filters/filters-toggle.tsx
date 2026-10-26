/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// plane imports
import { IconButton } from "@plane/propel/icon-button";
import { FilterIcon, FilterAppliedIcon } from "@plane/propel/icons";
import { cn } from "@plane/utils/common";
// hooks
import { useWorkItemFilterUi } from "@/providers/work-item-filter-ui";

const COMMON_CLASSNAME =
  "grid place-items-center h-7 w-full py-0.5 px-2 rounded-md border border-subtle-1 transition-all duration-200 cursor-pointer";

export function FiltersToggle() {
  const { state, update } = useWorkItemFilterUi();
  const hasAnyConditions = state.hasActiveFilters;
  const isFilterRowVisible = state.visible ?? false;
  const showFilterRowChangesPill = state.hasUnsavedChanges || hasAnyConditions;

  const activeFilterBaseClasses =
    "text-accent-primary border border-accent-subtle-1 hover:border-accent-subtle-1 active:border-accent-subtle-1 focus:border-accent-subtle-1";
  const noHoverStateClasses = "hover:text-accent-primary active:text-accent-primary focus:text-accent-primary";
  const backgroundClasses = isFilterRowVisible
    ? "bg-accent-subtle-hover hover:bg-accent-subtle-hover active:bg-accent-subtle-hover focus:bg-accent-subtle-hover"
    : "bg-accent-subtle hover:bg-accent-subtle active:bg-accent-subtle focus:bg-accent-subtle";
  const buttonClassName = cn({
    [activeFilterBaseClasses]: showFilterRowChangesPill,
    [backgroundClasses]: showFilterRowChangesPill,
    [noHoverStateClasses]: showFilterRowChangesPill,
  });
  const iconClassName = cn({ "text-accent-primary [&_path]:fill-current": showFilterRowChangesPill });

  return (
    <IconButton
      size="lg"
      variant="secondary"
      icon={showFilterRowChangesPill ? FilterAppliedIcon : FilterIcon}
      onClick={() => update({ visible: !isFilterRowVisible })}
      className={cn(COMMON_CLASSNAME, buttonClassName)}
      iconClassName={iconClassName}
      aria-label={isFilterRowVisible ? "Hide filters" : "Show filters"}
      aria-pressed={isFilterRowVisible}
    />
  );
}
