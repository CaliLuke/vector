/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import type { LucideIcon } from "lucide-react";
import { useTranslation } from "@plane/i18n";
import { ChevronDownIcon } from "@plane/propel/icons";
// plane imports
import type { IUserLite } from "@plane/types";
import { ComboDropDown } from "@plane/ui";
import type { TComboDropDownRenderProps } from "@plane/ui";
// helpers
import { cn } from "@plane/utils/common";
// local imports
import { DropdownButton } from "../buttons";
import { BUTTON_VARIANTS_WITH_TEXT } from "../constants";
import { ButtonAvatars } from "./avatar";
import { MemberOptions } from "./member-options";
import type { MemberDropdownProps } from "./types";

type TMemberDropdownBaseProps = {
  getUserDetails: (userId: string) => IUserLite | undefined;
  getIsUserSuspended: (userId: string) => boolean;
  isRosterLoading: boolean;
  isRosterError: boolean;
  onRosterRetry: () => void;
  icon?: LucideIcon;
  memberIds?: string[];
  onClose?: () => void;
  onDropdownOpen?: () => void;
  optionsClassName?: string;
  renderByDefault?: boolean;
  workspaceSlug: string | undefined;
} & MemberDropdownProps;

export function MemberDropdownBase(props: TMemberDropdownBaseProps) {
  const { t } = useTranslation();
  const {
    button,
    buttonClassName,
    buttonContainerClassName,
    buttonVariant,
    className = "",
    disabled = false,
    dropdownArrow = false,
    dropdownArrowClassName = "",
    getUserDetails,
    getIsUserSuspended,
    isRosterLoading,
    isRosterError,
    onRosterRetry,
    hideIcon = false,
    icon,
    memberIds,
    onClose,
    onDropdownOpen,
    optionsClassName = "",
    placeholder = t("members"),
    placement,
    renderByDefault = true,
    showTooltip = false,
    showUserDetails = false,
    tabIndex,
    tooltipContent,
    value,
    workspaceSlug,
  } = props;
  // popper-js refs
  const [referenceElement, setReferenceElement] = useState<HTMLButtonElement | null>(null);

  const getDisplayName = (
    selectedValue: string | string[] | null,
    withUserDetails: boolean,
    emptyPlaceholder: string = ""
  ) => {
    if (Array.isArray(selectedValue)) {
      if (selectedValue.length > 0) {
        if (selectedValue.length === 1) {
          return getUserDetails(selectedValue[0])?.display_name || emptyPlaceholder;
        } else {
          return withUserDetails ? `${selectedValue.length} ${t("members").toLocaleLowerCase()}` : "";
        }
      } else {
        return emptyPlaceholder;
      }
    } else {
      if (withUserDetails && selectedValue) {
        return getUserDetails(selectedValue)?.display_name || emptyPlaceholder;
      } else {
        return emptyPlaceholder;
      }
    }
  };

  const comboButton = ({ open }: { open: boolean }) =>
    button ? (
      <button
        ref={setReferenceElement}
        type="button"
        className={cn("clickable block h-full w-full outline-none", buttonContainerClassName)}
        disabled={disabled}
        tabIndex={tabIndex}
      >
        {button}
      </button>
    ) : (
      <button
        ref={setReferenceElement}
        type="button"
        className={cn(
          "clickable block h-full max-w-full outline-none",
          {
            "cursor-not-allowed text-secondary": disabled,
            "cursor-pointer": !disabled,
          },
          buttonContainerClassName
        )}
        disabled={disabled}
        tabIndex={tabIndex}
      >
        <DropdownButton
          className={cn("text-11", buttonClassName)}
          isActive={open}
          tooltipHeading={placeholder}
          tooltipContent={
            tooltipContent ?? `${value?.length ?? 0} ${value?.length !== 1 ? t("assignees") : t("assignee")}`
          }
          showTooltip={showTooltip}
          variant={buttonVariant}
          renderToolTipByDefault={renderByDefault}
        >
          {!hideIcon && (
            <ButtonAvatars showTooltip={showTooltip} userIds={value} icon={icon} workspaceSlug={workspaceSlug} />
          )}
          {BUTTON_VARIANTS_WITH_TEXT.includes(buttonVariant) && (
            <span className="flex-grow truncate text-left text-body-xs-medium leading-5">
              {getDisplayName(value, showUserDetails, placeholder)}
            </span>
          )}
          {dropdownArrow && (
            <ChevronDownIcon className={cn("h-2.5 w-2.5 flex-shrink-0", dropdownArrowClassName)} aria-hidden="true" />
          )}
        </DropdownButton>
      </button>
    );

  const comboProps = {
    as: "div" as const,
    role: "group" as const,
    tabIndex,
    className: cn("h-full", className),
    disabled,
    onClose,
    onOpen: onDropdownOpen,
    button: comboButton,
    renderByDefault,
  };

  const renderOptions = ({ open, restoreButtonFocusOnClose }: TComboDropDownRenderProps) =>
    open && (
      <MemberOptions
        getUserDetails={getUserDetails}
        getIsUserSuspended={getIsUserSuspended}
        isRosterLoading={isRosterLoading}
        isRosterError={isRosterError}
        onRosterRetry={onRosterRetry}
        memberIds={memberIds}
        optionsClassName={optionsClassName}
        placement={placement}
        referenceElement={referenceElement}
        restoreButtonFocusOnClose={restoreButtonFocusOnClose}
        value={value}
      />
    );

  if (props.multiple) {
    return (
      <ComboDropDown<string, true> {...comboProps} multiple value={props.value} onChange={props.onChange}>
        {renderOptions}
      </ComboDropDown>
    );
  }

  return (
    <ComboDropDown<string | null> {...comboProps} multiple={false} value={props.value} onChange={props.onChange}>
      {renderOptions}
    </ComboDropDown>
  );
}
