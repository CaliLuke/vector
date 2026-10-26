/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";

// plane imports
import type { TDateRangeFilterFieldConfig, TFilterConditionNodeForDisplay, TFilterProperty } from "@plane/types";
import { cn } from "@plane/utils/common";
import { getDate, isValidDate, renderFormattedPayloadDate } from "@plane/utils/datetime";
import { toFilterArray } from "@plane/utils/rich-filters";
// components
import { DateRangeDropdown } from "@/components/dropdowns/date-range";
// local imports
import { COMMON_FILTER_ITEM_BORDER_CLASSNAME, EMPTY_FILTER_PLACEHOLDER_TEXT } from "../../shared";

type TDateRangeFilterValueInputProps<P extends TFilterProperty> = {
  config: TDateRangeFilterFieldConfig<string>;
  condition: TFilterConditionNodeForDisplay<P, string>;
  isDisabled?: boolean;
  onChange: (value: string[]) => void;
};

export const DateRangeFilterValueInput = function DateRangeFilterValueInput<P extends TFilterProperty>(
  props: TDateRangeFilterValueInputProps<P>
) {
  const { config, condition, isDisabled, onChange } = props;
  // derived values
  const [fromRaw, toRaw] = toFilterArray(condition.value) ?? [];
  const from = isValidDate(fromRaw) ? getDate(fromRaw) : undefined;
  const to = isValidDate(toRaw) ? getDate(toRaw) : undefined;
  const isIncomplete = !from || !to;

  // Handler for date range selection
  const handleSelect = (range: { from?: Date; to?: Date } | undefined) => {
    const formattedFrom = range?.from ? renderFormattedPayloadDate(range.from) : undefined;
    const formattedTo = range?.to ? renderFormattedPayloadDate(range.to) : undefined;
    if (formattedFrom && formattedTo) {
      onChange([formattedFrom, formattedTo]);
    } else {
      onChange([]);
    }
  };

  return (
    <DateRangeDropdown
      value={{ from, to }}
      onSelect={handleSelect}
      minDate={config.min}
      maxDate={config.max}
      mergeDates
      placeholder={{ from: EMPTY_FILTER_PLACEHOLDER_TEXT }}
      buttonVariant="transparent-with-text"
      buttonClassName={cn("rounded-none", {
        [COMMON_FILTER_ITEM_BORDER_CLASSNAME]: !isDisabled,
        "text-danger-primary": isIncomplete,
        "hover:bg-surface-1": isDisabled,
      })}
      renderPlaceholder
      renderInPortal
      defaultOpen={isIncomplete}
      disabled={isDisabled}
    />
  );
};
