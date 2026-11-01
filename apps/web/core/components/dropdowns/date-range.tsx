/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { useRef, useState } from "react";
import type { Placement } from "@popperjs/core";
import { ArrowRight, CalendarDays } from "lucide-react";
import { useTranslation } from "@plane/i18n";
import type { DateRange } from "@plane/propel/calendar";
import { Calendar } from "@plane/propel/calendar";
import { Popover } from "@plane/propel/popover";
import { CloseIcon, DueDatePropertyIcon } from "@plane/propel/icons";
import { cn } from "@plane/utils/common";
import { renderFormattedDate } from "@plane/utils/datetime";
import { useCurrentUserProfile } from "@/hooks/use-current-user-profile";
import { DropdownButton } from "./buttons";
import { MergedDateDisplay } from "./merged-date";
import type { TButtonVariants } from "./types";
import { dateDisabledDays, dateTriggerClassName } from "./date-control";

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
  const {
    defaultOpen = false,
    value,
    minDate,
    maxDate,
    onSelect,
    className,
    placement,
    disabled = false,
    buttonContainerClassName,
    tabIndex,
    renderInPortal = false,
  } = props;
  const [isOpen, setIsOpen] = useState(defaultOpen);
  const containerRef = useRef<HTMLDivElement | null>(null);
  const { data } = useCurrentUserProfile();
  return (
    <div ref={containerRef} className={cn("h-full", className)}>
      <Popover open={isOpen} onOpenChange={setIsOpen}>
        <Popover.Button
          disabled={disabled}
          render={
            <button
              type="button"
              className={dateTriggerClassName(disabled, buttonContainerClassName)}
              tabIndex={tabIndex}
              disabled={disabled}
            >
              <DateRangeButtonContent options={props} isOpen={isOpen} />
            </button>
          }
        />
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
            selected={value}
            onSelect={(range: DateRange | undefined) => onSelect?.(range)}
            mode="range"
            disabled={dateDisabledDays(minDate, maxDate)}
            showOutsideDays
            fixedWeeks
            weekStartsOn={data?.start_of_the_week}
            // oxlint-disable-next-line jsx_a11y/no-autofocus -- DayPicker v10 uses autoFocus to preserve the previous initialFocus behavior in the opened calendar.
            autoFocus
          />
        </Popover.Panel>
      </Popover>
    </div>
  );
}

function DateRangeButtonContent({ options, isOpen }: { options: Props; isOpen: boolean }) {
  const { t } = useTranslation();
  const {
    buttonClassName,
    buttonVariant,
    customTooltipHeading,
    customTooltipContent,
    value,
    showTooltip = false,
    renderByDefault = true,
    mergeDates,
  } = options;
  return (
    <DropdownButton
      className={buttonClassName}
      isActive={isOpen}
      tooltipHeading={customTooltipHeading ?? t("project_cycles.date_range")}
      tooltipContent={customTooltipContent ?? <DateRangeTooltip value={value} />}
      showTooltip={showTooltip}
      variant={buttonVariant}
      renderToolTipByDefault={renderByDefault}
    >
      {mergeDates ? (
        <div className="flex w-full items-center gap-1.5">
          <MergedRangeLabel options={options} />
          <ClearDateRange options={options} />
        </div>
      ) : (
        <>
          <SeparateRangeLabel options={options} />
          <ClearDateRange options={options} />
        </>
      )}
    </DropdownButton>
  );
}

function DateRangeTooltip({ value }: Pick<Props, "value">) {
  return (
    <>
      {value.from ? renderFormattedDate(value.from) : ""}
      {value.from && value.to ? " - " : ""}
      {value.to ? renderFormattedDate(value.to) : ""}
    </>
  );
}

function ClearDateRange({ options }: { options: Props }) {
  const { isClearable = false, disabled, value, clearIconClassName, mergeDates, onSelect } = options;
  if (!isClearable || disabled || !(value.from || value.to)) return null;
  return (
    <CloseIcon
      className={cn("h-2.5 w-2.5 flex-shrink-0 cursor-pointer", !mergeDates && "ml-1", clearIconClassName)}
      onClick={(event) => {
        event.stopPropagation();
        event.preventDefault();
        onSelect?.({ from: undefined, to: undefined });
      }}
    />
  );
}

function MergedRangeLabel({ options }: { options: Props }) {
  const { t } = useTranslation();
  const {
    value,
    hideIcon = { from: true, to: true },
    renderPlaceholder = true,
    placeholder = { from: t("project_cycles.add_date"), to: t("project_cycles.add_date") },
  } = options;
  return (
    <>
      {!hideIcon.from && <CalendarDays className="h-3 w-3 flex-shrink-0" />}
      {value.from || value.to ? (
        <MergedDateDisplay startDate={value.from} endDate={value.to} className="flex-grow truncate text-11" />
      ) : (
        renderPlaceholder && (
          <>
            <span className="text-placeholder">{placeholder.from}</span>
            {placeholder.from && placeholder.to && <ArrowRight className="h-3 w-3 flex-shrink-0 text-placeholder" />}
            <span className="text-placeholder">{placeholder.to}</span>
          </>
        )
      )}
    </>
  );
}

function SeparateRangeLabel({ options }: { options: Props }) {
  const { t } = useTranslation();
  const {
    value,
    hideIcon = { from: true, to: true },
    renderPlaceholder = true,
    buttonFromDateClassName,
    buttonToDateClassName,
    placeholder = { from: t("project_cycles.add_date"), to: t("project_cycles.add_date") },
  } = options;
  return (
    <>
      <span
        className={cn("flex h-full flex-grow items-center justify-center gap-1 rounded-xs", buttonFromDateClassName)}
      >
        {!hideIcon.from && <CalendarDays className="h-3 w-3 flex-shrink-0" />}
        {value.from ? renderFormattedDate(value.from) : renderPlaceholder ? placeholder.from : ""}
      </span>
      <ArrowRight className="h-3 w-3 flex-shrink-0" />
      <span className={cn("flex h-full flex-grow items-center justify-center gap-1 rounded-xs", buttonToDateClassName)}>
        {!hideIcon.to && <DueDatePropertyIcon className="h-3 w-3 flex-shrink-0" />}
        {value.to ? renderFormattedDate(value.to) : renderPlaceholder ? placeholder.to : ""}
      </span>
    </>
  );
}
