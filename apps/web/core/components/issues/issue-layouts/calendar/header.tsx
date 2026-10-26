/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// components
import { useTranslation } from "@plane/i18n";
import { ChevronLeftIcon, ChevronRightIcon } from "@plane/propel/icons";
import type { IIssueDisplayFilterOptions } from "@plane/types";
import { Row } from "@plane/ui";
// icons
import type { TCalendarDates } from "@/providers/issue-layout-ui-state";
import type { ICalendarWeek } from "@plane/types";
import { CalendarMonthsDropdown, CalendarOptionsDropdown } from "./dropdowns";
import type { TCalendarDisplayFilterUpdate } from "@/hooks/use-calendar-display-filter-update";
import { shiftCalendarDate } from "./calendar-state";

interface ICalendarHeader {
  displayFilters: IIssueDisplayFilterOptions;
  updateFilters: TCalendarDisplayFilterUpdate;
  setSelectedDate: (date: Date) => void;
  activeMonthDate: Date;
  activeWeekDate: Date;
  activeWeek: ICalendarWeek | undefined;
  updateCalendarDates: (filters: Partial<TCalendarDates>) => void;
}

export const CalendarHeader = function CalendarHeader(props: ICalendarHeader) {
  const {
    displayFilters,
    updateFilters,
    setSelectedDate,
    activeMonthDate,
    activeWeekDate,
    activeWeek,
    updateCalendarDates,
  } = props;

  const { t } = useTranslation();

  const calendarLayout = displayFilters.calendar?.layout ?? "month";

  const handlePrevious = () => {
    if (calendarLayout === "month") {
      updateCalendarDates({ activeMonthDate: shiftCalendarDate(activeMonthDate, -1, "month") });
    } else {
      updateCalendarDates({ activeWeekDate: shiftCalendarDate(activeWeekDate, -1, "week") });
    }
  };

  const handleNext = () => {
    if (calendarLayout === "month") {
      updateCalendarDates({ activeMonthDate: shiftCalendarDate(activeMonthDate, 1, "month") });
    } else {
      updateCalendarDates({ activeWeekDate: shiftCalendarDate(activeWeekDate, 1, "week") });
    }
  };

  const handleToday = () => {
    const today = new Date();
    const firstDayOfCurrentMonth = new Date(today.getFullYear(), today.getMonth(), 1);

    updateCalendarDates({ activeMonthDate: firstDayOfCurrentMonth, activeWeekDate: today });
    setSelectedDate(today);
  };

  return (
    <Row className="mb-4 flex items-center justify-between gap-2">
      <div className="flex items-center gap-1.5">
        <button type="button" className="grid place-items-center" onClick={handlePrevious}>
          <ChevronLeftIcon height={16} width={16} strokeWidth={2} />
        </button>
        <button type="button" className="grid place-items-center" onClick={handleNext}>
          <ChevronRightIcon height={16} width={16} strokeWidth={2} />
        </button>
        <CalendarMonthsDropdown
          displayFilters={displayFilters}
          activeMonthDate={activeMonthDate}
          activeWeek={activeWeek}
          updateCalendarDates={updateCalendarDates}
        />
      </div>
      <div className="flex items-center gap-1.5">
        <button
          type="button"
          className="rounded-sm bg-layer-transparent px-2.5 py-1 text-11 font-medium text-secondary hover:bg-layer-transparent-hover hover:text-primary"
          onClick={handleToday}
        >
          {t("common.today")}
        </button>
        <CalendarOptionsDropdown displayFilters={displayFilters} updateFilters={updateFilters} />
      </div>
    </Row>
  );
};
