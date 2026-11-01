/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";

// plane imports
import { EIssuesStoreType } from "@plane/types";
// hooks
// helper
import { IssueQuickActionControls } from "./issue-quick-action-modals";
import type { IQuickActionProps } from "../list/list-view-types";
import { useCycleIssueMenuItems, useIssueMenuContext } from "./helper";
import { useIssueQuickActionModals } from "./use-issue-quick-action-modals";

export const CycleIssueQuickActions = function CycleIssueQuickActions(props: IQuickActionProps) {
  const { issue, placements = "bottom-start" } = props;
  // states
  const modals = useIssueQuickActionModals(issue);
  const { cycleId } = useParams({ strict: false });
  const menuItemProps = useIssueMenuContext(props, modals, EIssuesStoreType.CYCLE);

  const MENU_ITEMS = useCycleIssueMenuItems({ ...menuItemProps, cycleId: cycleId?.toString() });

  return (
    <IssueQuickActionControls
      {...props}
      placements={placements}
      items={MENU_ITEMS}
      modals={modals}
      storeType={EIssuesStoreType.CYCLE}
    />
  );
};
