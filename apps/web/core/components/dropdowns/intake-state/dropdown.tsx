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

export function IntakeStateDropdown(props: StateDropdownProps) {
  const { projectId, stateIds: propsStateIds } = props;
  // state query
  const stateQuery = useRouteProjectStates(projectId, true);
  // derived values
  const stateIds = propsStateIds ?? stateQuery.getProjectIntakeStateIds(projectId);

  return (
    <WorkItemStateDropdownBase
      {...props}
      getStateById={stateQuery.getIntakeStateById}
      isInitializing={stateQuery.intakeQuery.isPending}
      stateIds={stateIds ?? []}
    />
  );
}
