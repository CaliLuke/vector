/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { useState } from "react";
import { CalendarDays } from "lucide-react";
// ui
import type { Matcher } from "@plane/propel/calendar";
import { Calendar } from "@plane/propel/calendar";
import { Popover } from "@plane/propel/popover";
import { CloseIcon } from "@plane/propel/icons";
import { cn } from "@plane/utils/common";
import { renderFormattedDate, getDate } from "@plane/utils/datetime";
// helpers
// hooks
import { useCurrentUserProfile } from "@/hooks/use-current-user-profile";
// components
import { DropdownButton } from "./buttons";
// constants
import { BUTTON_VARIANTS_WITH_TEXT } from "./constants";
// types
import type { TDropdownProps } from "./types";

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
    buttonClassName = "",
    buttonContainerClassName,
    buttonVariant,
    className = "",
    clearIconClassName = "",
    defaultOpen = false,
    optionsClassName = "",
    closeOnSelect = true,
    disabled = false,
    hideIcon = false,
    icon = <CalendarDays className="h-3 w-3 flex-shrink-0" />,
    isClearable = true,
    minDate,
    maxDate,
    onChange,
    onClose,
    placeholder = "Date",
    placement,
    showTooltip = false,
    tabIndex,
    value,
    formatToken,
    renderByDefault = true,
    labelClassName = "",
  } = props;
  // states
  const [isOpen, setIsOpen] = useState(defaultOpen);
  // hooks
  const { data } = useCurrentUserProfile();
  const startOfWeek = data?.start_of_the_week;
  const isDateSelected = value && value.toString().trim() !== "";

  const handleOpenChange = (open: boolean) => {
    setIsOpen(open);
    if (!open) onClose?.();
  };

  const dropdownOnChange = (val: Date | null) => {
    onChange(val);
    if (closeOnSelect) {
      handleOpenChange(false);
    }
  };

  const disabledDays: Matcher[] = [];
  if (minDate) disabledDays.push({ before: minDate });
  if (maxDate) disabledDays.push({ after: maxDate });

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
        tooltipHeading={placeholder}
        tooltipContent={value ? renderFormattedDate(value, formatToken) : "None"}
        showTooltip={showTooltip}
        variant={buttonVariant}
        renderToolTipByDefault={renderByDefault}
      >
        {!hideIcon && icon}
        {BUTTON_VARIANTS_WITH_TEXT.includes(buttonVariant) && (
          <span className={cn("flex-grow truncate text-left text-body-xs-medium", labelClassName)}>
            {value ? renderFormattedDate(value, formatToken) : placeholder}
          </span>
        )}
        {isClearable && !disabled && isDateSelected && (
          <CloseIcon
            className={cn("h-2.5 w-2.5 flex-shrink-0", clearIconClassName)}
            onClick={(e) => {
              e.stopPropagation();
              e.preventDefault();
              onChange(null);
            }}
          />
        )}
      </DropdownButton>
    </button>
  );

  return (
    <div className={cn("h-full", className)}>
      <Popover open={isOpen} onOpenChange={handleOpenChange}>
        <Popover.Button render={comboButton} disabled={disabled} />
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
              dropdownOnChange(date ?? null);
            }}
            showOutsideDays
            // oxlint-disable-next-line jsx_a11y/no-autofocus -- DayPicker v10 uses autoFocus to preserve the previous initialFocus behavior in the opened calendar.
            autoFocus
            disabled={disabledDays}
            mode="single"
            fixedWeeks
            weekStartsOn={startOfWeek}
          />
        </Popover.Panel>
      </Popover>
    </div>
  );
}
