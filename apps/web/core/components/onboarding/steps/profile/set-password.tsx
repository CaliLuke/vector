/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { LockIcon, ChevronDownIcon } from "@plane/propel/icons";
import { AuthPasswordInput, AuthConfirmPasswordInput } from "@plane/ui";

type Props = {
  password: string;
  confirmPassword: string;
  onPasswordChange: (password: string) => void;
  onConfirmPasswordChange: (password: string) => void;
  disabled?: boolean;
};

export function SetPasswordRoot({
  password,
  confirmPassword,
  onPasswordChange,
  onConfirmPasswordChange,
  disabled = false,
}: Props) {
  const [isExpanded, setIsExpanded] = useState(false);
  return (
    <div className="flex flex-col overflow-hidden rounded-lg bg-surface-2">
      <button
        type="button"
        disabled={disabled}
        aria-expanded={isExpanded}
        onClick={() => setIsExpanded(!isExpanded)}
        className="flex items-center justify-between px-3 py-2 text-13 text-tertiary"
      >
        <span className="flex items-center gap-1">
          <LockIcon className="size-3" />
          <span className="font-medium">Set a password</span> (Optional)
        </span>
        <ChevronDownIcon className={`size-4 transition-transform ${isExpanded ? "rotate-180" : ""}`} />
      </button>
      {isExpanded && (
        <div className="flex flex-col gap-4 px-3 pb-3">
          <AuthPasswordInput
            id="password"
            name="password"
            value={password}
            onChange={(event) => onPasswordChange(event.target.value)}
            autoComplete="new-password"
            disabled={disabled}
          />
          <AuthConfirmPasswordInput
            id="confirm_password"
            name="confirm_password"
            password={password}
            value={confirmPassword}
            onChange={(event) => onConfirmPasswordChange(event.target.value)}
            autoComplete="new-password"
            disabled={disabled}
          />
        </div>
      )}
    </div>
  );
}
