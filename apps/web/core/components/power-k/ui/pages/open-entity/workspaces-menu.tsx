/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// plane types
import type { IWorkspace } from "@plane/types";
import { Spinner } from "@plane/ui";
// components
import { PowerKWorkspacesMenu } from "@/components/power-k/menus/workspaces";
// hooks
import { useWorkspaces } from "@/hooks/use-workspaces";

type Props = {
  handleSelect: (workspace: IWorkspace) => void;
};

export function PowerKOpenWorkspaceMenu(props: Props) {
  const { handleSelect } = props;
  const { data: workspaces = [], isPending } = useWorkspaces();
  // derived values
  const workspacesList = workspaces;

  if (isPending) return <Spinner />;

  return <PowerKWorkspacesMenu workspaces={workspacesList} onSelect={handleSelect} />;
}
