/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TInboxIssue } from "@plane/types";
// local imports
import { InboxIssueListItem } from "./inbox-list-item";

export type InboxIssueListProps = {
  workspaceSlug: string;
  projectId: string;
  projectIdentifier?: string;
  inboxIssues: TInboxIssue[];
  setIsMobileSidebar: (value: boolean) => void;
};

export const InboxIssueList = function InboxIssueList(props: InboxIssueListProps) {
  const { workspaceSlug, projectId, projectIdentifier, inboxIssues, setIsMobileSidebar } = props;

  return (
    <>
      {inboxIssues.map((inboxIssue) => (
        <InboxIssueListItem
          key={inboxIssue.issue.id}
          setIsMobileSidebar={setIsMobileSidebar}
          workspaceSlug={workspaceSlug}
          projectId={projectId}
          projectIdentifier={projectIdentifier}
          inboxIssue={inboxIssue}
        />
      ))}
    </>
  );
};
