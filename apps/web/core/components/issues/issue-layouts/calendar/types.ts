/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IIssueDisplayFilterOptions, TIssue, TIssueMap, TPaginationData } from "@plane/types";
import type { IssueListGroupedIds } from "@/queries/issue-lists";
import type { TRenderQuickActions } from "../list/list-view-types";

export type CalendarDayContentProps = {
  displayFilters: IIssueDisplayFilterOptions;
  issues: TIssueMap | undefined;
  groupedIssueIds: IssueListGroupedIds;
  handleDragAndDrop: (
    issueId: string | undefined,
    issueProjectId: string | undefined,
    sourceDate: string | undefined,
    destinationDate: string | undefined
  ) => Promise<void>;
  loadMoreIssues: (dateString: string) => void;
  getPaginationData: (groupId: string | undefined) => TPaginationData | undefined;
  getGroupIssueCount: (groupId: string | undefined) => number | undefined;
  enableQuickIssueCreate?: boolean;
  disableIssueCreation?: boolean;
  quickAddCallback?: (projectId: string | null | undefined, data: TIssue) => Promise<TIssue | undefined>;
  quickActions: TRenderQuickActions;
  addIssuesToView?: (issueIds: string[]) => Promise<unknown>;
  readOnly?: boolean;
  selectedDate: Date;
  setSelectedDate: (date: Date) => void;
  canEditProperties: (projectId: string | undefined) => boolean;
  isEpic?: boolean;
};
