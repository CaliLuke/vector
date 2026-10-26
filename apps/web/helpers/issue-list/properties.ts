/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TIssue, TIssueGroupByOptions } from "@plane/types";

export type TIssueDisplayFilterOptions = Exclude<TIssueGroupByOptions, null> | "target_date";

// This constant maps the group by keys to the respective issue property that the key relies on
export const ISSUE_GROUP_BY_KEY: Record<TIssueDisplayFilterOptions, keyof TIssue> = {
  type_id: "type_id",
  project: "project_id",
  state: "state_id",
  "state_detail.group": "state_id", // state_detail.group is only being used for state_group display,
  priority: "priority",
  labels: "label_ids",
  created_by: "created_by",
  assignees: "assignee_ids",
  target_date: "target_date",
  cycle: "cycle_id",
  module: "module_ids",
  team_project: "project_id",
};

export const ISSUE_FILTER_DEFAULT_DATA: Record<TIssueDisplayFilterOptions, keyof TIssue> = {
  type_id: "type_id",
  project: "project_id",
  cycle: "cycle_id",
  module: "module_ids",
  state: "state_id",
  "state_detail.group": "state__group", // state_detail.group is only being used for state_group display,
  priority: "priority",
  labels: "label_ids",
  created_by: "created_by",
  assignees: "assignee_ids",
  target_date: "target_date",
  team_project: "project_id",
};
