/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { EIssueServiceType } from "@plane/types";

/** Choose the endpoint family from the canonical item, independent of the current list route. */
export function workItemServiceType(item: { is_epic?: boolean }) {
  return item.is_epic ? EIssueServiceType.EPICS : EIssueServiceType.ISSUES;
}
