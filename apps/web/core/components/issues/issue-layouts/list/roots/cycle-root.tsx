/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCycleIssueLayout } from "../../use-cycle-issue-layout";
import { BaseListRoot } from "../base-list-root";

export function CycleListLayout() {
  const props = useCycleIssueLayout();
  return <BaseListRoot {...props} />;
}
