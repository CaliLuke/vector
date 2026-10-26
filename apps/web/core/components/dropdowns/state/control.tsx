/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ReactNode } from "react";
// plane imports
import { useTranslation } from "@plane/i18n";
import { ChevronDownIcon } from "@plane/propel/icons";
import { Spinner } from "@plane/ui";
import { cn } from "@plane/utils/common";
// components
import { DropdownButton } from "@/components/dropdowns/buttons";
import { BUTTON_VARIANTS_WITH_TEXT } from "@/components/dropdowns/constants";
import type { StateDropdownProps } from "../state/types";
import { useSearchableDropdown } from "../searchable-options";
// hooks
// plane web imports
import { DropdownOption } from "../dropdown-option";

type StateRecord = { id: string; name: string; color: string; default: boolean };

export type StateDropdownControlProps<TState extends StateRecord> = StateDropdownProps & {
  getStateById: (stateId: string | null | undefined) => TState | undefined;
  isInitializing?: boolean;
  stateIds: string[];
};

type Props<TState extends StateRecord> = StateDropdownControlProps<TState> & {
  renderIcon: (state: TState | undefined, className: string) => ReactNode;
  fallbackToFirstState?: boolean;
};

export function StateDropdownControl<TState extends StateRecord>(props: Props<TState>) {
  const {
    button,
    buttonClassName,
    buttonContainerClassName,
    buttonVariant,
    className,
    disabled,
    dropdownArrow,
    dropdownArrowClassName,
    getStateById,
    renderIcon,
    fallbackToFirstState = false,
    hideIcon,
    iconSize = "size-4",
    isInitializing,
    onChange,
    onClose,
    placement,
    renderByDefault = true,
    showDefaultState = true,
    showTooltip = false,
    stateIds,
    tabIndex,
    value,
  } = props;
  const { t } = useTranslation();
  const dropdown = useSearchableDropdown(placement, onClose);
  const { query, setReferenceElement } = dropdown;
  const statesList = stateIds.map((stateId) => getStateById(stateId)).filter((state) => !!state);
  const defaultState = statesList.find((state) => state.default) ?? (fallbackToFirstState ? statesList[0] : undefined);
  const stateValue = value ? value : showDefaultState ? defaultState?.id : undefined;
  // derived values
  const options = statesList.map((state) => ({
    value: state.id,
    query: state.name,
    content: (
      <div className="flex items-center gap-2">
        {renderIcon(state, cn("flex-shrink-0", iconSize))}
        <span className="flex-grow truncate text-left">{state.name}</span>
      </div>
    ),
  }));

  const filteredOptions =
    query === "" ? options : options.filter((o) => o.query.toLowerCase().includes(query.toLowerCase()));

  const selectedState = stateValue ? getStateById(stateValue) : undefined;

  const dropdownOnChange = (val: string | string[] | null) => {
    if (typeof val === "string") onChange(val);
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
        tabIndex={tabIndex}
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
      >
        <DropdownButton
          className={buttonClassName}
          isActive={open}
          tooltipHeading={t("state")}
          tooltipContent={selectedState?.name ?? t("state")}
          showTooltip={showTooltip}
          variant={buttonVariant}
          renderToolTipByDefault={renderByDefault}
        >
          {isInitializing ? (
            <Spinner className="h-3.5 w-3.5" />
          ) : (
            <>
              {!hideIcon && renderIcon(selectedState, cn("flex-shrink-0", iconSize))}
              {BUTTON_VARIANTS_WITH_TEXT.includes(buttonVariant) && (
                <span className="flex-grow truncate text-left">{selectedState?.name ?? t("state")}</span>
              )}
              {dropdownArrow && (
                <ChevronDownIcon
                  className={cn("h-2.5 w-2.5 flex-shrink-0", dropdownArrowClassName)}
                  aria-hidden="true"
                />
              )}
            </>
          )}
        </DropdownButton>
      </button>
    );

  return dropdown.render(
    {
      className: cn("h-full", className),
      // Keep Headless UI's Combobox controlled while the state catalogue loads.
      value: stateValue ?? null,
      onChange: dropdownOnChange,
      disabled,
      button: comboButton,
      renderByDefault,
    },
    filteredOptions.length > 0 ? (
      filteredOptions.map((option) => (
        <DropdownOption key={option.value} value={option.value}>
          {option.content}
        </DropdownOption>
      ))
    ) : (
      <p className="px-1.5 py-1 text-placeholder italic">{t("no_matching_results")}</p>
    ),
    t("common.search.label")
  );
}
