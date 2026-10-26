/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { useRef, useState } from "react";
import type { Placement } from "@popperjs/core";
import { ArrowRight, CalendarDays } from "lucide-react";
// plane imports
import { useTranslation } from "@plane/i18n";
// ui
import type { DateRange, Matcher } from "@plane/propel/calendar";
import { Calendar } from "@plane/propel/calendar";
import { Popover } from "@plane/propel/popover";
import { CloseIcon, DueDatePropertyIcon } from "@plane/propel/icons";
import { cn } from "@plane/utils/common";
import { renderFormattedDate } from "@plane/utils/datetime";
// helpers
// hooks
import { useCurrentUserProfile } from "@/hooks/use-current-user-profile";
// components
import { DropdownButton } from "./buttons";
import { MergedDateDisplay } from "./merged-date";
// types
import type { TButtonVariants } from "./types";

type Props = {
  applyButtonText?: string;
  bothRequired?: boolean;
  buttonClassName?: string;
  buttonContainerClassName?: string;
  buttonFromDateClassName?: string;
  buttonToDateClassName?: string;
  buttonVariant: TButtonVariants;
  cancelButtonText?: string;
  className?: string;
  clearIconClassName?: string;
  disabled?: boolean;
  hideIcon?: {
    from?: boolean;
    to?: boolean;
  };
  isClearable?: boolean;
  mergeDates?: boolean;
  minDate?: Date;
  maxDate?: Date;
  onSelect?: (range: DateRange | undefined) => void;
  placeholder?: {
    from?: string;
    to?: string;
  };
  placement?: Placement;
  required?: boolean;
  showTooltip?: boolean;
  tabIndex?: number;
  value: {
    from: Date | undefined;
    to: Date | undefined;
  };
  renderByDefault?: boolean;
  renderPlaceholder?: boolean;
  customTooltipContent?: React.ReactNode;
  customTooltipHeading?: string;
  defaultOpen?: boolean;
  renderInPortal?: boolean;
};

