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
import { useModuleIssueMenuItems, useIssueMenuContext } from "./helper";
import { useIssueQuickActionModals } from "./use-issue-quick-action-modals";

export const ModuleIssueQuickActions = function ModuleIssueQuickActions(props: IQuickActionProps) {
  const { issue, placements = "bottom-start" } = props;
  // states
  const modals = useIssueQuickActionModals(issue);
  const { moduleId } = useParams({ strict: false });
  const menuItemProps = useIssueMenuContext(props, modals, EIssuesStoreType.MODULE);

  const MENU_ITEMS = useModuleIssueMenuItems({ ...menuItemProps, moduleId: moduleId?.toString() });

  return (
    <IssueQuickActionControls
      {...props}
      placements={placements}
      items={MENU_ITEMS}
      modals={modals}
      storeType={EIssuesStoreType.MODULE}
    />
  );
};
