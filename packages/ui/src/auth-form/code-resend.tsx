/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ReactNode } from "react";
import { CircleCheck } from "lucide-react";

type Props = { hint: ReactNode; disabled: boolean; onResend: () => void; children: ReactNode };

export function AuthCodeResend({ hint, disabled, onResend, children }: Props) {
  return (
    <div className="flex w-full items-center justify-between px-1 pt-1 text-11">
      <p className="flex items-center gap-1 font-medium text-success-primary">
        <CircleCheck height={12} width={12} />
        {hint}
      </p>
      <button
        type="button"
        onClick={onResend}
        disabled={disabled}
        className={disabled ? "text-placeholder" : "font-medium text-accent-secondary hover:text-accent-secondary"}
      >
        {children}
      </button>
    </div>
  );
}
