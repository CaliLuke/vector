/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Popover } from "@plane/propel/popover";
import type { TIssue } from "@plane/types";
import { WorkItemPreviewCard } from "../preview-card";

export function IssuePreviewPopoverPanel({ issue }: { issue: TIssue }) {
  return (
    <Popover.Panel side="bottom" align="start">
      {issue.project_id ? (
        <WorkItemPreviewCard
          projectId={issue.project_id}
          stateDetails={{ id: issue.state_id ?? undefined }}
          workItem={issue}
        />
      ) : null}
    </Popover.Panel>
  );
}
