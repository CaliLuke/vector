/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { ISearchIssueResponse, TIssue } from "@plane/types";

export async function addSelectedIssuesToView(
  data: ISearchIssueResponse[],
  addIssuesToView?: (issueIds: string[]) => Promise<void | TIssue>
) {
  const issues = data.map((issue) => issue.id);
  try {
    await addIssuesToView?.(issues);
    setToast({
      type: TOAST_TYPE.SUCCESS,
      title: "Success!",
      message: "Work items added to the cycle successfully.",
    });
  } catch (_error) {
    setToast({
      type: TOAST_TYPE.ERROR,
      title: "Error!",
      message: "Selected work items could not be added to the cycle. Please try again.",
    });
  }
}
