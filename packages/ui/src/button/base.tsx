/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import * as React from "react";
import { cn } from "../utils/classname";

export interface ButtonBaseProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  loading?: boolean;
  appendIcon?: React.ReactElement<{ strokeWidth?: number }> | null;
  prependIcon?: React.ReactElement<{ strokeWidth?: number }> | null;
  children: React.ReactNode;
}

type ButtonStyleConfig<Variant extends string, Size extends string> = {
  displayName: string;
  defaultVariant: Variant;
  defaultSize: Size;
  getStyle: (variant: Variant, size: Size, disabled: boolean) => string;
  getIconStyle: (size: Size) => string;
};

export function createButton<Variant extends string, Size extends string>(config: ButtonStyleConfig<Variant, Size>) {
  const StyledButton = React.forwardRef<HTMLButtonElement, ButtonBaseProps & { variant?: Variant; size?: Size }>(
    function StyledButton(props, ref) {
      const {
        variant = config.defaultVariant,
        size = config.defaultSize,
        className = "",
        type = "button",
        loading = false,
        disabled = false,
        prependIcon = null,
        appendIcon = null,
        children,
        ...rest
      } = props;
      const iconStyle = config.getIconStyle(size);
      return (
        <button
          ref={ref}
          type={type}
          className={cn(config.getStyle(variant, size, disabled || loading), className)}
          disabled={disabled || loading}
          {...rest}
        >
          {prependIcon && <div className={iconStyle}>{React.cloneElement(prependIcon, { strokeWidth: 2 })}</div>}
          {children}
          {appendIcon && <div className={iconStyle}>{React.cloneElement(appendIcon, { strokeWidth: 2 })}</div>}
        </button>
      );
    }
  );
  StyledButton.displayName = config.displayName;
  return StyledButton;
}
