/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";
import type { ICustomSearchSelectOption } from "@plane/types";

type Placement =
  | "top"
  | "top-start"
  | "top-end"
  | "bottom"
  | "bottom-start"
  | "bottom-end"
  | "left"
  | "left-start"
  | "left-end"
  | "right"
  | "right-start"
  | "right-end";

interface IDropdownProps {
  customButtonClassName?: string;
  customButtonTabIndex?: number;
  buttonClassName?: string;
  className?: string;
  customButton?: React.ReactNode;
  /** Render a custom interactive element as the dropdown trigger instead of nesting it inside a button. */
  customButtonAsChild?: boolean;
  disabled?: boolean;
  input?: boolean;
  label?: string | React.ReactNode;
  maxHeight?: "sm" | "rg" | "md" | "lg" | "xl" | "2xl";
  noChevron?: boolean;
  chevronClassName?: string;
  onOpen?: () => void;
  optionsClassName?: string;
  placement?: Placement;
  tabIndex?: number;
  useCaptureForOutsideClick?: boolean;
  defaultOpen?: boolean;
}

export function getDropdownEventTarget(event: Event): Element | null {
  const target = event.target;
  if (target instanceof Element) return target;
  return target instanceof Node ? target.parentElement : null;
}

export function isCustomButtonDisabled(disabled: boolean, customButton: React.ReactNode): boolean {
  const childDisabled = React.isValidElement<{ disabled?: boolean }>(customButton)
    ? customButton.props.disabled
    : false;
  return disabled || childDisabled === true;
}

export function SyncComboboxOpenState({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const previousOpen = React.useRef(open);

  React.useEffect(() => {
    if (previousOpen.current === open) return;
    previousOpen.current = open;
    onOpenChange(open);
  }, [onOpenChange, open]);

  return null;
}

export interface ICustomMenuDropdownProps extends IDropdownProps {
  children: React.ReactNode;
  ellipsis?: boolean;
  noBorder?: boolean;
  verticalEllipsis?: boolean;
  menuButtonOnClick?: () => void;
  menuItemsClassName?: string;
  onMenuClose?: () => void;
  closeOnSelect?: boolean;
  portalElement?: Element | null;
  openOnHover?: boolean;
  ariaLabel?: string;
}

export interface ICustomSelectProps<T> extends IDropdownProps {
  children: React.ReactNode;
  value: T | null | undefined;
  onChange: (value: T | null) => void;
  ref?: React.Ref<HTMLButtonElement>;
}

export type ICustomSearchSelectProps<T> = IDropdownProps & {
  footerOption?: React.ReactNode;
  onClose?: () => void;
  noResultsMessage?: string;
  options?: ICustomSearchSelectOption<T>[];
} & (
    | { multiple?: false; value: T | null | undefined; onChange: (value: T | null) => void }
    | { multiple: true; value: T[] | null; onChange: (value: T[]) => void }
  );

export interface ICustomMenuItemProps {
  children: React.ReactNode;
  disabled?: boolean;
  onClick?: React.MouseEventHandler<HTMLButtonElement>;
  className?: string;
}

export interface ICustomSelectItemProps {
  children: React.ReactNode;
  value: unknown;
  className?: string;
}

// Submenu interfaces
export interface ICustomSubMenuProps {
  children: React.ReactNode;
  trigger: React.ReactNode;
  disabled?: boolean;
  className?: string;
  contentClassName?: string;
  placement?: Placement;
}

export interface ICustomSubMenuTriggerProps {
  children: React.ReactNode;
  disabled?: boolean;
  className?: string;
}

export interface ICustomSubMenuContentProps {
  children: React.ReactNode;
  className?: string;
  placement?: Placement;
  sideOffset?: number;
  alignOffset?: number;
}

export function searchSelectOptionKey(value: unknown, query: string): React.Key {
  return typeof value === "string" || typeof value === "number" || typeof value === "bigint" ? value : query;
}
