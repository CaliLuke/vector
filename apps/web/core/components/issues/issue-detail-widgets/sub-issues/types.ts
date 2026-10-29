/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TIssue, TIssueServiceType } from "@plane/types";

export type SubIssueAction = {
  kind: "update" | "delete";
  issue: TIssue;
  parent: { projectId: string; issueId: string; serviceType: TIssueServiceType };
};
