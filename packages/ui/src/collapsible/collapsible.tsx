/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Disclosure, Transition } from "@headlessui/react";
import React, { useState, useEffect, useCallback, useId } from "react";

export type TCollapsibleProps = {
  title: string | React.ReactNode;
  titleActionElement?: React.ReactNode;
  children: React.ReactNode;
  buttonRef?: React.RefObject<HTMLButtonElement>;
  className?: string;
  buttonClassName?: string;
  isOpen?: boolean;
  onToggle?: () => void;
  defaultOpen?: boolean;
};

export function Collapsible(props: TCollapsibleProps) {
  const { title, titleActionElement, children, buttonRef, className, buttonClassName, isOpen, onToggle, defaultOpen } =
    props;
  // state
  const [localIsOpen, setLocalIsOpen] = useState<boolean>(isOpen ?? defaultOpen ?? false);
  const isDisclosureOpen = isOpen ?? localIsOpen;
  const panelId = useId();

  useEffect(() => {
    if (isOpen !== undefined) {
      setLocalIsOpen(isOpen);
    }
  }, [isOpen]);

  // handlers
  const handleOnClick = useCallback(() => {
    if (isOpen !== undefined) {
      if (onToggle) onToggle();
    } else {
      setLocalIsOpen((prev) => !prev);
    }
  }, [isOpen, onToggle]);

  const titleButton = (
    <button
      ref={buttonRef}
      type="button"
      className={titleActionElement ? `min-w-0 flex-1 ${buttonClassName ?? ""}` : buttonClassName}
      aria-expanded={isDisclosureOpen}
      aria-controls={panelId}
      onClick={handleOnClick}
    >
      {title}
    </button>
  );

  // Keep the Disclosure provider for nested Disclosure.Panel consumers; Transition supplies their effective open state.
  return (
    <Disclosure as="div" className={className} defaultOpen={isOpen ?? defaultOpen}>
      {titleActionElement ? (
        <div className="flex w-full items-center">
          {titleButton}
          {isDisclosureOpen && titleActionElement}
        </div>
      ) : (
        titleButton
      )}
      <Transition
        show={isDisclosureOpen}
        enter="transition-all duration-300 ease-in-out"
        enterFrom="grid-rows-[0fr] opacity-0"
        enterTo="grid-rows-[1fr] opacity-100"
        leave="transition-all duration-300 ease-in-out"
        leaveFrom="grid-rows-[1fr] opacity-100"
        leaveTo="grid-rows-[0fr] opacity-0"
        className="grid overflow-hidden"
      >
        <div id={panelId} className="min-h-0">
          {children}
        </div>
      </Transition>
    </Disclosure>
  );
}
