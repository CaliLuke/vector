/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// plane imports
import type { IFilterOption, TFilterValue } from "@plane/types";
import { cn } from "@plane/utils/common";
// local imports
import { COMMON_FILTER_ITEM_BORDER_CLASSNAME } from "../../shared";

export const getFormattedOptions = <V extends TFilterValue>(options: IFilterOption<V>[]) =>
  options.map((option) => ({
    value: option.value,
    content: (
      <div className="flex items-center gap-2 transition-all duration-200 ease-in-out">
        {option.icon && (
          <span className={cn("transition-transform duration-200", option.iconClassName)}>{option.icon}</span>
        )}
        <span>{option.label}</span>
      </div>
    ),
    query: option.label.toString().toLowerCase(),
    disabled: option.disabled,
    tooltip: option.description,
  }));

export const getCommonCustomSearchSelectProps = (isDisabled?: boolean) => ({
  customButtonClassName: cn(
    "h-full w-full px-2 text-13 font-regular transition-all duration-300 ease-in-out",
    !isDisabled && COMMON_FILTER_ITEM_BORDER_CLASSNAME,
    isDisabled && "hover:bg-surface-1"
  ),
  optionsClassName: "w-56",
  maxHeight: "md" as const,
});
