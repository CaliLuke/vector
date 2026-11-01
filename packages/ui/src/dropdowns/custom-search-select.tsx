/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Combobox, Portal } from "@headlessui/react";
import { Info } from "lucide-react";
import React, { useCallback, useEffect, useEffectEvent, useRef, useState } from "react";
import { usePopper } from "react-popper";
import { useOutsideClickDetector } from "@plane/hooks";
import { CheckIcon, SearchIcon, ChevronDownIcon } from "@plane/propel/icons";
// plane imports
// local imports
import { Tooltip } from "@plane/propel/tooltip";
import { useDropdownButtonKeyDown, useDropdownKeyDown } from "../hooks/use-dropdown-key-down";
import { cn } from "../utils";
import { isCustomButtonDisabled, SyncComboboxOpenState, searchSelectOptionKey } from "./helper";
import type { ICustomSearchSelectProps } from "./helper";

// This picker mounts its search input only while open. Its button therefore owns
// the closed-state Tab stop (a standard Combobox leaves that to its input).
function SearchSelectButton({
  elementType = "button",
  triggerTabIndex = 0,
  tabIndex: _inputOwnedTabIndex,
  ...props
}: React.ComponentPropsWithRef<"button"> & { elementType?: React.ReactElement["type"]; triggerTabIndex?: number }) {
  return React.createElement(elementType, { ...props, tabIndex: triggerTabIndex });
}

