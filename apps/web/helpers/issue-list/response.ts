/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors.
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import set from "lodash-es/set.js";
import { ALL_ISSUES } from "@plane/constants";
import type {
  TIssue,
  TIssuesResponse,
  TIssues,
  TGroupedIssueCount,
  TGroupedIssues,
  TSubGroupedIssues,
} from "@plane/types";
import { getGroupKey } from "./grouping";

/** Project an API response into entity rows, grouped IDs and counts without store state. */
export function processIssueResponse(issueResponse: TIssuesResponse): {
  issueList: TIssue[];
  groupedIssues: TIssues;
  groupedIssueCount: TGroupedIssueCount;
} {
  const issueResult = issueResponse?.results;

  // if undefined return empty objects
  if (!issueResult)
    return {
      issueList: [],
      groupedIssues: {},
      groupedIssueCount: {},
    };

  //if is an array then it's an ungrouped response. return values with groupId as ALL_ISSUES
  if (Array.isArray(issueResult)) {
    return {
      issueList: issueResult,
      groupedIssues: {
        [ALL_ISSUES]: issueResult.map((issue) => issue.id),
      },
      groupedIssueCount: {
        [ALL_ISSUES]: issueResponse.total_count,
      },
    };
  }

  const issueList: TIssue[] = [];
  const groupedIssues: TGroupedIssues | TSubGroupedIssues = {};
  const groupedIssueCount: TGroupedIssueCount = {};

  // update total issue count to ALL_ISSUES
  set(groupedIssueCount, [ALL_ISSUES], issueResponse.total_count);

  // loop through all the groupIds from issue Result
  for (const groupId in issueResult) {
    const groupIssuesObject = issueResult[groupId];
    const groupIssueResult = groupIssuesObject?.results;

    // if groupIssueResult is undefined then continue the loop
    if (!groupIssueResult) continue;

    // set grouped Issue count of the current groupId
    set(groupedIssueCount, [groupId], groupIssuesObject.total_results);

    // if groupIssueResult, the it is not subGrouped
    if (Array.isArray(groupIssueResult)) {
      // add the result to issueList
      issueList.push(...groupIssueResult);
      // set the issue Ids to the groupId path
      set(
        groupedIssues,
        [groupId],
        groupIssueResult.map((issue) => issue.id)
      );
      continue;
    }

    // loop through all the subGroupIds from issue Result
    for (const subGroupId in groupIssueResult) {
      const subGroupIssuesObject = groupIssueResult[subGroupId];
      const subGroupIssueResult = subGroupIssuesObject?.results;

      // if subGroupIssueResult is undefined then continue the loop
      if (!subGroupIssueResult) continue;

      // set sub grouped Issue count of the current groupId
      set(groupedIssueCount, [getGroupKey(groupId, subGroupId)], subGroupIssuesObject.total_results);

      if (Array.isArray(subGroupIssueResult)) {
        // add the result to issueList
        issueList.push(...subGroupIssueResult);
        // set the issue Ids to the [groupId, subGroupId] path
        set(
          groupedIssues,
          [groupId, subGroupId],
          subGroupIssueResult.map((issue) => issue.id)
        );

        continue;
      }
    }
  }

  return { issueList, groupedIssues, groupedIssueCount };
}
