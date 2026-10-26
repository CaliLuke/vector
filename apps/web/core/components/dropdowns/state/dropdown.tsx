/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// hooks
import { useRouteProjectStates } from "@/hooks/use-project-states";
// local imports
import type { StateDropdownProps } from "../state/types";
import { WorkItemStateDropdownBase } from "./base";

export function StateDropdown(props: StateDropdownProps) {
  const { projectId, stateIds: propsStateIds } = props;
  // state query
  const stateQuery = useRouteProjectStates(projectId);
  // derived values
  const stateIds = propsStateIds ?? stateQuery.getProjectStateIds(projectId);

  return (
    <WorkItemStateDropdownBase
      {...props}
      getStateById={stateQuery.getStateById}
      isInitializing={stateQuery.isPending}
      stateIds={stateIds ?? []}
    />
  );
}
