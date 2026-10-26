/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { useState } from "react";
import { AuthPasswordInput } from "./auth-password-input";

type TAuthConfirmPasswordInputProps = React.InputHTMLAttributes<HTMLInputElement> & {
  password: string;
  label?: string;
  error?: string;
  showPasswordToggle?: boolean;
  containerClassName?: string;
  labelClassName?: string;
  errorClassName?: string;
};

export function AuthConfirmPasswordInput({
  password,
  label = "Confirm Password",
  error,
  showPasswordToggle = true,
  containerClassName = "",
  value = "",
  ...props
}: TAuthConfirmPasswordInputProps) {
  const [isFocused, setIsFocused] = useState(false);

  const confirmPassword = value as string;
  const passwordsMatch = password === confirmPassword && password.length > 0;
  const showMatchError =
    confirmPassword.length > 0 && !passwordsMatch && (!isFocused || confirmPassword.length >= password.length);

  const getError = () => {
    if (error) return error;
    if (showMatchError) return "Passwords don't match";
    return "";
  };

  return (
    <div className="space-y-2">
      <AuthPasswordInput
        {...props}
        containerClassName={containerClassName}
        showPasswordStrength={false}
        onFocus={(event) => {
          setIsFocused(true);
          props.onFocus?.(event);
        }}
        onBlur={(event) => {
          setIsFocused(false);
          props.onBlur?.(event);
        }}
        label={label}
        error={getError()}
        showPasswordToggle={showPasswordToggle}
        value={value}
      />
      {confirmPassword && passwordsMatch && <p className="text-13 text-success-primary">Passwords match</p>}
    </div>
  );
}
