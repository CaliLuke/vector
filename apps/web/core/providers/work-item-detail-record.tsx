/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createContext, useContext } from "react";
import type { ReactNode } from "react";
import type { TIssue, TIssueServiceType } from "@plane/types";

export type TWorkItemDetailRecord = {
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  serviceType: TIssueServiceType;
  issue: TIssue;
};

const WorkItemDetailRecordContext = createContext<TWorkItemDetailRecord | undefined>(undefined);

export function WorkItemDetailRecordProvider({
  value,
  children,
}: {
  value: TWorkItemDetailRecord;
  children: ReactNode;
}) {
  if (value.issue.id !== value.issueId || value.issue.project_id !== value.projectId)
    throw new Error("The detail record does not match its scoped route.");
  return <WorkItemDetailRecordContext.Provider value={value}>{children}</WorkItemDetailRecordContext.Provider>;
}

export function useWorkItemDetailRecord() {
  return useContext(WorkItemDetailRecordContext);
}
