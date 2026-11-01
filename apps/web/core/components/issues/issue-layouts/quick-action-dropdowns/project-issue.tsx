/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// plane imports
import { EIssuesStoreType } from "@plane/types";
// hooks
// helper
import { IssueQuickActionControls } from "./issue-quick-action-modals";
import type { IQuickActionProps } from "../list/list-view-types";
import { useIssueMenuItems, useIssueMenuContext } from "./helper";
import { useIssueQuickActionModals } from "./use-issue-quick-action-modals";

export const ProjectIssueQuickActions = function ProjectIssueQuickActions(props: IQuickActionProps) {
  const { issue, placements = "bottom-end" } = props;
  // states
  const modals = useIssueQuickActionModals(issue);
  const menuItemProps = useIssueMenuContext(props, modals, EIssuesStoreType.PROJECT);

  const MENU_ITEMS = useIssueMenuItems(menuItemProps);

  return (
    <IssueQuickActionControls
      {...props}
      placements={placements}
      items={MENU_ITEMS}
      modals={modals}
      storeType={EIssuesStoreType.PROJECT}
    />
  );
};
