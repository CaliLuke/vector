/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Combobox } from "@headlessui/react";
import type { ComboboxInputProps, ComboboxProps } from "@headlessui/react";
import type {
  ChangeEvent,
  Dispatch,
  ElementType,
  KeyboardEvent,
  ReactNode,
  Ref,
  RefObject,
  SetStateAction,
} from "react";
import { Fragment, useEffect, useRef, useState } from "react";

export type TComboDropDownRenderProps = {
  open: boolean;
  restoreButtonFocusOnClose: () => void;
};

type Renderable<T> = ReactNode | ((props: T) => ReactNode);

type ComboSearchInputProps = Omit<ComboboxInputProps, "value" | "onChange" | "onKeyDown" | "displayValue"> & {
  ref?: Ref<HTMLInputElement>;
  query: string;
  onQueryChange: (query: string) => void;
  onEscapeClose: () => void;
  onKeyDown?: (event: KeyboardEvent<HTMLInputElement>) => void;
};

function ComboSearchInput({ query, onQueryChange, onEscapeClose, onKeyDown, ...inputProps }: ComboSearchInputProps) {
  const handleKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "Escape") {
      if (query !== "") {
        event.preventDefault();
        event.stopPropagation();
        onQueryChange("");
        return;
      }
      onEscapeClose();
    }
    onKeyDown?.(event);
  };

  const handleChange = (event: ChangeEvent<HTMLInputElement>) => onQueryChange(event.target.value);

  return <Combobox.Input {...inputProps} value={query} onChange={handleChange} onKeyDown={handleKeyDown} />;
}

type Props<TValue, TMultiple extends boolean | undefined> = Omit<
  ComboboxProps<TValue, TMultiple, "div">,
  "as" | "children" | "onChange" | "onClose" | "value"
> & {
  as?: ElementType;
  ref?: Ref<HTMLElement>;
  value?: ComboboxProps<TValue, TMultiple, "div">["value"];
  renderByDefault?: boolean;
  onOpen?: () => void;
  onClose?: () => void;
  button: Renderable<TComboDropDownRenderProps>;
  children: Renderable<TComboDropDownRenderProps>;
} & Pick<ComboboxProps<TValue, TMultiple>, "onChange">;

function OpenStateObserver({
  open,
  onOpen,
  shouldRestoreButtonFocus,
  clearRestoreButtonFocus,
  buttonRef,
}: {
  open: boolean;
  onOpen?: () => void;
  shouldRestoreButtonFocus: boolean;
  clearRestoreButtonFocus: Dispatch<SetStateAction<boolean>>;
  buttonRef: RefObject<HTMLButtonElement | null>;
}) {
  const wasOpen = useRef(open);

  useEffect(() => {
    if (open && !wasOpen.current) onOpen?.();
    if (!open && wasOpen.current) {
      if (shouldRestoreButtonFocus) buttonRef.current?.focus();
      clearRestoreButtonFocus(false);
    }
    wasOpen.current = open;
  }, [buttonRef, clearRestoreButtonFocus, onOpen, open, shouldRestoreButtonFocus]);

  return null;
}

function ComboDropDown<TValue, TMultiple extends boolean | undefined = false>(props: Props<TValue, TMultiple>) {
  const { button, children, onOpen, onClose, ref, renderByDefault = true, ...comboboxProps } = props;
  const buttonRef = useRef<HTMLButtonElement>(null);
  const [shouldRestoreButtonFocus, setShouldRestoreButtonFocus] = useState(false);

  const renderProps = (open: boolean): TComboDropDownRenderProps => ({
    open,
    restoreButtonFocusOnClose: () => {
      setShouldRestoreButtonFocus(true);
    },
  });

  return (
    <Combobox {...comboboxProps} ref={ref} onClose={onClose}>
      {({ open }) => (
        <>
          <Combobox.Button as={Fragment} ref={buttonRef}>
            {typeof button === "function" ? button(renderProps(open)) : button}
          </Combobox.Button>
          <OpenStateObserver
            open={open}
            onOpen={onOpen}
            shouldRestoreButtonFocus={shouldRestoreButtonFocus}
            clearRestoreButtonFocus={setShouldRestoreButtonFocus}
            buttonRef={buttonRef}
          />
          {(renderByDefault || open) && (typeof children === "function" ? children(renderProps(open)) : children)}
        </>
      )}
    </Combobox>
  );
}

const ComboOptions = Combobox.Options;
const ComboOption = Combobox.Option;
const ComboInput = Combobox.Input;

export { ComboDropDown, ComboOptions, ComboOption, ComboInput, ComboSearchInput };