export function CustomSearchSelect<T>(props: ICustomSearchSelectProps<T>) {
  const {
    customButtonClassName = "",
    buttonClassName = "",
    className = "",
    chevronClassName = "",
    customButton,
    customButtonAsChild,
    placement,
    disabled = false,
    footerOption,
    input = false,
    label,
    maxHeight = "md",
    multiple,
    noChevron = false,
    options,
    onOpen,
    onClose,
    optionsClassName = "",
    value,
    tabIndex,
    noResultsMessage = "No matches found",
    defaultOpen = false,
  } = props;
  const [query, setQuery] = useState("");
  // Portal children mount after their trigger's render. Focus at attachment so
  // pointer opening and keyboard opening both enter the mounted search field.
  const focusSearchInput = useCallback((element: HTMLInputElement | null) => element?.focus(), []);

  const [referenceElement, setReferenceElement] = useState<HTMLButtonElement | null>(null);
  // Position Options itself until Headless UI preserves refs on its immediate child.
  // https://github.com/tailwindlabs/headlessui/issues/3908
  const [popperElement, setPopperElement] = useState<HTMLElement | null>(null);
  const [isOpen, setIsOpen] = useState(defaultOpen);
  const customButtonDisabled = isCustomButtonDisabled(disabled, customButton);

  const notifyOpen = useEffectEvent(() => onOpen?.());

  useEffect(() => {
    if (isOpen) notifyOpen();
  }, [isOpen]);
  // refs
  const dropdownRef = useRef<HTMLDivElement | null>(null);

  const { styles, attributes } = usePopper(referenceElement, popperElement, {
    placement: placement ?? "bottom-start",
  });

  const filteredOptions =
    query === "" ? options : options?.filter((option) => option.query.toLowerCase().includes(query.toLowerCase()));

  const openDropdown = () => {
    setIsOpen(true);
    if (referenceElement) referenceElement.focus();
  };

  const closeDropdown = () => {
    setIsOpen(false);
  };

  const handleKeyDown = useDropdownKeyDown(openDropdown, closeDropdown, isOpen);
  useOutsideClickDetector(dropdownRef, closeDropdown);

  const handleButtonClick = (event: React.MouseEvent<HTMLButtonElement>, headlessOpen: boolean) => {
    // Headless UI handles pointer activation; only synthesize non-pointer clicks here.
    if (event.detail > 0) return;
    if (headlessOpen) closeDropdown();
    else openDropdown();
  };

  const handleCustomButtonKeyDown = useDropdownButtonKeyDown(customButtonDisabled, isOpen, openDropdown, closeDropdown);

  return (
    <Combobox<unknown, boolean, "div">
      as="div"
      role="presentation"
      ref={dropdownRef}
      immediate={defaultOpen}
      className={cn("relative flex-shrink-0 text-left", className)}
      onKeyDown={handleKeyDown}
      onClose={() => {
        // Selection unmounts the search input. Keep keyboard focus on the picker;
        // an outside click has already moved focus and must retain its target.
        if (popperElement?.contains(document.activeElement)) referenceElement?.focus();
        setIsOpen(false);
        onClose?.();
      }}
      value={value}
      multiple={multiple}
      disabled={customButtonDisabled}
      onChange={(nextValue) => {
        if (props.multiple) {
          if (!Array.isArray(nextValue)) return;
          const candidates = [...(options ?? []), ...(props.value ?? []).map((selected) => ({ value: selected }))];
          props.onChange(
            nextValue.flatMap((selected: unknown) => {
              const match = candidates.find((option) => Object.is(option.value, selected));
              return match ? [match.value] : [];
            })
          );
        } else if (nextValue !== null || value !== null) {
          const match = options?.find((option) => Object.is(option.value, nextValue));
          if (match) props.onChange(match.value);
          else if (nextValue === null) props.onChange(null);
        }
      }}
    >
      {({ open }) => (
        <>
          <SyncComboboxOpenState open={open} onOpenChange={setIsOpen} />
          {customButton ? (
            customButtonAsChild ? (
              <Combobox.Button
                as={React.Fragment}
                {...({
                  ref: setReferenceElement,
                  type: "button",
                  className: customButtonClassName,
                  onClick: (event) => handleButtonClick(event, open),
                  onKeyDown: handleCustomButtonKeyDown,
                  disabled: customButtonDisabled,
                } satisfies React.ComponentPropsWithRef<"button">)}
              >
                {React.isValidElement<React.ComponentPropsWithRef<"button">>(customButton) ? (
                  <SearchSelectButton
                    {...customButton.props}
                    elementType={customButton.type}
                    triggerTabIndex={customButton.props.tabIndex ?? tabIndex ?? 0}
                  />
                ) : (
                  customButton
                )}
              </Combobox.Button>
            ) : (
              <Combobox.Button as={React.Fragment}>
                <SearchSelectButton
                  triggerTabIndex={tabIndex ?? 0}
                  ref={setReferenceElement}
                  type="button"
                  className={cn(
                    "flex w-full items-center justify-between gap-1 text-11",
                    {
                      "cursor-not-allowed text-secondary": disabled,
                      "cursor-pointer hover:bg-layer-transparent-hover": !disabled,
                    },
                    customButtonClassName
                  )}
                  onClick={(event) => handleButtonClick(event, open)}
                >
                  {customButton}
                </SearchSelectButton>
              </Combobox.Button>
            )
          ) : (
            <Combobox.Button as={React.Fragment}>
              <SearchSelectButton
                triggerTabIndex={tabIndex ?? 0}
                ref={setReferenceElement}
                type="button"
                className={cn(
                  "flex w-full items-center justify-between gap-1 rounded-sm border-[0.5px] border-strong",
                  {
                    "px-3 py-2 text-13": input,
                    "px-2 py-1 text-11": !input,
                    "cursor-not-allowed text-secondary": disabled,
                    "cursor-pointer hover:bg-layer-transparent-hover": !disabled,
                  },
                  buttonClassName
                )}
                onClick={(event) => handleButtonClick(event, open)}
              >
                {label}
                {!noChevron && !disabled && (
                  <ChevronDownIcon className={cn("h-3 w-3 flex-shrink-0", chevronClassName)} aria-hidden="true" />
                )}
              </SearchSelectButton>
            </Combobox.Button>
          )}
          {(isOpen || open) && (
            <Portal>
              <Combobox.Options
                modal={false}
                data-prevent-outside-click
                static
                className={cn(
                  "z-30 my-1 min-w-48 overflow-y-scroll rounded-md border-[0.5px] border-subtle-1 bg-surface-1 py-2.5 text-11 whitespace-nowrap focus:outline-none",
                  optionsClassName
                )}
                ref={setPopperElement}
                style={styles.popper}
                {...attributes.popper}
              >
                <div className="mx-2 flex items-center gap-1.5 rounded-sm border border-subtle px-2">
                  <SearchIcon className="h-3.5 w-3.5 text-placeholder" strokeWidth={1.5} />
                  <Combobox.Input
                    className="w-full bg-transparent py-1 text-11 text-secondary placeholder:text-placeholder focus:outline-none"
                    ref={focusSearchInput}
                    value={query}
                    onChange={(e) => setQuery(e.target.value)}
                    onKeyDown={(event) => {
                      if (event.key === "Escape") {
                        referenceElement?.focus();
                      }
                    }}
                    placeholder="Search"
                  />
                </div>
                <div
                  className={cn("vertical-scrollbar mt-2 scrollbar-xs space-y-1 overflow-y-scroll px-2", {
                    "max-h-96": maxHeight === "2xl",
                    "max-h-80": maxHeight === "xl",
                    "max-h-60": maxHeight === "lg",
                    "max-h-48": maxHeight === "md",
                    "max-h-36": maxHeight === "rg",
                    "max-h-28": maxHeight === "sm",
                  })}
                >
                  {filteredOptions ? (
                    filteredOptions.length > 0 ? (
                      filteredOptions.map((option) => (
                        <Combobox.Option
                          key={searchSelectOptionKey(option.value, option.query)}
                          value={option.value}
                          className={({ active }) =>
                            cn(
                              "flex w-full cursor-pointer items-center justify-between gap-2 truncate rounded-sm px-1 py-1.5 select-none",
                              {
                                "bg-layer-transparent-hover": active,
                                "cursor-not-allowed text-placeholder opacity-60": option.disabled,
                              }
                            )
                          }
                          onClick={() => {
                            if (!multiple) closeDropdown();
                          }}
                          disabled={option.disabled}
                        >
                          {({ selected }) => (
                            <>
                              <span className="flex-grow truncate">{option.content}</span>
                              {selected && <CheckIcon className="h-3.5 w-3.5 flex-shrink-0" />}
                              {option.tooltip && (
                                <>
                                  {typeof option.tooltip === "string" ? (
                                    <Tooltip tooltipContent={option.tooltip}>
                                      <Info className="h-3.5 w-3.5 flex-shrink-0 cursor-pointer text-secondary" />
                                    </Tooltip>
                                  ) : (
                                    option.tooltip
                                  )}
                                </>
                              )}
                            </>
                          )}
                        </Combobox.Option>
                      ))
                    ) : (
                      <p className="px-1.5 py-1 text-placeholder italic">{noResultsMessage}</p>
                    )
                  ) : (
                    <p className="px-1.5 py-1 text-placeholder italic">Loading...</p>
                  )}
                </div>
                {footerOption}
              </Combobox.Options>
            </Portal>
          )}
        </>
      )}
    </Combobox>
  );
}
