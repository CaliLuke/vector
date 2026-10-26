/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IIssueLabel } from "@plane/types";

export function LabelOptionContent({ label }: { label: Pick<IIssueLabel, "name" | "color"> }) {
  return (
    <div className="flex items-center justify-start gap-2 overflow-hidden">
      <span className="h-2.5 w-2.5 flex-shrink-0 rounded-full" style={{ backgroundColor: label.color }} />
      <div className="line-clamp-1 inline-block truncate">{label.name}</div>
    </div>
  );
}
