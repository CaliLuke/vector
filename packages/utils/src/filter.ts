/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { differenceInCalendarDays } from "date-fns/differenceInCalendarDays";
// local imports
import { getDate } from "./datetime";

/**
 * @description calculates the total number of filters applied
 * @param {T} filters
 * @returns {number}
 */
export const calculateTotalFilters = (filters: object | null | undefined): number =>
  Object.values(filters ?? {}).reduce((total: number, value: unknown) => {
    if (Array.isArray(value)) return total + value.length;
    return total + (value === true ? 1 : 0);
  }, 0);

/** Empty selections do not constrain a value. Values within one selection are ORed. */
export const matchesFilterValue = <T>(value: T, selected: readonly T[] | null | undefined): boolean =>
  !selected?.length || selected.includes(value);

export const matchesAnyFilterValue = <T>(
  values: readonly T[] | null | undefined,
  selected: readonly T[] | null | undefined
): boolean => !selected?.length || selected.some((value) => values?.includes(value));

/** Date constraints within one filter are ANDed, including both ends of a range. */
export const matchesDateFilters = (
  value: string | Date | null | undefined,
  filters: readonly string[] | null | undefined
): boolean => {
  if (!filters?.length) return true;
  const date = getDate(value);
  return Boolean(date && filters.every((filter) => satisfiesDateFilter(date, filter)));
};

/**
 * @description checks if the date satisfies the filter
 * @param {Date} date
 * @param {string} filter
 * @returns {boolean}
 */
export const satisfiesDateFilter = (date: Date, filter: string): boolean => {
  const [value, operator, from] = filter.split(";");

  const dateValue = getDate(value);
  const differenceInDays = differenceInCalendarDays(date, new Date());

  if (operator === "custom" && from === "custom") return matchesCalendarShortcut(value, differenceInDays);

  if (!from) {
    if (!dateValue) return false;
    return operator === "after" ? date >= dateValue : operator === "before" && date <= dateValue;
  }

  if (from === "fromnow") {
    const offsets = new Map([
      ["1_weeks", 7],
      ["2_weeks", 14],
      ["1_months", 30],
      ["2_months", 60],
    ]);
    const days = offsets.get(value);
    if (days === undefined) return false;
    if (operator === "before") return days !== 60 && differenceInDays <= -days;
    if (operator === "after") return differenceInDays >= days;
  }

  return false;
};

function matchesCalendarShortcut(value: string, differenceInDays: number): boolean {
  switch (value) {
    case "today":
      return differenceInDays === 0;
    case "yesterday":
      return differenceInDays === -1;
    case "last_7_days":
      return differenceInDays >= -7;
    case "last_30_days":
      return differenceInDays >= -30;
    default:
      return false;
  }
}
