/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TIssueRelation } from "@plane/types";

export function countVisibleRelations(
  relations: TIssueRelation | undefined,
  options: Partial<Record<keyof TIssueRelation, unknown>>
) {
  if (!relations) return 0;
  const relationKeys: (keyof TIssueRelation)[] = ["blocking", "blocked_by", "duplicate", "relates_to"];
  return relationKeys.reduce(
    (count, relationKey) => count + (options[relationKey] ? (relations[relationKey]?.length ?? 0) : 0),
    0
  );
}
