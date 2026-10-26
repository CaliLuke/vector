/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCycleIssueLayout } from "../../use-cycle-issue-layout";
import { BaseKanBanRoot } from "../base-kanban-root";

export function CycleKanBanLayout() {
  const props = useCycleIssueLayout();
  return <BaseKanBanRoot {...props} />;
}
