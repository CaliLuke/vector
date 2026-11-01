/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ReactNode } from "react";

export function ExampleGroup({ title, children }: { title: string; children: ReactNode }) {
  return (
    <div className="space-y-4">
      <div className="space-y-2">
        <h3 className="text-16 font-semibold">{title}</h3>
        <div className="flex flex-wrap gap-2">{children}</div>
      </div>
    </div>
  );
}

export function ExampleSizes({ children }: { children: ReactNode }) {
  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2">{children}</div>
    </div>
  );
}
