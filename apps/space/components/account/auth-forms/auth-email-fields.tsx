/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { XCircle } from "lucide-react";
import { Input } from "@plane/ui";

type Props = { email: string; nextPath?: string; onClear: () => void };

export function AuthEmailFields({ email, nextPath, onClear }: Props) {
  return (
    <>
      <input type="hidden" value={email} name="email" />
      <input type="hidden" value={nextPath ?? ""} name="next_path" />
      <div className="space-y-1">
        <label className="text-13 font-medium text-tertiary" htmlFor="email">
          Email
        </label>
        <div className="relative flex items-center rounded-md border border-subtle bg-surface-1">
          <Input
            id="email"
            type="email"
            value={email}
            placeholder="name@company.com"
            className="h-10 w-full border-0 disable-autofill-style placeholder:text-placeholder"
            disabled
          />
          {email && (
            <button
              type="button"
              aria-label="Change email"
              onClick={onClear}
              className="absolute right-3 text-placeholder"
            >
              <XCircle className="h-5 w-5" />
            </button>
          )}
        </div>
      </div>
    </>
  );
}
