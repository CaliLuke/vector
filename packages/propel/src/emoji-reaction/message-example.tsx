/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import type { ReactNode } from "react";

export function ReactionMessageExample({ message, children }: { message: string; children: ReactNode }) {
  return (
    <div className="max-w-md space-y-3 rounded-lg border border-subtle p-4">
      <div className="flex items-start gap-3">
        <div className="flex h-8 w-8 items-center justify-center rounded-full bg-accent-primary text-13 text-on-color">
          AB
        </div>
        <div className="flex-1">
          <div className="text-13 font-medium">Alice Brown</div>
          <div className="mt-1 text-13 text-tertiary">{message}</div>
        </div>
      </div>
      {children}
    </div>
  );
}
