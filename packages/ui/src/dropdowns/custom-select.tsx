/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Combobox } from "@headlessui/react";

import React, { createContext, useCallback, useContext, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { usePopper } from "react-popper";
import { useOutsideClickDetector } from "@plane/hooks";
import { CheckIcon, ChevronDownIcon } from "@plane/propel/icons";
// plane helpers
// hooks
import { useDropdownKeyDown } from "../hooks/use-dropdown-key-down";
// helpers
import { cn } from "../utils";
// types
import { isCustomButtonDisabled } from "./helper";
import type { ICustomSelectItemProps, ICustomSelectProps, TAsChildButtonProps } from "./helper";

const HeadlessComboboxButton = Combobox.Button as unknown as React.ComponentType<TAsChildButtonProps>;

// Context to share the close handler with option components
const DropdownContext = createContext<() => void>(() => {});

const CustomSelect = React.forwardRef<HTMLButtonElement, ICustomSelectProps>(
  function CustomSelect(props, forwardedRef) {
    const {
      customButtonClassName = "",
      buttonClassName = "",
      placement,
      children,
      className = "",
      customButton,
      customButtonAsChild = false,
      disabled = false,
      input = false,
      label,
      maxHeight = "md",
      noChevron = false,
      onChange,
      optionsClassName = "",
      value,
      tabIndex,
    } = props;
    // states
    const [referenceElement, setReferenceElement] = useState<HTMLButtonElement | null>(null);
    const [popperElement, setPopperElement] = useState<HTMLDivElement | null>(null);
    const [isOpen, setIsOpen] = useState(false);
    const customButtonDisabled = isCustomButtonDisabled(disabled, customButton);
    // refs
    const dropdownRef = useRef<HTMLDivElement | null>(null);

    const setButtonRef = useCallback(
      (node: HTMLButtonElement | null) => {
        setReferenceElement(node);
        if (typeof forwardedRef === "function") forwardedRef(node);
        else if (forwardedRef) forwardedRef.current = node;
      },
      [forwardedRef]
    );

    const { styles, attributes } = usePopper(referenceElement, popperElement, {
      placement: placement ?? "bottom-start",
    });

    const openDropdown = useCallback(() => {
      setIsOpen(true);
      if (referenceElement) referenceElement.focus();
    }, [referenceElement]);

    const closeDropdown = useCallback(() => setIsOpen(false), []);
    const handleKeyDown = useDropdownKeyDown(openDropdown, closeDropdown, isOpen);
    useOutsideClickDetector(dropdownRef, closeDropdown);

    const toggleDropdown = useCallback(() => {
      if (isOpen) closeDropdown();
      else openDropdown();
    }, [closeDropdown, isOpen, openDropdown]);

    const handleCustomButtonKeyDown = useCallback(
      (event: React.KeyboardEvent<HTMLButtonElement>) => {
        if (customButtonDisabled) return;
        if (event.key === "Enter" || event.key === " ") {
          // Let the native button click activate both this dropdown and Headless UI once.
          event.stopPropagation();
        } else if (event.key === "Escape" && isOpen) {
          event.stopPropagation();
          closeDropdown();
        } else if (!isOpen && ["ArrowDown", "ArrowUp"].includes(event.key)) {
          event.stopPropagation();
          openDropdown();
        }
      },
      [closeDropdown, customButtonDisabled, isOpen, openDropdown]
    );

    return (
      <DropdownContext.Provider value={closeDropdown}>
        <Combobox
          as="div"
          role="presentation"
          ref={dropdownRef}
          tabIndex={tabIndex}
          value={value}
          onChange={(val) => {
            onChange?.(val);
            closeDropdown();
          }}
          className={cn("relative flex-shrink-0 text-left", className)}
          onKeyDown={handleKeyDown}
          disabled={customButtonDisabled}
        >
          <>
            {customButton ? (
              customButtonAsChild ? (
                <HeadlessComboboxButton
                  as={React.Fragment}
                  ref={setButtonRef}
                  type="button"
                  className={customButtonClassName}
                  onClick={toggleDropdown}
                  onKeyDown={handleCustomButtonKeyDown}
                  disabled={customButtonDisabled}
                >
                  {customButton}
                </HeadlessComboboxButton>
              ) : (
                <Combobox.Button as={React.Fragment}>
                  <button
                    ref={setButtonRef}
                    type="button"
                    className={`flex items-center justify-between gap-1 rounded text-11 ${
                      disabled ? "cursor-not-allowed text-secondary" : "cursor-pointer hover:bg-layer-transparent-hover"
                    } ${customButtonClassName}`}
                    onClick={toggleDropdown}
                  >
                    {customButton}
                  </button>
                </Combobox.Button>
              )
            ) : (
              <Combobox.Button as={React.Fragment}>
                <button
                  ref={setButtonRef}
                  type="button"
                  className={cn(
                    "flex w-full items-center justify-between gap-1 rounded border border-strong",
                    {
                      "px-3 py-2 text-13": input,
                      "px-2 py-1 text-11": !input,
                      "cursor-not-allowed text-secondary": disabled,
                      "cursor-pointer hover:bg-layer-transparent-hover": !disabled,
                    },
                    buttonClassName
                  )}
                  onClick={toggleDropdown}
                >
                  {label}
                  {!noChevron && !disabled && <ChevronDownIcon className="h-3 w-3" aria-hidden="true" />}
                </button>
              </Combobox.Button>
            )}
          </>
          {isOpen &&
            createPortal(
              <Combobox.Options
                data-prevent-outside-click
                onKeyDown={(event) => {
                  if (event.key === "Escape") closeDropdown();
                }}
              >
                <div
                  className={cn(
                    "z-30 my-1 min-w-48 overflow-y-scroll rounded-md border-[0.5px] border-subtle-1 bg-surface-1 px-2 py-2.5 text-11 whitespace-nowrap focus:outline-none",
                    optionsClassName
                  )}
                  ref={setPopperElement}
                  style={styles.popper}
                  {...attributes.popper}
                >
                  <div
                    className={cn("space-y-1 overflow-y-scroll", {
                      "max-h-60": maxHeight === "lg",
                      "max-h-48": maxHeight === "md",
                      "max-h-36": maxHeight === "rg",
                      "max-h-28": maxHeight === "sm",
                    })}
                  >
                    {children}
                  </div>
                </div>
              </Combobox.Options>,
              document.body
            )}
        </Combobox>
      </DropdownContext.Provider>
    );
  }
);

CustomSelect.displayName = "CustomSelect";

function Option(props: ICustomSelectItemProps) {
  const { children, value, className } = props;
  const closeDropdown = useContext(DropdownContext);

  const handleClick = useCallback(() => {
    // Close dropdown for both new and already-selected options.
    // Use setTimeout to ensure HeadlessUI's onChange handler fires first for new selections.
    // For already-selected options, this ensures the dropdown closes since onChange won't fire.
    setTimeout(() => {
      closeDropdown();
    }, 0);
  }, [closeDropdown]);

  return (
    <Combobox.Option
      value={value}
      className={({ active }) =>
        cn(
          "flex cursor-pointer items-center justify-between gap-2 truncate rounded-sm px-1 py-1.5 text-secondary select-none",
          {
            "bg-layer-transparent-hover": active,
          },
          className
        )
      }
      onClick={handleClick}
    >
      {({ selected }) => (
        <div className="flex w-full items-center justify-between gap-2">
          {children}
          {selected && <CheckIcon className="h-3.5 w-3.5 flex-shrink-0" />}
        </div>
      )}
    </Combobox.Option>
  );
}

const CustomSelectWithOption = Object.assign(CustomSelect, { Option });

export { CustomSelectWithOption as CustomSelect };
