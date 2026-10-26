/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";

import { useCalendarHeaderPopover } from "./use-calendar-header-popover";
import { Popover, Transition } from "@headlessui/react";
import { ChevronLeftIcon, ChevronRightIcon } from "@plane/propel/icons";
//hooks
// icons
// constants
import { getDate } from "@plane/utils/datetime";
import { MONTHS_LIST } from "@plane/constants";
import type { ICalendarWeek } from "@plane/types";
import type { TCalendarDates } from "@/providers/issue-layout-ui-state";
import type { IIssueDisplayFilterOptions } from "@plane/types";
// helpers

interface Props {
  displayFilters: IIssueDisplayFilterOptions;
  activeMonthDate: Date;
  activeWeek: ICalendarWeek | undefined;
  updateCalendarDates: (filters: Partial<TCalendarDates>) => void;
}
export const CalendarMonthsDropdown = function CalendarMonthsDropdown(props: Props) {
  const { displayFilters, activeMonthDate, activeWeek, updateCalendarDates } = props;

  const calendarLayout = displayFilters.calendar?.layout ?? "month";

  const { setReferenceElement, setPopperElement, styles, attributes } = useCalendarHeaderPopover();

  const getWeekLayoutHeader = (): string => {
    const allDaysOfActiveWeek = activeWeek;

    if (!allDaysOfActiveWeek) return "Week view";

    const daysList = Object.keys(allDaysOfActiveWeek);

    const firstDay = getDate(daysList[0]);
    const lastDay = getDate(daysList[daysList.length - 1]);

    if (!firstDay || !lastDay) return "Week view";

    if (firstDay.getMonth() === lastDay.getMonth() && firstDay.getFullYear() === lastDay.getFullYear())
      return `${MONTHS_LIST[firstDay.getMonth() + 1].title} ${firstDay.getFullYear()}`;

    if (firstDay.getFullYear() !== lastDay.getFullYear()) {
      return `${MONTHS_LIST[firstDay.getMonth() + 1].shortTitle} ${firstDay.getFullYear()} - ${
        MONTHS_LIST[lastDay.getMonth() + 1].shortTitle
      } ${lastDay.getFullYear()}`;
    } else
      return `${MONTHS_LIST[firstDay.getMonth() + 1].shortTitle} - ${
        MONTHS_LIST[lastDay.getMonth() + 1].shortTitle
      } ${lastDay.getFullYear()}`;
  };

  const handleDateChange = (date: Date) => {
    updateCalendarDates({ activeMonthDate: date });
  };

  return (
    <Popover className="relative">
      <Popover.Button as={React.Fragment}>
        <button
          type="button"
          ref={setReferenceElement}
          className="text-18 font-semibold outline-none"
          disabled={calendarLayout === "week"}
        >
          {calendarLayout === "month"
            ? `${MONTHS_LIST[activeMonthDate.getMonth() + 1].title} ${activeMonthDate.getFullYear()}`
            : getWeekLayoutHeader()}
        </button>
      </Popover.Button>
      <Transition
        as={React.Fragment}
        enter="transition ease-out duration-200"
        enterFrom="opacity-0 translate-y-1"
        enterTo="opacity-100 translate-y-0"
        leave="transition ease-in duration-150"
        leaveFrom="opacity-100 translate-y-0"
        leaveTo="opacity-0 translate-y-1"
      >
        <Popover.Panel className="fixed z-50">
          <div
            ref={setPopperElement}
            style={styles.popper}
            {...attributes.popper}
            className="w-56 divide-y divide-subtle-1 rounded-sm border border-subtle bg-surface-1 p-3 shadow-raised-200"
          >
            <div className="flex items-center justify-between gap-2 pb-3">
              <button
                type="button"
                className="grid place-items-center"
                onClick={() => {
                  const previousYear = new Date(activeMonthDate.getFullYear() - 1, activeMonthDate.getMonth(), 1);
                  handleDateChange(previousYear);
                }}
              >
                <ChevronLeftIcon height={14} width={14} />
              </button>
              <span className="text-11">{activeMonthDate.getFullYear()}</span>
              <button
                type="button"
                className="grid place-items-center"
                onClick={() => {
                  const nextYear = new Date(activeMonthDate.getFullYear() + 1, activeMonthDate.getMonth(), 1);
                  handleDateChange(nextYear);
                }}
              >
                <ChevronRightIcon height={14} width={14} />
              </button>
            </div>
            <div className="grid grid-cols-4 items-stretch justify-items-stretch gap-4 pt-3">
              {Object.values(MONTHS_LIST).map((month, index) => (
                <button
                  key={month.shortTitle}
                  type="button"
                  className="rounded-sm py-0.5 text-11 hover:bg-layer-1"
                  onClick={() => {
                    const newDate = new Date(activeMonthDate.getFullYear(), index, 1);
                    handleDateChange(newDate);
                  }}
                >
                  {month.shortTitle}
                </button>
              ))}
            </div>
          </div>
        </Popover.Panel>
      </Transition>
    </Popover>
  );
};
