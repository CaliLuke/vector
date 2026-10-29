/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ComponentProps } from "react";
import type { TIssue } from "@plane/types";
import { getDate, renderFormattedPayloadDate } from "@plane/utils/datetime";
import { DateRangeDropdown } from "@/components/dropdowns/date-range";

type IssueDates = Pick<TIssue, "start_date" | "target_date">;
type Props = Pick<
  ComponentProps<typeof DateRangeDropdown>,
  "disabled" | "placement" | "renderByDefault" | "renderInPortal" | "clearIconClassName"
> & {
  dates: IssueDates;
  onChange: (dates: IssueDates) => void;
  highlight: boolean;
};

export function IssuePropertyDateRange({ dates, onChange, highlight, ...dropdownProps }: Props) {
  return (
    <DateRangeDropdown
      {...dropdownProps}
      value={{ from: getDate(dates.start_date) || undefined, to: getDate(dates.target_date) || undefined }}
      onSelect={(range) =>
        onChange({
          start_date: renderFormattedPayloadDate(range?.from) ?? null,
          target_date: renderFormattedPayloadDate(range?.to) ?? null,
        })
      }
      hideIcon={{ from: false }}
      isClearable
      mergeDates
      buttonVariant={dates.start_date || dates.target_date ? "border-with-text" : "border-without-text"}
      buttonClassName={highlight ? "text-danger-primary" : ""}
      showTooltip
      renderPlaceholder={false}
      customTooltipHeading="Date Range"
    />
  );
}
