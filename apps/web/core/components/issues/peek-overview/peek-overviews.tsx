/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { IssuePeekOverview } from "@/components/issues/peek-overview";
import { useIssueDetailUiState } from "@/providers/issue-detail-ui-state";

export const HomePeekOverviewsRoot = function HomePeekOverviewsRoot() {
  const { peekIssue } = useIssueDetailUiState();

  return peekIssue ? <IssuePeekOverview /> : null;
};
