/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { useRef, useState } from "react";
import { usePopper } from "react-popper";
import { usePlatformOS } from "@/hooks/use-platform-os";
import type { TDropdownProps } from "./types";
import type { CSSProperties, ReactNode, Ref } from "react";
import { Combobox } from "@headlessui/react";
import { SearchIcon } from "@plane/propel/icons";
import { ComboDropDown, ComboSearchInput } from "@plane/ui";
import type { TComboDropDownRenderProps } from "@plane/ui";

/** Shared searchable popper shell; callers own the option records and selection behavior. */
function SearchableDropdownOptions({
  inputRef,
  popperRef,
  style,
  attributes,
  query,
  onQueryChange,
  onEscapeClose,
  placeholder,
  children,
}: {
  inputRef: Ref<HTMLInputElement>;
  popperRef: Ref<HTMLElement>;
  style: CSSProperties | undefined;
  attributes: Record<string, string> | undefined;
  query: string;
  onQueryChange: (value: string) => void;
  onEscapeClose: () => void;
  placeholder: string;
  children: ReactNode;
}) {
  return (
    <Combobox.Options
      modal={false}
      className="z-10 my-1 w-48 rounded-sm border-[0.5px] border-strong bg-surface-1 px-2 py-2.5 text-11 shadow-raised-200 focus:outline-none"
      ref={popperRef}
      style={style}
      {...attributes}
    >
      <div className="flex items-center gap-1.5 rounded-sm border border-subtle bg-surface-2 px-2">
        <SearchIcon className="h-3.5 w-3.5 text-placeholder" strokeWidth={1.5} />
        <ComboSearchInput
          ref={inputRef}
          className="w-full bg-transparent py-1 text-11 text-secondary placeholder:text-placeholder focus:outline-none"
          query={query}
          onQueryChange={onQueryChange}
          onEscapeClose={onEscapeClose}
          placeholder={placeholder}
        />
      </div>
      <div className="mt-2 max-h-48 space-y-1 overflow-y-scroll">{children}</div>
    </Combobox.Options>
  );
}

/** One owner for dropdown search, focus, reset and popper lifecycle across property controls. */
export function useSearchableDropdown(
  placement: TDropdownProps["placement"],
  onClose?: () => void,
  onOpen?: () => void
) {
  const [query, setQuery] = useState("");
  const inputRef = useRef<HTMLInputElement | null>(null);
  const [referenceElement, setReferenceElement] = useState<HTMLButtonElement | null>(null);
  const [popperElement, setPopperElement] = useState<HTMLElement | null>(null);
  const { isMobile } = usePlatformOS();
  const { styles, attributes } = usePopper(referenceElement, popperElement, {
    placement: placement ?? "bottom-start",
    modifiers: [{ name: "preventOverflow", options: { padding: 12 } }],
  });
  const handleOpen = () => {
    if (!isMobile) inputRef.current?.focus();
    onOpen?.();
  };
  const handleClose = () => {
    setQuery("");
    onClose?.();
  };
  const renderOptions =
    (children: ReactNode, placeholder: string) =>
    ({ open, restoreButtonFocusOnClose }: TComboDropDownRenderProps) =>
      open ? (
        <SearchableDropdownOptions
          inputRef={inputRef}
          popperRef={setPopperElement}
          style={styles.popper}
          attributes={attributes.popper}
          query={query}
          onQueryChange={setQuery}
          onEscapeClose={restoreButtonFocusOnClose}
          placeholder={placeholder}
        >
          {children}
        </SearchableDropdownOptions>
      ) : null;
  return {
    query,
    setReferenceElement,
    onOpen: handleOpen,
    onClose: handleClose,
    renderOptions,
    render: (
      props: {
        value: string | null | undefined;
        onChange: (value: string | string[] | null) => void;
        button: ReactNode | ((state: { open: boolean }) => ReactNode);
        disabled?: boolean;
        className?: string;
        tabIndex?: number;
        renderByDefault?: boolean;
      },
      children: ReactNode,
      placeholder: string
    ) => (
      <ComboDropDown as="div" {...props} onOpen={handleOpen} onClose={handleClose}>
        {renderOptions(children, placeholder)}
      </ComboDropDown>
    ),
  };
}