export function DateRangeDropdown(props: Props) {
  const { t } = useTranslation();
  const {
    buttonClassName,
    buttonContainerClassName,
    buttonFromDateClassName,
    buttonToDateClassName,
    buttonVariant,
    className,
    clearIconClassName = "",
    disabled = false,
    hideIcon = {
      from: true,
      to: true,
    },
    isClearable = false,
    mergeDates,
    minDate,
    maxDate,
    onSelect,
    placeholder = {
      from: t("project_cycles.add_date"),
      to: t("project_cycles.add_date"),
    },
    placement,
    showTooltip = false,
    tabIndex,
    value,
    renderByDefault = true,
    renderPlaceholder = true,
    customTooltipContent,
    customTooltipHeading,
    defaultOpen = false,
    renderInPortal = false,
  } = props;
  // states
  const [isOpen, setIsOpen] = useState(defaultOpen);
  const containerRef = useRef<HTMLDivElement | null>(null);
  const dateRange = value;
  // hooks
  const { data } = useCurrentUserProfile();
  const startOfWeek = data?.start_of_the_week;
  const disabledDays: Matcher[] = [];
  if (minDate) disabledDays.push({ before: minDate });
  if (maxDate) disabledDays.push({ after: maxDate });

  const clearDates = () => {
    const clearedRange = { from: undefined, to: undefined };
    onSelect?.(clearedRange);
  };

  const hasDisplayedDates = dateRange.from || dateRange.to;

  const comboButton = (
    <button
      type="button"
      className={cn(
        "clickable block h-full max-w-full outline-none",
        {
          "cursor-not-allowed text-secondary": disabled,
          "cursor-pointer": !disabled,
        },
        buttonContainerClassName
      )}
      tabIndex={tabIndex}
      disabled={disabled}
    >
      <DropdownButton
        className={buttonClassName}
        isActive={isOpen}
        tooltipHeading={customTooltipHeading ?? t("project_cycles.date_range")}
        tooltipContent={
          <>
            {customTooltipContent ?? (
              <>
                {dateRange.from ? renderFormattedDate(dateRange.from) : ""}
                {dateRange.from && dateRange.to ? " - " : ""}
                {dateRange.to ? renderFormattedDate(dateRange.to) : ""}
              </>
            )}
          </>
        }
        showTooltip={showTooltip}
        variant={buttonVariant}
        renderToolTipByDefault={renderByDefault}
      >
        {mergeDates ? (
          // Merged date display
          <div className="flex w-full items-center gap-1.5">
            {!hideIcon.from && <CalendarDays className="h-3 w-3 flex-shrink-0" />}
            {dateRange.from || dateRange.to ? (
              <MergedDateDisplay
                startDate={dateRange.from}
                endDate={dateRange.to}
                className="flex-grow truncate text-11"
              />
            ) : (
              renderPlaceholder && (
                <>
                  <span className="text-placeholder">{placeholder.from}</span>
                  {placeholder.from && placeholder.to && (
                    <ArrowRight className="h-3 w-3 flex-shrink-0 text-placeholder" />
                  )}
                  <span className="text-placeholder">{placeholder.to}</span>
                </>
              )
            )}
            {isClearable && !disabled && hasDisplayedDates && (
              <CloseIcon
                className={cn("h-2.5 w-2.5 flex-shrink-0 cursor-pointer", clearIconClassName)}
                onClick={(e) => {
                  e.stopPropagation();
                  e.preventDefault();
                  clearDates();
                }}
              />
            )}
          </div>
        ) : (
          // Original separate date display
          <>
            <span
              className={cn(
                "flex h-full flex-grow items-center justify-center gap-1 rounded-xs",
                buttonFromDateClassName
              )}
            >
              {!hideIcon.from && <CalendarDays className="h-3 w-3 flex-shrink-0" />}
              {dateRange.from ? renderFormattedDate(dateRange.from) : renderPlaceholder ? placeholder.from : ""}
            </span>
            <ArrowRight className="h-3 w-3 flex-shrink-0" />
            <span
              className={cn(
                "flex h-full flex-grow items-center justify-center gap-1 rounded-xs",
                buttonToDateClassName
              )}
            >
              {!hideIcon.to && <DueDatePropertyIcon className="h-3 w-3 flex-shrink-0" />}
              {dateRange.to ? renderFormattedDate(dateRange.to) : renderPlaceholder ? placeholder.to : ""}
            </span>
            {isClearable && !disabled && hasDisplayedDates && (
              <CloseIcon
                className={cn("ml-1 h-2.5 w-2.5 flex-shrink-0 cursor-pointer", clearIconClassName)}
                onClick={(e) => {
                  e.stopPropagation();
                  e.preventDefault();
                  clearDates();
                }}
              />
            )}
          </>
        )}
      </DropdownButton>
    </button>
  );

  return (
    <div ref={containerRef} className={cn("h-full", className)}>
      <Popover open={isOpen} onOpenChange={setIsOpen}>
        <Popover.Button render={comboButton} disabled={disabled} />
        <Popover.Panel
          placement={placement ?? "bottom-start"}
          sideOffset={4}
          initialFocus={false}
          containerRef={renderInPortal ? undefined : containerRef}
          positionerClassName="z-30"
          className="overflow-hidden rounded-md border-[0.5px] border-subtle-1 bg-surface-1"
        >
          <Calendar
            className="rounded-md border border-subtle p-3 text-12"
            captionLayout="dropdown"
            selected={dateRange}
            onSelect={(val: DateRange | undefined) => {
              onSelect?.(val);
            }}
            mode="range"
            disabled={disabledDays}
            showOutsideDays
            fixedWeeks
            weekStartsOn={startOfWeek}
            // oxlint-disable-next-line jsx_a11y/no-autofocus -- DayPicker v10 uses autoFocus to preserve the previous initialFocus behavior in the opened calendar.
            autoFocus
          />
        </Popover.Panel>
      </Popover>
    </div>
  );
}
