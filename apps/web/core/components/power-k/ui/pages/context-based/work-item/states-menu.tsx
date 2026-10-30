/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import { Command } from "cmdk";
// plane types

import type { TIssue } from "@plane/types";
import { Spinner } from "@plane/ui";
// hooks
import { useRouteProjectStates } from "@/hooks/use-project-states";
// local imports
import { PowerKProjectStatesMenuItems } from "./state-menu-item";

type Props = {
  handleSelect: (stateId: string) => void;
  workItemDetails: TIssue;
};

export function PowerKProjectStatesMenu(props: Props) {
  const { workItemDetails } = props;
  // router
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  // store hooks
  const stateQuery = useRouteProjectStates(workItemDetails.project_id);
  // derived values
  const filteredProjectStates = workItemDetails.project_id
    ? stateQuery.getProjectStates(workItemDetails.project_id)
    : undefined;

  if (!filteredProjectStates) return <Spinner />;

  return (
    <Command.Group>
      <PowerKProjectStatesMenuItems
        {...props}
        projectId={workItemDetails.project_id ?? undefined}
        selectedStateId={workItemDetails.state_id ?? undefined}
        states={filteredProjectStates}
        workspaceSlug={workspaceSlug}
      />
    </Command.Group>
  );
}
