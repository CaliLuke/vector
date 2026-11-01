/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { useState } from "react";
import { CalendarDays } from "lucide-react";
import { Calendar } from "@plane/propel/calendar";
import { Popover } from "@plane/propel/popover";
import { CloseIcon } from "@plane/propel/icons";
import { cn } from "@plane/utils/common";
import { renderFormattedDate, getDate } from "@plane/utils/datetime";
import { useCurrentUserProfile } from "@/hooks/use-current-user-profile";
import { DropdownButton } from "./buttons";
import { BUTTON_VARIANTS_WITH_TEXT } from "./constants";
import type { TDropdownProps } from "./types";
import { dateDisabledDays, dateTriggerClassName } from "./date-control";

type Props = TDropdownProps & {
  clearIconClassName?: string;
  defaultOpen?: boolean;
  optionsClassName?: string;
  icon?: React.ReactNode;
  isClearable?: boolean;
  minDate?: Date;
  maxDate?: Date;
  onChange: (val: Date | null) => void;
  onClose?: () => void;
  value: Date | string | null;
  closeOnSelect?: boolean;
  formatToken?: string;
  renderByDefault?: boolean;
  labelClassName?: string;
};

export function DateDropdown(props: Props) {
  const {
    defaultOpen = false,
    closeOnSelect = true,
    onClose,
    onChange,
    minDate,
    maxDate,
    className,
    buttonContainerClassName,
    disabled = false,
    tabIndex,
    placement,
    value,
    optionsClassName,
  } = props;
  const [isOpen, setIsOpen] = useState(defaultOpen);
  const { data } = useCurrentUserProfile();
  const handleOpenChange = (open: boolean) => {
    setIsOpen(open);
    if (!open) onClose?.();
  };
  return (
    <div className={cn("h-full", className)}>
      <Popover open={isOpen} onOpenChange={handleOpenChange}>
        <Popover.Button
          disabled={disabled}
          render={
            <button
              type="button"
              className={dateTriggerClassName(disabled, buttonContainerClassName)}
              tabIndex={tabIndex}
              disabled={disabled}
            >
              <DateButtonContent options={props} isOpen={isOpen} />
            </button>
          }
        />
        <Popover.Panel
          placement={placement ?? "bottom-start"}
          sideOffset={4}
          initialFocus={false}
          positionerClassName="z-30"
          className={cn(
            "overflow-hidden rounded-md border-[0.5px] border-strong bg-surface-1 shadow-raised-200",
            optionsClassName
          )}
        >
          <Calendar
            className="rounded-md border border-subtle p-3"
            captionLayout="dropdown"
            selected={getDate(value)}
            defaultMonth={getDate(value)}
            onSelect={(date: Date | undefined) => {
              onChange(date ?? null);
              if (closeOnSelect) handleOpenChange(false);
            }}
            showOutsideDays
            // oxlint-disable-next-line jsx_a11y/no-autofocus -- DayPicker v10 uses autoFocus to preserve the previous initialFocus behavior in the opened calendar.
            autoFocus
            disabled={dateDisabledDays(minDate, maxDate)}
            mode="single"
            fixedWeeks
            weekStartsOn={data?.start_of_the_week}
          />
        </Popover.Panel>
      </Popover>
    </div>
  );
}

function DateButtonContent({ options, isOpen }: { options: Props; isOpen: boolean }) {
  const {
    buttonClassName,
    buttonVariant,
    placeholder = "Date",
    value,
    formatToken,
    showTooltip = false,
    renderByDefault = true,
    isClearable = true,
    disabled,
    clearIconClassName,
    onChange,
  } = options;
  return (
    <DropdownButton
      className={buttonClassName}
      isActive={isOpen}
      tooltipHeading={placeholder}
      tooltipContent={value ? renderFormattedDate(value, formatToken) : "None"}
      showTooltip={showTooltip}
      variant={buttonVariant}
      renderToolTipByDefault={renderByDefault}
    >
      <DateValueLabel options={options} />
      {isClearable && !disabled && value && value.toString().trim() !== "" && (
        <CloseIcon
          className={cn("h-2.5 w-2.5 flex-shrink-0", clearIconClassName)}
          onClick={(event) => {
            event.stopPropagation();
            event.preventDefault();
            onChange(null);
          }}
        />
      )}
    </DropdownButton>
  );
}

function DateValueLabel({ options }: { options: Props }) {
  const {
    hideIcon = false,
    icon = <CalendarDays className="h-3 w-3 flex-shrink-0" />,
    buttonVariant,
    labelClassName,
    value,
    formatToken,
    placeholder = "Date",
  } = options;
  return (
    <>
      {!hideIcon && icon}
      {BUTTON_VARIANTS_WITH_TEXT.includes(buttonVariant) && (
        <span className={cn("flex-grow truncate text-left text-body-xs-medium", labelClassName)}>
          {value ? renderFormattedDate(value, formatToken) : placeholder}
        </span>
      )}
    </>
  );
}
