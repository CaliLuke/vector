/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useEffect, useId, useRef, useState } from "react";
import { useOutsideClickDetector } from "@plane/hooks";
import { CloseIcon, SearchIcon } from "@plane/propel/icons";
import { IconButton } from "@plane/propel/icon-button";
import { cn } from "@plane/utils/common";

type Props = {
  value: string;
  onChange: (value: string) => void;
  placeholder: string;
  initiallyOpen?: boolean;
  openWhenValueExists?: boolean;
  clearOnClose?: boolean;
  className?: string;
  triggerClassName?: string;
  fieldClassName?: string;
  expandedFieldClassName?: string;
  inputClassName?: string;
  searchIconClassName?: string;
};

export function ControlledExpandableSearch({
  value,
  onChange,
  placeholder,
  initiallyOpen = false,
  openWhenValueExists = false,
  clearOnClose = true,
  className,
  triggerClassName,
  fieldClassName,
  expandedFieldClassName,
  inputClassName,
  searchIconClassName,
}: Props) {
  const [isManuallyExpanded, setIsManuallyExpanded] = useState(initiallyOpen);
  const isExpanded = isManuallyExpanded || (openWhenValueExists && value.trim() !== "");
  const inputId = useId();
  const closeLabel = clearOnClose ? `Clear ${placeholder}` : `Close ${placeholder}`;
  const rootRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const focusInput = useRef(false);
  const restoreTriggerFocus = useRef(false);

  useOutsideClickDetector(rootRef, () => {
    if (isExpanded && value.trim() === "") {
      restoreTriggerFocus.current = false;
      setIsManuallyExpanded(false);
    }
  });

  useEffect(() => {
    if (isExpanded && focusInput.current) {
      focusInput.current = false;
      inputRef.current?.focus();
    } else if (restoreTriggerFocus.current) {
      restoreTriggerFocus.current = false;
      triggerRef.current?.focus();
    }
  }, [isExpanded]);

  const close = (shouldRestoreFocus: boolean) => {
    restoreTriggerFocus.current = shouldRestoreFocus;
    setIsManuallyExpanded(false);
  };

  return (
    <div ref={rootRef} className={className}>
      <IconButton
        ref={triggerRef}
        variant="ghost"
        size="lg"
        className={triggerClassName}
        icon={SearchIcon}
        aria-label={placeholder}
        aria-expanded={isExpanded}
        aria-controls={inputId}
        hidden={isExpanded}
        onClick={() => {
          focusInput.current = true;
          setIsManuallyExpanded(true);
        }}
      />
      <div
        className={cn(
          "flex w-0 items-center justify-start overflow-hidden rounded-md border border-transparent text-placeholder opacity-0 transition-[width] ease-linear",
          fieldClassName,
          isExpanded && expandedFieldClassName,
          isExpanded && "opacity-100"
        )}
      >
        <SearchIcon className={searchIconClassName} aria-hidden="true" />
        {isExpanded && (
          <input
            id={inputId}
            ref={inputRef}
            className={cn(
              "w-full max-w-[234px] border-none bg-transparent text-13 text-primary placeholder:text-placeholder focus:outline-none",
              inputClassName
            )}
            aria-label={placeholder}
            placeholder={placeholder}
            value={value}
            onChange={(event) => onChange(event.target.value)}
            onKeyDown={(event) => {
              if (event.key !== "Escape") return;
              if (value.trim() !== "") {
                onChange("");
                if (openWhenValueExists) setIsManuallyExpanded(true);
              } else {
                close(true);
              }
            }}
          />
        )}
        {isExpanded && (
          <button
            type="button"
            className="grid place-items-center"
            aria-label={closeLabel}
            onClick={() => {
              if (clearOnClose) onChange("");
              close(true);
            }}
          >
            <CloseIcon className="h-3 w-3" aria-hidden="true" />
          </button>
        )}
      </div>
    </div>
  );
}
