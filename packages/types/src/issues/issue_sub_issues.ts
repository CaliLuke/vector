/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TIssue } from "./issue";
import type { ISearchIssueResponse } from "../project/projects";

export type TSubIssuesStateDistribution = {
  backlog: string[];
  unstarted: string[];
  started: string[];
  completed: string[];
  cancelled: string[];
};

export type TIssueSubIssues = {
  state_distribution: TSubIssuesStateDistribution;
  sub_issues: TSubIssueResponse;
};

export type TSubIssueResponse = TIssue[] | { [key: string]: TIssue[] };

export type TIssueSubIssuesStateDistributionMap = {
  [issue_id: string]: TSubIssuesStateDistribution;
};

export type TIssueSubIssuesIdMap = {
  [issue_id: string]: string[];
};

export type TSubIssueOperations = {
  addSubIssue: (
    workspaceSlug: string,
    projectId: string,
    parentIssueId: string,
    issues: Pick<ISearchIssueResponse, "id" | "project_id">[]
  ) => Promise<void>;
  refreshSubIssues: (parentIssueId: string) => void;
  updateSubIssue: (
    workspaceSlug: string,
    projectId: string,
    parentIssueId: string,
    issueId: string,
    issueData: Partial<TIssue>
  ) => Promise<void>;
  removeSubIssue: (workspaceSlug: string, projectId: string, parentIssueId: string, issueId: string) => Promise<void>;
  deleteSubIssue: (workspaceSlug: string, projectId: string, parentIssueId: string, issueId: string) => Promise<void>;
};
