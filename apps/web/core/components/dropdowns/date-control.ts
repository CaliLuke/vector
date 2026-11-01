/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Matcher } from "@plane/propel/calendar";
import { cn } from "@plane/utils/common";

export function dateDisabledDays(minDate?: Date, maxDate?: Date): Matcher[] {
  const days: Matcher[] = [];
  if (minDate) days.push({ before: minDate });
  if (maxDate) days.push({ after: maxDate });
  return days;
}

export function dateTriggerClassName(disabled: boolean, className?: string) {
  return cn(
    "clickable block h-full max-w-full outline-none",
    {
      "cursor-not-allowed text-secondary": disabled,
      "cursor-pointer": !disabled,
    },
    className
  );
}
