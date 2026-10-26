/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { useState } from "react";
import { cn } from "@plane/utils/common";
import { PasswordStrengthIndicator } from "../form-fields/password/indicator";
import { AuthInput } from "./auth-input";

type TAuthPasswordInputProps = React.InputHTMLAttributes<HTMLInputElement> & {
  label?: string;
  error?: string;
  showPasswordStrength?: boolean;
  showPasswordToggle?: boolean;
  containerClassName?: string;
  errorClassName?: string;
};

export function AuthPasswordInput({
  label = "Password",
  error,
  showPasswordStrength = true,
  showPasswordToggle = true,
  containerClassName = "",
  value = "",
  ...props
}: TAuthPasswordInputProps) {
  const [isFocused, setIsFocused] = useState(false);

  return (
    <div
      className={cn("space-y-2", containerClassName)}
      onFocus={() => setIsFocused(true)}
      onBlur={() => setIsFocused(false)}
    >
      <AuthInput
        {...props}
        type="password"
        label={label}
        error={error}
        showPasswordToggle={showPasswordToggle}
        value={value}
      />
      {showPasswordStrength && value && isFocused && (
        <PasswordStrengthIndicator password={value as string} showCriteria />
      )}
    </div>
  );
}
